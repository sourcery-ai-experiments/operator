package mres

import (
	"context"
	"encoding/json"
	"operators.kloudlite.io/operator"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	crdsv1 "operators.kloudlite.io/apis/crds/v1"
	influxdbMsvcv1 "operators.kloudlite.io/apis/influxdb.msvc/v1"
	mongodbMsvcv1 "operators.kloudlite.io/apis/mongodb.msvc/v1"
	mysqlMsvcv1 "operators.kloudlite.io/apis/mysql.msvc/v1"
	redisMsvcv1 "operators.kloudlite.io/apis/redis.msvc/v1"
	"operators.kloudlite.io/operators/msvc-n-mres/internal/env"
	"operators.kloudlite.io/pkg/constants"
	fn "operators.kloudlite.io/pkg/functions"
	"operators.kloudlite.io/pkg/harbor"
	"operators.kloudlite.io/pkg/kubectl"
	"operators.kloudlite.io/pkg/logging"
	rApi "operators.kloudlite.io/pkg/operator"
	stepResult "operators.kloudlite.io/pkg/operator/step-result"
	"operators.kloudlite.io/pkg/templates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ManagedResourceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	harborCli  *harbor.Client
	logger     logging.Logger
	Name       string
	Env        *env.Env
	yamlClient *kubectl.YAMLClient
}

func (r *ManagedResourceReconciler) GetName() string {
	return r.Name
}

const (
	RealMresCreated string = "real-mres-created"
	RealMresReady   string = "real-mres-ready"
	MsvcIsOwner     string = "msvc-is-owner"
)

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds/finalizers,verbs=update

func (r *ManagedResourceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.logger), r.Client, request.NamespacedName, &crdsv1.ManagedResource{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	req.LogPreReconcile()
	defer req.LogPostReconcile()

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	if strings.HasSuffix(request.Namespace, "-blueprint") {
		return ctrl.Result{}, nil
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureChecks(RealMresReady); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if req.Object.Enabled != nil && !*req.Object.Enabled {
		anchor := &crdsv1.Anchor{ObjectMeta: metav1.ObjectMeta{Name: req.GetAnchorName(), Namespace: req.Object.GetNamespace()}}
		return ctrl.Result{}, client.IgnoreNotFound(r.Delete(ctx, anchor))
	}

	if step := operator.EnsureAnchor(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.reconOwnership(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureRealMresCreated(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureRealMresReady(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	req.Object.Status.LastReconcileTime = metav1.Time{Time: time.Now()}
	return ctrl.Result{RequeueAfter: r.Env.ReconcilePeriod}, r.Status().Update(ctx, req.Object)
}

func (r *ManagedResourceReconciler) finalize(req *rApi.Request[*crdsv1.ManagedResource]) stepResult.Result {
	return req.Finalize()
}

func (r *ManagedResourceReconciler) reconOwnership(req *rApi.Request[*crdsv1.ManagedResource]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	msvc, err := rApi.Get(
		ctx, r.Client, fn.NN(obj.Namespace, obj.Spec.MsvcRef.Name), &crdsv1.ManagedService{},
	)

	if err != nil {
		return req.CheckFailed(MsvcIsOwner, check, err.Error())
	}

	if !fn.IsOwner(obj, fn.AsOwner(msvc)) {
		obj.SetOwnerReferences(append(obj.GetOwnerReferences(), fn.AsOwner(msvc)))
		if err := r.Update(ctx, obj); err != nil {
			return req.FailWithOpError(err)
		}
		return req.Done()
	}

	check.Status = true
	if check != checks[MsvcIsOwner] {
		checks[MsvcIsOwner] = check
		return req.UpdateStatus()
	}

	return req.Next()
}

func (r *ManagedResourceReconciler) ensureRealMresCreated(req *rApi.Request[*crdsv1.ManagedResource]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(RealMresCreated)
	defer req.LogPostCheck(RealMresCreated)

	anchor, _ := rApi.GetLocal[*crdsv1.Anchor](req, "anchor")
	mresBytes, err := templates.Parse(
		templates.CommonMres, map[string]any{
			"object":     obj,
			"owner-refs": []metav1.OwnerReference{fn.AsOwner(anchor, true)},
		},
	)

	if err != nil {
		return req.CheckFailed(RealMresCreated, check, err.Error()).Err(nil)
	}
	if err := r.yamlClient.ApplyYAML(ctx, mresBytes); err != nil {
		return req.CheckFailed(RealMresCreated, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[RealMresCreated] {
		checks[RealMresCreated] = check
		return req.UpdateStatus()
	}

	return req.Next()
}

func (r *ManagedResourceReconciler) ensureRealMresReady(req *rApi.Request[*crdsv1.ManagedResource]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(RealMresReady)
	defer req.LogPreCheck(RealMresReady)

	realMres, err := rApi.Get(
		ctx, r.Client, fn.NN(obj.Namespace, obj.Name), fn.NewUnstructured(metav1.TypeMeta{APIVersion: obj.Spec.MsvcRef.APIVersion, Kind: obj.Spec.MresKind.Kind}),
	)
	if err != nil {
		req.Logger.Infof("real managed resource (%s) does not exist, creating it now...", fn.NN(obj.Namespace, obj.Name).String())
	}

	b, err := json.Marshal(realMres)
	if err != nil {
		return req.CheckFailed(RealMresReady, check, err.Error()).Err(nil)
	}

	var realMresObj struct {
		Status rApi.Status `json:"status"`
	}

	if err := json.Unmarshal(b, &realMresObj); err != nil {
		return req.CheckFailed(RealMresReady, check, err.Error()).Err(nil)
	}

	if !realMresObj.Status.IsReady {
		if realMresObj.Status.Message.RawMessage == nil {
			return req.CheckFailed(RealMresReady, check, "waiting for real managed resource to reconcile ...")
		}
		b, err := json.Marshal(realMresObj.Status.Message)
		if err != nil {
			return req.CheckFailed(RealMresReady, check, err.Error()).Err(nil)
		}
		return req.CheckFailed(RealMresReady, check, string(b)).Err(nil)
	}

	check.Status = true
	if check != checks[RealMresReady] {
		checks[RealMresReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *ManagedResourceReconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig())

	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.ManagedResource{})
	builder.Owns(&corev1.Secret{})
	builder.Owns(&crdsv1.Anchor{})

	children := []client.Object{
		&redisMsvcv1.StandaloneService{},
		&redisMsvcv1.ClusterService{},
		&redisMsvcv1.ACLAccount{},

		&mongodbMsvcv1.StandaloneService{},
		&mongodbMsvcv1.ClusterService{},
		&mongodbMsvcv1.Database{},

		&mysqlMsvcv1.ClusterService{},
		&mysqlMsvcv1.StandaloneService{},
		&mysqlMsvcv1.Database{},

		&influxdbMsvcv1.Bucket{},
		&influxdbMsvcv1.Service{},
	}

	for i := range children {
		builder.Watches(
			&source.Kind{Type: children[i]},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				if v, ok := obj.GetLabels()[constants.MresNameKey]; ok {
					return []reconcile.Request{{NamespacedName: fn.NN(obj.GetNamespace(), v)}}
				}
				return nil
			}))
	}
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
