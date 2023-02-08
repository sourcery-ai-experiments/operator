package managed

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"github.com/kloudlite/operator/pkg/constants"
	"github.com/kloudlite/operator/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kloudlite/operator/apis/cluster-setup/v1"
	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	lc "github.com/kloudlite/operator/operators/cluster-setup/internal/constants"
	"github.com/kloudlite/operator/operators/cluster-setup/internal/env"
	"github.com/kloudlite/operator/operators/cluster-setup/internal/templates"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-playground/validator/v10"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	logger     logging.Logger
	Name       string
	yamlClient *kubectl.YAMLClient
	restConfig *rest.Config
	Env        *env.Env
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	InternalOperatorInstalled string = "internal-operator-installed"
	DefaultsPatched           string = "defaults-patched"
	KloudliteCredsValidated   string = "kloudlite-creds-validated"
	UserKubeConfigCreated     string = "user-kubeconfig-created"
)

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.logger), r.Client, request.NamespacedName, &v1.ManagedCluster{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	req.LogPreReconcile()
	defer req.LogPostReconcile()

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	//if step := req.EnsureChecks(KloudliteCredsValidated); !step.ShouldProceed() {
	//	return step.ReconcilerResponse()
	//}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.checkKloudliteCreds(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.patchDefaults(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureInternalOperator(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureUserKubeConfig(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// 1. kloudlite CRDs
	// 2. Internal Operator (Env and controller)

	req.Object.Status.IsReady = true
	req.Object.Status.LastReconcileTime = metav1.Time{Time: time.Now()}

	return ctrl.Result{}, nil
}

func (r *Reconciler) finalize(req *rApi.Request[*v1.ManagedCluster]) stepResult.Result {
	return req.Finalize()
}

func (r *Reconciler) checkKloudliteCreds(req *rApi.Request[*v1.ManagedCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	scrt, err := rApi.Get(ctx, r.Client, fn.NN(obj.Spec.KloudliteCreds.Namespace, obj.Spec.KloudliteCreds.Name), &corev1.Secret{})
	if err != nil {
		return req.CheckFailed(KloudliteCredsValidated, check, err.Error()).Err(nil)
	}

	klCreds, err := fn.ParseFromSecret[v1.KloudliteCreds](scrt)
	if err != nil {
		return req.CheckFailed(KloudliteCredsValidated, check, err.Error()).Err(nil)
	}
	if err := validator.New().Struct(klCreds); err != nil {
		return req.CheckFailed(KloudliteCredsValidated, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[KloudliteCredsValidated] {
		checks[KloudliteCredsValidated] = check
		req.UpdateStatus()
	}

	rApi.SetLocal(req, "kl-creds", klCreds)

	return req.Next()
}

func (r *Reconciler) patchDefaults(req *rApi.Request[*v1.ManagedCluster]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(DefaultsPatched)
	defer req.LogPostCheck(DefaultsPatched)

	hasUpdated := false

	if obj.Spec.Domain == nil {
		hasUpdated = true
		obj.Spec.Domain = fn.New(fmt.Sprintf("%s.clusters.kloudlite.io", obj.Name))
	}

	if hasUpdated {
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(DefaultsPatched, check, err.Error())
		}

		return req.Done().RequeueAfter(100 * time.Millisecond)
	}

	check.Status = true
	if check != obj.Status.Checks[DefaultsPatched] {
		obj.Status.Checks[DefaultsPatched] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) ensureInternalOperator(req *rApi.Request[*v1.ManagedCluster]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(InternalOperatorInstalled)
	defer req.LogPostCheck(InternalOperatorInstalled)

	klCreds, ok := rApi.GetLocal[*v1.KloudliteCreds](req, "kl-creds")
	if !ok {
		return req.CheckFailed(KloudliteCredsValidated, check, errors.NotInLocals("kl-creds").Error()).Err(nil)
	}

	b, err := templates.Parse(templates.InternalOperatorEnv, map[string]any{
		"namespace":           lc.NsOperators,
		"cluster-id":          obj.Name,
		"wildcard-domain":     obj.Spec.Domain,
		"nameserver-endpoint": fmt.Sprintf("https://%s", klCreds.DnsApiEndpoint),
		"nameserver-username": klCreds.DnsApiUsername,
		"nameserver-password": klCreds.DnsApiPassword,
	})

	if err != nil {
		return req.CheckFailed(InternalOperatorInstalled, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(InternalOperatorInstalled, check, err.Error()).Err(nil)
	}

	b2, err := os.ReadFile("/tmp/res/crds.yml")
	if err != nil {
		return req.CheckFailed(KloudliteCredsValidated, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b2); err != nil {
		return req.CheckFailed(InternalOperatorInstalled, check, err.Error()).Err(nil)
	}

	b3, err := os.ReadFile("/tmp/res/operator.yml.tpl")
	if err != nil {
		return req.CheckFailed(KloudliteCredsValidated, check, err.Error()).Err(nil)
	}

	b4, err := templates.ParseBytes(b3, map[string]any{
		"Namespace":       lc.NsOperators,
		"SvcAccountName":  lc.ClusterSvcAccount,
		"ImagePullPolicy": "Always",
		"EnvName":         "development",
		"ImageTag":        "v1.0.5",
	})

	if err != nil {
		return req.CheckFailed(InternalOperatorInstalled, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b4); err != nil {
		return req.CheckFailed(InternalOperatorInstalled, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != obj.Status.Checks[InternalOperatorInstalled] {
		obj.Status.Checks[InternalOperatorInstalled] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) ensureUserKubeConfig(req *rApi.Request[*v1.ManagedCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	svcAccountName := obj.Name + "-admin"
	svcAccountNs := "kube-system"

	// 1. create service account for user
	// 2. create cluster role for this user
	// 3. create service account token for that user
	b, err := templates.Parse(templates.UserAccountRbac, map[string]any{
		"svc-account-name":      svcAccountName,
		"svc-account-namespace": svcAccountNs,
	})

	if err != nil {
		return req.CheckFailed(UserKubeConfigCreated, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(UserKubeConfigCreated, check, err.Error()).Err(nil)
	}

	// 4. then read that service account secret for `.data.token` field
	s, err := rApi.Get(ctx, r.Client, fn.NN(svcAccountNs, svcAccountName), &corev1.Secret{})
	if err != nil {
		return req.CheckFailed(UserKubeConfigCreated, check, err.Error()).Err(nil)
	}

	b64CaCrt := strings.TrimSpace(base64.StdEncoding.EncodeToString(s.Data["ca.crt"]))

	b, err = templates.Parse(templates.Kubeconfig, map[string]any{
		"ca-data":    b64CaCrt,
		"user-token": string(s.Data["token"]),

		"cluster-endpoint": fmt.Sprintf("https://k8s.%s:6443", *obj.Spec.Domain),
		"user-name":        svcAccountName,
	})

	if err != nil {
		return req.CheckFailed(UserKubeConfigCreated, check, err.Error()).Err(nil)
	}

	kubeConfig := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: svcAccountName + "-kubeconfig", Namespace: "kube-system"}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, kubeConfig, func() error {
		if kubeConfig.Data == nil {
			kubeConfig.Data = make(map[string][]byte, 1)
		}
		kubeConfig.Data["kubeconfig"] = b
		return nil
	}); err != nil {
		return req.CheckFailed(UserKubeConfigCreated, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[UserKubeConfigCreated] {
		checks[UserKubeConfigCreated] = check
		req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig())
	r.restConfig = mgr.GetConfig()

	builder := ctrl.NewControllerManagedBy(mgr).For(&v1.ManagedCluster{})
	builder.Owns(&crdsv1.App{})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}