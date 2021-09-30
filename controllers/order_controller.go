/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "github.com/order/api/v1"
)

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.order.io,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.order.io,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.order.io,resources=orders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Order object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var order webappv1.Order
	if err := r.Get(ctx, req.NamespacedName, &order); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Error(err, "error getting order ")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// name of your custom finalizer
	myFinalizerName := "storage.finalizers.order.io"

	if order.ObjectMeta.DeletionTimestamp.IsZero() {
		if order.Status.Status == "" {
			// Order is just created, so it's initial status is empty. We set status to running and update the status.
			order.Status.Status = "Running"
			if err := r.Status().Update(ctx, &order); err != nil {
				return reconcile.Result{}, err
			}
		}

		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(order.ObjectMeta.Finalizers, myFinalizerName) {
			order.ObjectMeta.Finalizers = append(order.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, &order); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else { // The object is being deleted
		order.Status.Status = "Terminating"
		if err := r.Status().Update(ctx, &order); err != nil {
			return reconcile.Result{}, err
		}

		if containsString(order.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteExternalDependencies(&order); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}
			// remove our finalizer from the list and update it.
			order.ObjectMeta.Finalizers = removeString(order.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, &order); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	//lbls := labels.Set{
	//	"app": order.Name,
	//	"version": "v0.1",
	//}

	lbls := labels.Set{}
	existingPods := corev1.PodList{}

	listOptions := &client.ListOptions{
		Namespace:     req.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	if err := r.Client.List(ctx, &existingPods, listOptions); err != nil {
		logger.Error(err, "failed to list existing pods in the podSet")
		return ctrl.Result{}, err
	}

	podToBeCreated := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: order.Name + "-pod",
			Namespace:    order.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			}},
	}

	if err := r.Client.Create(ctx, &podToBeCreated); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Order{}).
		Complete(r)
}

func (r *OrderReconciler) deleteExternalDependencies(order *webappv1.Order) error {
	fmt.Println("deleting the external dependencies")
	//
	// delete the external dependency here
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.
	return nil
}

//
// Helper functions to check and remove string from a slice of strings.
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
