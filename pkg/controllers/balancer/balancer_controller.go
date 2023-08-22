/*
Copyright 2023 xincechen.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller-controller")

// BalancerReconciler reconciles a Balancer object
type BalancerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// newReconciler creates the BalancerReconciler with input controller-manager.
func newReconciler(manager manager.Manager) reconcile.Reconciler {
	return &BalancerReconciler{
		manager.GetClient(),
		manager.GetScheme(),
	}
}

//+kubebuilder:rbac:groups=exposer.xincechen.io,resources=balancers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=exposer.xincechen.io,resources=balancers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=exposer.xincechen.io,resources=balancers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=replicasets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Balancer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *BalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.name", req.Name)
	reqLogger.Info("Reconciling Balancer")

	balancer := &exposerv1.Balancer{}
	if err := r.Get(ctx, req.NamespacedName, balancer); err != nil {
		// controller not exists
		if errors.IsNotFound(err) {
			// the namespaced name in request is not found, return empty result and requeue the request
			return reconcile.Result{}, nil
		}
	}

	// Founded. Update SVCs, deployments, etc. according to the expected Balancer.
	// If any error happens, the request would be requeue
	if err := r.syncFrontendService(balancer); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncDeployment(balancer); err != nil {
		return reconcile.Result{}, nil
	}
	if err := r.syncBackendServices(balancer); err != nil {
		return reconcile.Result{}, nil
	}
	if err := r.syncBalancerStatus(balancer); err != nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controllers with the Manager.
func (r *BalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("controller-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// takes events provided by a Source and uses the EventHandler to enqueue reconcile.Requests in response to the events.
	if err = c.Watch(source.Kind(mgr.GetCache(), &exposerv1.Balancer{}), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// the changes of the configmap, pod, and svc which are created by controller will also be enqueued.
	//if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.ConfigMap{}), &handler.EnqueueRequestForOwner(
	//	mgr.GetScheme(), mgr.GetRESTMapper(), &exposerv1.Balancer{}, handler.OnlyControllerOwner())); err != nil {
	//
	//}
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.ConfigMap{}), handler.EnqueueRequestForOwner(
		mgr.GetScheme(), mgr.GetRESTMapper(), &exposerv1.Balancer{}, handler.OnlyControllerOwner())); err != nil {
		return err
	}
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Pod{}), handler.EnqueueRequestForOwner(
		mgr.GetScheme(), mgr.GetRESTMapper(), &exposerv1.Balancer{}, handler.OnlyControllerOwner())); err != nil {
		return err
	}
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Service{}), handler.EnqueueRequestForOwner(
		mgr.GetScheme(), mgr.GetRESTMapper(), &exposerv1.Balancer{}, handler.OnlyControllerOwner())); err != nil {
		return err
	}

	return nil
}
