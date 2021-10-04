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
	webappv1 "github.com/order/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	instance := &webappv1.Order{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// object not found, could have been deleted after
			// reconcile request, hence don't requeue
			return ctrl.Result{}, nil
		}

		// error reading the object, requeue the request
		return ctrl.Result{}, err
	}

	// if no phase set, default to Pending
	if instance.Status.Status == "" {
		instance.Status.Status = webappv1.StatusPending
	}

	switch instance.Status.Status {
	case webappv1.StatusPending:
		println("Phase: PENDING")

		//diff, err := schedule.TimeUntilSchedule(instance.Spec.Schedule)
		//if err != nil {
		//	println(err, "Schedule parsing failure")
		//
		//	return ctrl.Result{}, err
		//}
		//
		//println("Schedule parsing done", "Result", fmt.Sprintf("%v", diff))
		//
		//if diff > 0 {
		//	// not yet time to execute, wait until scheduled time
		//	return ctrl.Result{RequeueAfter: diff * time.Second}, nil
		//}
		//
		//println("It's time!", "Ready to execute", instance.Spec.Command)
		//// change state
		instance.Status.Status = webappv1.StatusRunning
	case webappv1.StatusRunning:
		println("Phase: RUNNING")

		//pod := spawn.NewPodForCR(instance)
		//err := ctrl.SetControllerReference(instance, pod, r.Scheme)
		//if err != nil {
		//	// requeue with error
		//	return ctrl.Result{}, err
		//}

		query := &corev1.Pod{}
		// try to see if the pod already exists
		err = r.Get(context.TODO(), req.NamespacedName, query)
		return ctrl.Result{}, nil
		//if err != nil && errors.IsNotFound(err) {
		//	//// does not exist, create a pod
		//	//err = r.Create(context.TODO(), pod)
		//	//if err != nil {
		//	//	return ctrl.Result{}, err
		//	//}
		//	//
		//	//// Successfully created a Pod
		//	//println("Pod Created successfully", "name", pod.Name)
		//	return ctrl.Result{}, nil
		//} else if err != nil {
		//	// requeue with err
		//	println(err, "cannot create pod")
		//	return ctrl.Result{}, err
		//} else if query.Status.Phase == corev1.PodFailed ||
		//	query.Status.Phase == corev1.PodSucceeded {
		//	// pod already finished or errored out`
		//	println("Container terminated", "reason", query.Status.Reason,
		//		"message", query.Status.Message)
		//	instance.Status.Status = webappv1.StatusDone
		//} else {
		//	// don't requeue, it will happen automatically when
		//	// pod status changes
		//	return ctrl.Result{}, nil
		//}
	case webappv1.StatusDone:
		println("Phase: DONE")
		// reconcile without requeuing
		return ctrl.Result{}, nil
	default:
		println("NOP")
		return ctrl.Result{}, nil
	}

	// update status
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Order{}).
		Owns().
		Owns()
		Complete(r)
}

func (r *OrderReconciler) deleteExternalDependencies(ctx context.Context, req ctrl.Request, order *webappv1.Order) error {
	fmt.Println("deleting the external dependencies")
	//
	// delete the external dependency here
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.

	lbls := labels.Set{
		"app":     "order",
		"version": "v0.1",
	}
	listOptions := client.ListOptions{
		Namespace:     req.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}

	if err := r.Client.DeleteAllOf(ctx, &corev1.Pod{}, &client.DeleteAllOfOptions{
		ListOptions:   listOptions,
		DeleteOptions: client.DeleteOptions{},
	}); err != nil {
		return err
	}
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
