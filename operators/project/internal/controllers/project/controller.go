package project

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/project/internal/env"
	"github.com/kloudlite/operator/pkg/constants"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
	"github.com/kloudlite/operator/pkg/templates"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	logger     logging.Logger
	Name       string
	Env        *env.Env
	yamlClient kubectl.YAMLClient
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	NamespacedRBACsReady string = "namespaced-rbacs-ready"
	NamespaceExists      string = "namespace-exists"

	ProjectFinalizing string = "project-finalizing"
)

var (
	ApplyChecklist = []rApi.CheckMeta{
		{Name: NamespaceExists, Title: "ensure namespace exists"},
		{Name: NamespacedRBACsReady, Title: "ensure namespaced rbacs are ready"},
	}
	DestroyChecklist = []rApi.CheckMeta{
		{Name: ProjectFinalizing, Title: "finalizing project"},
	}
)

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=projects/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.logger), r.Client, request.NamespacedName, &crdsv1.Project{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	req.PreReconcile()
	defer req.PostReconcile()

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.CommonFinalizer, constants.ForegroundFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureCheckList(ApplyChecklist); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureChecks(NamespacedRBACsReady, NamespaceExists); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureNamespace(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureNamespacedRBACs(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func findResourceBelongingToProject(ctx context.Context, kclient client.Client, resources client.ObjectList, projectTargetNamespace string) error {
	if err := kclient.List(ctx, resources, &client.ListOptions{
		Namespace: projectTargetNamespace,
	}); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) finalize(req *rApi.Request[*crdsv1.Project]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.NewRunningCheck(ProjectFinalizing, req)

	if step := req.EnsureCheckList(DestroyChecklist); !step.ShouldProceed() {
		return step
	}

	// ensure deletion of other kloudlite resources, that belong to this project
	var envList crdsv1.EnvironmentList
	if err := findResourceBelongingToProject(ctx, r.Client, &envList, obj.Spec.TargetNamespace); err != nil {
		return check.StillRunning(err)
	}

	envs := make([]client.Object, len(envList.Items))
	for i := range envList.Items {
		envs[i] = &envList.Items[i]
	}

	if err := fn.DeleteAndWait(ctx, r.logger, r.Client, envs...); err != nil {
		return check.StillRunning(err).RequeueAfter(500 * time.Millisecond)
	}

	var projectMsvcList crdsv1.ProjectManagedServiceList
	if err := findResourceBelongingToProject(ctx, r.Client, &projectMsvcList, obj.Spec.TargetNamespace); err != nil {
		return check.StillRunning(err).RequeueAfter(500 * time.Millisecond)
	}

	projectMsvcs := make([]client.Object, len(projectMsvcList.Items))
	for i := range projectMsvcList.Items {
		projectMsvcs[i] = &projectMsvcList.Items[i]
	}

	if err := fn.DeleteAndWait(ctx, r.logger, r.Client, projectMsvcs...); err != nil {
		return check.StillRunning(err).RequeueAfter(500 * time.Millisecond)
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: obj.Spec.TargetNamespace}}
	if err := fn.DeleteAndWait(ctx, r.logger, r.Client, ns); err != nil {
		return check.StillRunning(err).RequeueAfter(500 * time.Millisecond)
	}

	return check.Completed()
}

func (r *Reconciler) ensureNamespace(req *rApi.Request[*crdsv1.Project]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.NewRunningCheck(NamespaceExists, req)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: obj.Spec.TargetNamespace}}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string, 3)
		}

		ns.Labels[constants.ProjectNameKey] = obj.Name

		if ns.Annotations == nil {
			ns.Annotations = make(map[string]string, 1)
		}

		ns.Annotations[constants.DescriptionKey] = "this namespace is now being managed by kloudlite project"
		return nil
	}); err != nil {
		return check.StillRunning(err)
	}

	return check.Completed()
}

func (r *Reconciler) ensureNamespacedRBACs(req *rApi.Request[*crdsv1.Project]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.NewRunningCheck(NamespacedRBACsReady, req)

	var pullSecrets corev1.SecretList
	if err := r.List(ctx, &pullSecrets, client.InNamespace(obj.Spec.TargetNamespace)); err != nil {
		return check.StillRunning(err)
	}

	secretNames := make([]string, 0, len(pullSecrets.Items))
	for i := range pullSecrets.Items {
		if pullSecrets.Items[i].Type == corev1.SecretTypeDockerConfigJson {
			secretNames = append(secretNames, pullSecrets.Items[i].Name)
		}
	}

	b, err := templates.Parse(
		templates.ProjectRBAC, map[string]any{
			"namespace":          obj.Spec.TargetNamespace,
			"svc-account-name":   r.Env.SvcAccountName,
			"image-pull-secrets": secretNames,
			"owner-refs":         []metav1.OwnerReference{fn.AsOwner(obj, true)},
		},
	)
	if err != nil {
		return check.Failed(err).Err(nil)
	}

	refs, err := r.yamlClient.ApplyYAML(ctx, b)
	if err != nil {
		return check.StillRunning(err)
	}

	req.AddToOwnedResources(refs...)

	return check.Completed()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig(), kubectl.YAMLClientOpts{Logger: r.logger})

	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.Project{})
	builder.Owns(&corev1.ServiceAccount{})
	builder.Owns(&rbacv1.Role{})
	builder.Owns(&rbacv1.RoleBinding{})
	builder.Owns(&appsv1.Deployment{})
	builder.Owns(&crdsv1.App{})

	builder.Watches(
		&corev1.Namespace{},
		handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
			if v, ok := obj.GetLabels()[constants.ProjectNameKey]; ok {
				return []reconcile.Request{{NamespacedName: fn.NN("", v)}}
			}
			return nil
		}),
	)

	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
