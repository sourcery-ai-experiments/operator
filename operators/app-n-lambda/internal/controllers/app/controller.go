package app

import (
	"context"
	"encoding/json"
	"fmt"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/client-go/tools/record"

	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/app-n-lambda/internal/env"
	"github.com/kloudlite/operator/pkg/conditions"
	"github.com/kloudlite/operator/pkg/constants"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
	"github.com/kloudlite/operator/pkg/templates"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Logger     logging.Logger
	Name       string
	Env        *env.Env
	YamlClient *kubectl.YAMLClient
	recorder   record.EventRecorder
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	DeploymentSvcAndHpaCreated string = "deployment-svc-and-hpa-created"
	ImagesLabelled             string = "images-labelled"
	DeploymentReady            string = "deployment-ready"
	AnchorReady                string = "anchor-ready"
)

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.Logger), r.Client, request.NamespacedName, &crdsv1.App{})
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

	// if crdsv1.IsBlueprintNamespace(ctx, r.Client, request.Namespace) {
	// 	return ctrl.Result{}, nil
	// }

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.RestartIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// if req.Object.Enabled != nil && !*req.Object.Enabled {
	// anchor := &crdsv1.Anchor{ObjectMeta: metav1.ObjectMeta{Name: req.GetAnchorName(), Namespace: req.Object.Namespace}}
	// return ctrl.Result{}, client.IgnoreNotFound(r.Delete(ctx, anchor))
	// }

	// if step := operator.EnsureAnchor(req); !step.ShouldProceed() {
	// 	return step.ReconcilerResponse()
	// }
	//

	if step := r.reconLabellingImages(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureDeploymentThings(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.checkDeploymentReady(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	req.Object.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}

	//req.Object.Status.DisplayVars.Set("intercepted", func() string {
	//	if req.Object.GetLabels()[constants.LabelKeys.IsIntercepted] == "true" {
	//		return "true/" + req.Object.GetLabels()[constants.LabelKeys.DeviceRef]
	//	}
	//	return "false"
	//}())
	//req.Object.Status.DisplayVars.Set("frozen", req.Object.GetLabels()[constants.LabelKeys.Freeze] == "true")

	req.Object.Status.Resources = req.GetOwnedResources()
	if err := r.Status().Update(ctx, req.Object); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: r.Env.ReconcilePeriod}, nil
}

func (r *Reconciler) cleanupLogic(req *rApi.Request[*crdsv1.App]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	checkName := "cleanupLogic"

	resources := req.Object.Status.Resources

	for i := range resources {
		res := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": resources[i].APIVersion,
			"kind":       resources[i].Kind,
			"metadata": map[string]any{
				"name":      resources[i].Name,
				"namespace": resources[i].Namespace,
			},
		}}

		if err := r.Get(ctx, client.ObjectKeyFromObject(res), res); err != nil {
			if !apiErrors.IsNotFound(err) {
				return req.CheckFailed("CleanupResource", check, err.Error()).Err(nil)
			}
			return req.CheckFailed("CleanupResource", check,
				fmt.Sprintf("waiting for resource gvk=%s, nn=%s", res.GetObjectKind().GroupVersionKind().String(), fn.NN(res.GetNamespace(), res.GetName())),
			).Err(nil)
		}

		if res.GetDeletionTimestamp() == nil {
			if err := r.Delete(ctx, res); err != nil {
				return req.CheckFailed("CleanupResource", check, err.Error()).Err(nil)
			}
		}
	}

	check.Status = true
	if check != obj.Status.Checks[checkName] {
		obj.Status.Checks[checkName] = check
		req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) finalize(req *rApi.Request[*crdsv1.App]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	checkName := "finalizing"

	if step := r.cleanupLogic(req); !step.ShouldProceed() {
		return step
	}

	controllerutil.RemoveFinalizer(obj, constants.ForegroundFinalizer)
	controllerutil.RemoveFinalizer(obj, constants.CommonFinalizer)

	if err := r.Update(ctx, obj); err != nil {
		return req.CheckFailed(checkName, check, err.Error()).Err(nil)
	}

	return req.Next()
}

func (r *Reconciler) reconLabellingImages(req *rApi.Request[*crdsv1.App]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(ImagesLabelled)
	defer req.LogPostCheck(ImagesLabelled)

	newLabels := make(map[string]string, len(obj.GetLabels()))
	for s, v := range obj.GetLabels() {
		newLabels[s] = v
	}

	for s := range newLabels {
		if strings.HasPrefix(s, "kloudlite.io/image-") {
			delete(newLabels, s)
		}
	}

	for i := range obj.Spec.Containers {
		newLabels[fmt.Sprintf("kloudlite.io/image-%s", fn.Sha1Sum([]byte(obj.Spec.Containers[i].Image)))] = "true"
	}

	if !reflect.DeepEqual(newLabels, obj.GetLabels()) {
		obj.SetLabels(newLabels)
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(ImagesLabelled, check, err.Error())
		}
		return req.Done().RequeueAfter(200 * time.Millisecond)
	}

	check.Status = true
	if check != obj.Status.Checks[ImagesLabelled] {
		obj.Status.Checks[ImagesLabelled] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) ensureDeploymentThings(req *rApi.Request[*crdsv1.App]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(DeploymentSvcAndHpaCreated)
	defer req.LogPostCheck(DeploymentSvcAndHpaCreated)

	volumes, vMounts := crdsv1.ParseVolumes(obj.Spec.Containers)

	//isIntercepted := obj.GetLabels()[constants.LabelKeys.IsIntercepted] == "true"
	//isFrozen := obj.GetLabels()[constants.LabelKeys.Freeze] == "true"

	b, err := templates.Parse(
		templates.CrdsV1.App, map[string]any{
			"object":        obj,
			"volumes":       volumes,
			"volume-mounts": vMounts,
			"owner-refs":    []metav1.OwnerReference{fn.AsOwner(obj, true)},

			// for intercepting
			//"freeze":        isFrozen || isIntercepted,
			//"is-intercepted": obj.GetLabels()[constants.LabelKeys.IsIntercepted] == "true",
			//"device-ref":     obj.GetLabels()[constants.LabelKeys.DeviceRef],
			//"account-ref":    obj.Spec.AccountName,
		},
	)

	if err != nil {
		return req.CheckFailed(DeploymentSvcAndHpaCreated, check, err.Error()).Err(nil)
	}

	resRefs, err := r.YamlClient.ApplyYAML(ctx, b)
	if err != nil {
		return req.CheckFailed(DeploymentSvcAndHpaCreated, check, err.Error()).Err(nil)
	}

	req.AddToOwnedResources(resRefs...)
	req.UpdateStatus()

	fmt.Printf("resRefs: %+v\n", resRefs)
	fmt.Printf("obj.Status.Resources: %+v\n", obj.Status.Resources)

	check.Status = true
	if check != obj.Status.Checks[DeploymentSvcAndHpaCreated] {
		obj.Status.Checks[DeploymentSvcAndHpaCreated] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) checkDeploymentReady(req *rApi.Request[*crdsv1.App]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(DeploymentReady)
	defer req.LogPostCheck(DeploymentReady)

	deployment, err := rApi.Get(ctx, r.Client, fn.NN(obj.Namespace, obj.Name), &appsv1.Deployment{})
	if err != nil {
		return req.CheckFailed(DeploymentReady, check, err.Error()).Err(nil)
	}

	cds, err := conditions.FromObject(deployment)
	if err != nil {
		return req.CheckFailed(DeploymentReady, check, err.Error()).Err(nil)
	}

	isReady := meta.IsStatusConditionTrue(cds, "Available")

	if !isReady {
		var podList corev1.PodList
		if err := r.List(
			ctx, &podList, &client.ListOptions{
				LabelSelector: labels.SelectorFromValidatedSet(map[string]string{"app": obj.Name}),
				Namespace:     obj.Namespace,
			},
		); err != nil {
			return req.CheckFailed(DeploymentReady, check, err.Error())
		}

		pMessages := rApi.GetMessagesFromPods(podList.Items...)
		bMsg, err := json.Marshal(pMessages)
		if err != nil {
			check.Message = err.Error()
			return req.CheckFailed(DeploymentReady, check, err.Error())
		}
		check.Message = string(bMsg)
		return req.CheckFailed(DeploymentReady, check, "deployment is not ready")
	}

	if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
		return req.CheckFailed(
			DeploymentReady,
			check,
			fmt.Sprintf("ready-replicas (%d) != total replicas (%d)", deployment.Status.ReadyReplicas, deployment.Status.Replicas),
		)
	}

	check.Status = true
	if check != obj.Status.Checks[DeploymentReady] {
		obj.Status.Checks[DeploymentReady] = check
		req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.Logger = logger.WithName(r.Name)
	r.YamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig())
	r.recorder = mgr.GetEventRecorderFor(r.GetName())

	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.App{})
	builder.Owns(&crdsv1.Anchor{})

	watchList := []client.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
		&autoscalingv2.HorizontalPodAutoscaler{},
	}

	for i := range watchList {
		builder.Watches(&source.Kind{Type: watchList[i]}, handler.EnqueueRequestsFromMapFunc(
			func(obj client.Object) []reconcile.Request {
				if v, ok := obj.GetLabels()[constants.AppNameKey]; ok {
					return []reconcile.Request{{NamespacedName: fn.NN(obj.GetNamespace(), v)}}
				}
				return nil
			}))
	}
	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
