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

	bootstrapv1beta1 "github.com/openshift-assisted/cluster-api-agent/bootstrap/api/v1beta1"
	aiv1beta1 "github.com/openshift/assisted-service/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InfraEnvReconciler reconciles a InfraEnv object
type InfraEnvReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *InfraEnvReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1beta1.InfraEnv{}).
		Complete(r)
}
func (r *InfraEnvReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	infraEnv := &aiv1beta1.InfraEnv{}
	if err := r.Client.Get(ctx, req.NamespacedName, infraEnv); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if infraEnv == nil {
		return ctrl.Result{}, nil
	}
	configName, ok := infraEnv.Labels[agentBootstrapConfigLabel]
	if !ok {
		return ctrl.Result{}, nil
	}

	// Check for ISO
	if infraEnv.Status.ISODownloadURL == "" {
		log.Info("InfraEnv corresponding has no image URL available.", "infra_env_name", infraEnv.Name)
		return ctrl.Result{}, nil
	}
	log.Info("InfraEnv corresponding has image URL available.", "infra_env_name", infraEnv.Name)

	//TODO: list all agentbootstrapconfig that ref this infraenv
	agentBootstrapConfig := &bootstrapv1beta1.AgentBootstrapConfig{}
	name := client.ObjectKey{Namespace: infraEnv.Namespace, Name: configName}
	if err := r.Client.Get(ctx, name, agentBootstrapConfig); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("agentbootstrapconfig not found", "namespace/name", name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if agentBootstrapConfig.Spec.InfraEnvRef.Name != infraEnv.Name {
		log.Info("InfraEnvRef on agentbootstrap config doesn't match infraenv found", "agentBootstrap.InfraEnvRef", agentBootstrapConfig.Spec.InfraEnvRef.Name, "infra env", infraEnv.Name)
		return ctrl.Result{}, nil
	}

	// Add ISO to agentBootstrapConfig status
	agentBootstrapConfig.Status.ISODownloadURL = infraEnv.Status.ISODownloadURL

	if err := r.Client.Status().Update(ctx, agentBootstrapConfig); err != nil {
		log.Error(err, "couldn't update agentbootstrapconfig with iso url", "infra env name", infraEnv.Name, "config", agentBootstrapConfig.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
