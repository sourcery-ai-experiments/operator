package mysqlstandalonemsvc

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	mysqlStandalone "operators.kloudlite.io/apis/mysql-standalone.msvc/v1"
	"operators.kloudlite.io/controllers/crds"
	"operators.kloudlite.io/lib/constants"
	"operators.kloudlite.io/lib/errors"
	"operators.kloudlite.io/lib/finalizers"
	fn "operators.kloudlite.io/lib/functions"
	reconcileResult "operators.kloudlite.io/lib/reconcile-result"
	"operators.kloudlite.io/lib/templates"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	lt     metav1.Time
}

type Output struct {
	RootPassword string `json:"ROOT_PASSWORD"`
	DbHosts      string `json:"HOSTS"`
}

const (
	MysqlRootPasswordKey = "mysql-root-password"
	MysqlPasswordKey     = "mysql-password"
)

type ServiceReconReq struct {
	ctrl.Request
	logger    *zap.SugaredLogger
	mysqlSvc  *mysqlStandalone.Service
	stateData map[string]string
}

// +kubebuilder:rbac:groups=mysql-standalone.msvc.kloudlite.io,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql-standalone.msvc.kloudlite.io,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql-standalone.msvc.kloudlite.io,resources=services/finalizers,verbs=update

func (r *ServiceReconciler) Reconcile(ctx context.Context, orgReq ctrl.Request) (ctrl.Result, error) {
	req := &ServiceReconReq{
		Request:   orgReq,
		logger:    crds.GetLogger(orgReq.NamespacedName),
		mysqlSvc:  new(mysqlStandalone.Service),
		stateData: map[string]string{},
	}

	if err := r.Get(ctx, orgReq.NamespacedName, req.mysqlSvc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !req.mysqlSvc.HasLabels() {
		req.mysqlSvc.EnsureLabels()
		if err := r.Update(ctx, req.mysqlSvc); err != nil {
			return reconcileResult.FailedE(err)
		}
		return reconcileResult.OK()
	}

	if req.mysqlSvc.GetDeletionTimestamp() != nil {
		return r.finalize(ctx, req.mysqlSvc)
	}

	reconResult, err := r.reconcileStatus(ctx, req)
	if err != nil {
		return r.failWithErr(ctx, req, err)
	}
	if reconResult != nil {
		return *reconResult, nil
	}

	req.logger.Infof("status is in sync, so proceeding with ops")
	return r.reconcileOperations(ctx, req)
}

func (r *ServiceReconciler) finalize(ctx context.Context, m *mysqlStandalone.Service) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(m, finalizers.MsvcCommonService.String()) {
		if err := r.Delete(
			ctx, &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": constants.MsvcApiVersion,
					"kind":       constants.HelmMySqlDBKind,
					"metadata": map[string]interface{}{
						"name":      m.Name,
						"namespace": m.Namespace,
					},
				},
			},
		); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(m, finalizers.MsvcCommonService.String())
		if err := r.Update(ctx, m); err != nil {
			return ctrl.Result{}, err
		}
		return reconcileResult.OK()
	}
	return reconcileResult.OK()
}

func (r *ServiceReconciler) failWithErr(ctx context.Context, req *ServiceReconReq, err error) (ctrl.Result, error) {
	fn.Conditions2.MarkNotReady(&req.mysqlSvc.Status.OpsConditions, err, "ReconFailedWithErr")
	if err2 := r.Status().Update(ctx, req.mysqlSvc); err2 != nil {
		return ctrl.Result{}, err2
	}
	return ctrl.Result{}, err
}

func (r *ServiceReconciler) reconcileStatus(ctx context.Context, req *ServiceReconReq) (*ctrl.Result, error) {
	var conditions []metav1.Condition

	err := fn.Conditions2.BuildFromHelmMsvc(
		&conditions,
		ctx,
		r.Client,
		constants.HelmMySqlDBKind,
		types.NamespacedName{
			Namespace: req.mysqlSvc.GetNamespace(),
			Name:      req.mysqlSvc.GetName(),
		},
	)

	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, err
		}
	}

	err = fn.Conditions2.BuildFromStatefulset(
		&conditions,
		ctx,
		r.Client,
		types.NamespacedName{
			Namespace: req.mysqlSvc.GetNamespace(),
			Name:      req.mysqlSvc.GetName(),
		},
	)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, err
		}
	}

	helmSecret := new(corev1.Secret)
	nn := types.NamespacedName{
		Namespace: req.mysqlSvc.GetNamespace(),
		Name:      req.mysqlSvc.GetName(),
	}
	if err := r.Get(ctx, nn, helmSecret); err != nil {
		fn.Conditions2.Build(
			&conditions, "Helm", metav1.Condition{
				Type:    "ReleaseSecretExists",
				Status:  metav1.ConditionFalse,
				Reason:  "SecretNotFound",
				Message: err.Error(),
			},
		)
		helmSecret = nil
	}

	if helmSecret != nil {
		fn.Conditions2.Build(
			&conditions, "Helm", metav1.Condition{
				Type:    "ReleaseSecretExists",
				Status:  metav1.ConditionTrue,
				Reason:  "SecretFound",
				Message: fmt.Sprintf("secret %s found", helmSecret.Name),
			},
		)
	}

	//x, ok := helmSecret.Data[MysqlRootPasswordKey]
	//req.SetStateData(MysqlRootPasswordKey, fn.IfThenElse(ok, string(x), fn.CleanerNanoid(40)))
	//y, ok := helmSecret.Data[MysqlPasswordKey]
	//req.SetStateData(MysqlPasswordKey, fn.IfThenElse(ok, string(y), fn.CleanerNanoid(40)))
	//
	if fn.Conditions2.Equal(conditions, req.mysqlSvc.Status.Conditions) {
		req.logger.Infof("Status is already in sync, so moving forward with ops")
		return nil, nil
	}

	req.logger.Infof("status is different, so updating status ...")
	if err := r.Status().Update(ctx, req.mysqlSvc); err != nil {
		return nil, err
	}

	return reconcileResult.OKP()
}

func (r *ServiceReconciler) preOps(ctx context.Context, req *ServiceReconReq) error {
	if _, err := fn.JsonGet[string](req.mysqlSvc.Status.GeneratedVars, MysqlRootPasswordKey); err != nil {
		m, err := req.mysqlSvc.Status.GeneratedVars.ToMap()
		if err != nil {
			return err
		}
		m[MysqlRootPasswordKey] = fn.CleanerNanoid(40)
		m[MysqlPasswordKey] = fn.CleanerNanoid(40)
		if err := req.mysqlSvc.Status.GeneratedVars.FillFrom(m); err != nil {
			return err
		}
		return r.Status().Update(ctx, req.mysqlSvc)
	}
	return nil
}

func (r *ServiceReconciler) reconcileOperations(ctx context.Context, req *ServiceReconReq) (ctrl.Result, error) {

	if err := r.preOps(ctx, req); err != nil {
		return r.failWithErr(ctx, req, err)
	}

	//hash := req.mysqlSvc.Hash()
	//if hash == req.mysqlSvc.Status.LastHash {
	//	return reconcileResult.OK()
	//}

	b, err := templates.Parse(templates.MySqlStandalone, req.mysqlSvc)
	if err != nil {
		return reconcileResult.FailedE(err)
	}

	if _, err := fn.KubectlApplyExec(b); err != nil {
		return reconcileResult.FailedE(errors.NewEf(err, "could not apply kubectl for mysql standalone"))
	}

	if err := r.reconcileOutput(ctx, req); err != nil {
		return reconcileResult.FailedE(err)
	}

	//req.mysqlSvc.Status.LastHash = hash

	if err := r.Status().Update(ctx, req.mysqlSvc); err != nil {
		return ctrl.Result{}, err
	}

	return reconcileResult.OK()
}

func (r *ServiceReconciler) reconcileOutput(ctx context.Context, req *ServiceReconReq) error {
	rootPasswd, err := fn.JsonGet[string](req.mysqlSvc.Status.GeneratedVars, MysqlRootPasswordKey)
	if err != nil {
		return err
	}

	hostUrl := fmt.Sprintf("%s.%s.svc.cluster.local", req.mysqlSvc.Name, req.mysqlSvc.Namespace)

	scrt := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("msvc-%s", req.mysqlSvc.Name),
			Namespace: req.mysqlSvc.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				fn.AsOwner(req.mysqlSvc, true),
			},
			Labels: req.mysqlSvc.GetLabels(),
		},
		StringData: map[string]string{
			"ROOT_PASSWORD": rootPasswd,
			"HOSTS":         hostUrl,
		},
	}

	if err := fn.KubectlApply(ctx, r.Client, scrt); err != nil {
		return err
	}

	return nil
}

func (r *ServiceReconciler) kWatcherMap(o client.Object) []reconcile.Request {
	labels := o.GetLabels()
	if s := labels["app.kubernetes.io/component"]; s != "primary" {
		return nil
	}
	if s := labels["app.kubernetes.io/name"]; s != "mysql" {
		return nil
	}
	resourceName := labels["app.kubernetes.io/instance"]
	nn := types.NamespacedName{Namespace: o.GetNamespace(), Name: resourceName}

	return []reconcile.Request{
		{NamespacedName: nn},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlStandalone.Service{}).
		Watches(
			&source.Kind{
				Type: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": constants.MsvcApiVersion,
						"kind":       constants.HelmMySqlDBKind,
					},
				},
			}, handler.EnqueueRequestsFromMapFunc(
				func(obj client.Object) []reconcile.Request {
					var svcList mysqlStandalone.ServiceList
					key, value := mysqlStandalone.Service{}.LabelRef()
					if err := r.List(
						context.TODO(), &svcList, &client.ListOptions{
							LabelSelector: apiLabels.SelectorFromValidatedSet(map[string]string{key: value}),
						},
					); err != nil {
						return nil
					}
					var reqs []reconcile.Request
					for _, item := range svcList.Items {
						nn := types.NamespacedName{
							Name:      item.Name,
							Namespace: item.Namespace,
						}

						for _, req := range reqs {
							if req.NamespacedName.String() == nn.String() {
								return nil
							}
						}
						reqs = append(reqs, reconcile.Request{NamespacedName: nn})
					}
					return reqs
				},
			),
		).
		Watches(&source.Kind{Type: &appsv1.StatefulSet{}}, handler.EnqueueRequestsFromMapFunc(r.kWatcherMap)).
		Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(r.kWatcherMap)).
		Complete(r)
}
