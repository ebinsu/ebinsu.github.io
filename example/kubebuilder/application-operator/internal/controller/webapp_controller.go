/*
Copyright 2024.

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

	webappv1 "ebinsu.cn/m/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WebAppReconciler reconciles a WebApp object
type WebAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	myFinalizerName = "webapp.ebinsu.cn/finalizer"
)

//+kubebuilder:rbac:groups=webapp.ebinsu.cn,resources=webapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.ebinsu.cn,resources=webapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.ebinsu.cn,resources=webapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	webapp := &webappv1.WebApp{}
	err := r.Client.Get(ctx, req.NamespacedName, webapp)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "failed to get WebApp")
		return ctrl.Result{}, err
	}

	reqLogger.Info("recv req")

	// examine DeletionTimestamp to determine if object is under deletion
	if webapp.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(webapp, myFinalizerName) {
			controllerutil.AddFinalizer(webapp, myFinalizerName)
			if err := r.Update(ctx, webapp); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(webapp, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			r.deleteDeployAndService(ctx, req, *webapp)

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(webapp, myFinalizerName)
			if err := r.Update(ctx, webapp); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	r.reconcileDeployment(ctx, req, *webapp)
	r.reconcileService(ctx, req, *webapp)
	return ctrl.Result{}, nil
}

func (r *WebAppReconciler) deleteDeployAndService(ctx context.Context, req ctrl.Request, webapp webappv1.WebApp) {
	reqLogger := log.FromContext(ctx)
	foundDeploy := &appsv1.Deployment{}
	deployName := webapp.Spec.Name + "-deploy"
	foundDeployErr := r.Client.Get(ctx, types.NamespacedName{Name: deployName, Namespace: req.NamespacedName.Namespace}, foundDeploy)
	if foundDeployErr == nil {
		r.Client.Delete(ctx, foundDeploy)
		reqLogger.Info("delete deploy:" + deployName)
	}
	foundService := &coreV1.Service{}
	foundServiceErr := r.Client.Get(ctx, types.NamespacedName{Name: webapp.Name, Namespace: req.NamespacedName.Namespace}, foundService)
	if foundServiceErr == nil {
		r.Client.Delete(ctx, foundService)
		reqLogger.Info("delete service:" + webapp.Name)
	}
}

func (r *WebAppReconciler) reconcileService(ctx context.Context, req ctrl.Request, webapp webappv1.WebApp) {
	reqLogger := log.FromContext(ctx)
	foundService := &coreV1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: webapp.Name, Namespace: req.NamespacedName.Namespace}, foundService)
	if err != nil {
		if errors.IsNotFound(err) {
			// create new
			service := r.createService(webapp)
			r.Client.Create(ctx, service)

		} else {
			reqLogger.Error(err, "failed to get Service")
		}
	}
}

func (r *WebAppReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, webapp webappv1.WebApp) {
	reqLogger := log.FromContext(ctx)
	foundDeploy := &appsv1.Deployment{}
	deployName := webapp.Spec.Name + "-deploy"
	err := r.Client.Get(ctx, types.NamespacedName{Name: deployName, Namespace: req.NamespacedName.Namespace}, foundDeploy)
	if err != nil {
		if errors.IsNotFound(err) {
			// create new
			deploy := r.createDeployment(webapp)
			r.Client.Create(ctx, deploy)
		} else {
			reqLogger.Error(err, "failed to get WebApp")
		}
		return
	}
	// update replicas and images
	foundDeploy.Spec.Replicas = webapp.Spec.Replicas
	for _, c := range foundDeploy.Spec.Template.Spec.Containers {
		if c.Name == webapp.Spec.Name {
			c.Image = webapp.Spec.Image
			break
		}
	}
	r.Client.Update(ctx, foundDeploy)
}

func (r *WebAppReconciler) createService(webapp webappv1.WebApp) *coreV1.Service {
	return &coreV1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        webapp.Name,
			Namespace:   webapp.Namespace,
			Labels:      webapp.Labels,
			Annotations: webapp.Annotations,
		},
		Spec: coreV1.ServiceSpec{
			Selector: getSelectLable(webapp),
			Type:     coreV1.ServiceTypeNodePort,
			Ports: []coreV1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}
}

func (r *WebAppReconciler) createDeployment(webapp webappv1.WebApp) *appsv1.Deployment {
	deployName := webapp.Spec.Name + "-deploy"
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        deployName,
			Namespace:   webapp.Namespace,
			Labels:      webapp.Labels,
			Annotations: webapp.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getSelectLable(webapp),
			},
			Replicas: webapp.Spec.Replicas,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getSelectLable(webapp),
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:  webapp.Name,
							Image: webapp.Spec.Image,
							Env: []coreV1.EnvVar{
								{
									Name:  "JDK_JAVA_OPTIONS",
									Value: "-XX:+UseG1GC -XX:MaxRAMPercentage=75 -XX:MaxMetaspaceSize=128m -Xss256k -XX:G1PeriodicGCInterval=900k",
								},
							},
							Ports: []coreV1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
	return deploy
}

func getSelectLable(webapp webappv1.WebApp) map[string]string {
	return map[string]string{
		"app": webapp.Spec.Name,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.WebApp{}).
		Complete(r)
}
