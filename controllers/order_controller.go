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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	orderFinalizer := "webappv1.finalizer"
	err := r.Get(ctx, req.NamespacedName, instance)
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

	// state transition PENDING -> RUNNING -> DONE

	switch instance.Status.Status {
	case webappv1.StatusPending:
		fmt.Println("Status: PENDING")
		if err = r.updateStatus(ctx, instance, webappv1.StatusRunning); err != nil {
			return ctrl.Result{}, err
		}

		fmt.Println("Adding finalizer..")
		controllerutil.AddFinalizer(instance, orderFinalizer)
		if err = r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}

		err = r.patchResources(ctx, instance, req)
		if err != nil {
			return ctrl.Result{}, err
		}
	case webappv1.StatusRunning:
		fmt.Println("Status: RUNNING")
		if instance.DeletionTimestamp.IsZero() {
			err = r.patchResources(ctx, instance, req)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else { // is deleting..
			if err = r.updateStatus(ctx, instance, webappv1.StatusDeleting); err != nil {
				return ctrl.Result{}, err
			}
		}
	case webappv1.StatusDeleting:
		fmt.Println("Status: DELETING")
		fmt.Println("Deleting dependencies..")
		if err := r.deleteExternalDependencies(ctx, req, instance); err != nil {
			return ctrl.Result{}, err
		}

		fmt.Println("Deleting finalizer from order..")
		controllerutil.RemoveFinalizer(instance, orderFinalizer)
		if err = r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Order{}).
		Complete(r)
}

func (r *OrderReconciler) updateStatus(ctx context.Context, instance *webappv1.Order, status string) error {
	fmt.Println("Updating status to " + status)
	instance.Status.Status = status
	// update status
	return r.Status().Update(ctx, instance)
}

func (r *OrderReconciler) patchResources(ctx context.Context, instance *webappv1.Order, req ctrl.Request) error {
	resources := r.prepareResources(instance, req)
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("order-controller")}
	for _, resource := range resources {
		if err := ctrl.SetControllerReference(instance, resource, r.Scheme); err != nil {
			return err
		}

		if err := r.Patch(ctx, resource, client.Apply, applyOpts...); err != nil {
			return err
		}

		fmt.Println("Resource patched: ", resource.GetName())
	}
	return nil
}

func (r *OrderReconciler) deleteExternalDependencies(ctx context.Context, req ctrl.Request, order *webappv1.Order) error {
	fmt.Println("deleting the external dependencies")
	//
	// delete the external dependency here
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.

	resources := r.prepareResources(order, req)

	for _, resource := range resources {
		if err := r.Client.Delete(ctx, resource); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

func (r *OrderReconciler) prepareResources(instance *webappv1.Order, req ctrl.Request) []client.Object {
	result := []client.Object{}
	replicas := int32(2)
	podLabels := map[string]string{
		"foo": instance.Spec.Foo,
	}
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:         req.Name,
			GenerateName: req.Name + "-deployment",
			Namespace:    req.Namespace,
			Labels: map[string]string{
				"foo": instance.Spec.Foo,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					Volumes: nil,
					Containers: []corev1.Container{{
						Name:            req.Name,
						Image:           "nginx",
						Command:         nil,
						Args:            nil,
						WorkingDir:      "",
						Ports:           nil,
						EnvFrom:         nil,
						Env:             nil,
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    nil,
						VolumeDevices:   nil,
						LivenessProbe:   nil,
						ReadinessProbe:  nil,
						StartupProbe:    nil,
						ImagePullPolicy: "",
					}},
					RestartPolicy: "",
				},
			},
		},
	}

	result = append(result, &deployment)
	return result
}
