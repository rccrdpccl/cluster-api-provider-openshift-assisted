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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	metal3 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	bootstrapv1beta1 "github.com/openshift-assisted/cluster-api-agent/api/v1beta1"
)

// AgentBootstrapConfigReconciler reconciles a AgentBootstrapConfig object
type AgentBootstrapConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *AgentBootstrapConfigReconciler) getMachineTemplate(ctx context.Context, machineDeployment clusterv1.MachineDeployment) *metal3.Metal3MachineTemplate {
	log := ctrl.LoggerFrom(ctx)
	machineTemplateRef := machineDeployment.Spec.Template.Spec.InfrastructureRef
	log.Info("machine template ref", "groupversionkind", machineTemplateRef.GroupVersionKind())
	if machineTemplateRef.GroupVersionKind() == metal3.GroupVersion.WithKind("Metal3MachineTemplate") {
		nsName := types.NamespacedName{
			Name: machineTemplateRef.Name,
			Namespace: machineDeployment.Namespace,
		}
		machineTemplate := &metal3.Metal3MachineTemplate{}
		if err := r.Client.Get(ctx, nsName, machineTemplate); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("machine template not found", "namespacedname", nsName)
				return nil
			}
			return nil
		}
		return machineTemplate
	}
	return nil
}

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments;machinedeployments/status,verbs=get;list;watch;create;update;patch;delete
//metal3machinetemplates" in API group "infrastructure.cluster.x-k8s.io"
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machinetemplates;metal3machinetemplates/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AgentBootstrapConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *AgentBootstrapConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	config := &bootstrapv1beta1.AgentBootstrapConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	machineDeployments := clusterv1.MachineDeploymentList{}
	err := r.Client.List(context.Background(), &machineDeployments, client.InNamespace(req.Namespace))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err		
	}

	for _, md := range machineDeployments.Items {
		log.Info("found machine deployment", "name", md.Name, "namespace", md.Namespace)
		machineTemplate := r.getMachineTemplate(ctx, md)
		if machineTemplate == nil {
			log.Info("Machine template not metal3 or not found")
			continue
		}
		log.Info("found machine template", "name", machineTemplate.Name, "namespace", machineTemplate.Namespace)
	}
	// agentboostrapconfig
	// machinedeployment
	// infraref (machinetemplate)
	
	
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentBootstrapConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bootstrapv1beta1.AgentBootstrapConfig{}).
		Watches(
			&clusterv1.MachineDeployment{},
			handler.EnqueueRequestsFromMapFunc(r.FilterMachineDeployment),
		).Complete(r)
}

// Filter machine deployments owned by 
func (r *AgentBootstrapConfigReconciler) FilterMachineDeployment(_ context.Context, o client.Object) []ctrl.Request {
	result := []ctrl.Request{}
	m, ok := o.(*clusterv1.MachineDeployment)
	if !ok {
		panic(fmt.Sprintf("Expected a MachineDeployment but got a %T", o))
	}
	// m.Spec.ClusterName

	if m.Spec.Template.Spec.Bootstrap.ConfigRef != nil  && m.Spec.Template.Spec.Bootstrap.ConfigRef.GroupVersionKind() == bootstrapv1beta1.GroupVersion.WithKind("AgentBootstrapConfig") {
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.Template.Spec.Bootstrap.ConfigRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}
	return result
}
