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

	"github.com/openshift-assisted/cluster-api-agent/controlplane/api/v1beta1"
	hiveext "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// ClusterDeploymentReconciler reconciles a ClusterDeployment object
type ClusterDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hivev1.ClusterDeployment{}).
		Watches(&v1beta1.AgentControlPlane{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
func (r *ClusterDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	clusterDeployment := &hivev1.ClusterDeployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, clusterDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("Reconciling ClusterDeployment", "name", clusterDeployment.Name, "namespace", clusterDeployment.Namespace)

	agentCPList := &v1beta1.AgentControlPlaneList{}
	if err := r.Client.List(ctx, agentCPList, client.InNamespace(clusterDeployment.Namespace)); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("found agentcontrolplane", "num", len(agentCPList.Items))

	if len(agentCPList.Items) == 0 {
		return ctrl.Result{}, nil
	}
	for _, agentCP := range agentCPList.Items {
		if IsAgentControlPlaneReferencingClusterDeployment(agentCP, clusterDeployment) {
			log.Info("ClusterDeployment is referenced by AgentControlPlane")
			return r.ensureAgentClusterInstall(ctx, clusterDeployment, agentCP)
		}
	}
	log.Info("No agentcontrolplane is referenced by ClusterDeployment", "name", clusterDeployment.Name, "namespace", clusterDeployment.Namespace)

	return ctrl.Result{}, nil
}

func IsAgentControlPlaneReferencingClusterDeployment(agentCP v1beta1.AgentControlPlane, clusterDeployment *hivev1.ClusterDeployment) bool {
	return agentCP.Spec.AgentConfigSpec.ClusterDeploymentRef != nil &&
		agentCP.Spec.AgentConfigSpec.ClusterDeploymentRef.GroupVersionKind().String() == hivev1.SchemeGroupVersion.WithKind("ClusterDeployment").String() &&
		agentCP.Spec.AgentConfigSpec.ClusterDeploymentRef.Namespace == clusterDeployment.Namespace &&
		agentCP.Spec.AgentConfigSpec.ClusterDeploymentRef.Name == clusterDeployment.Name
}

func (r *ClusterDeploymentReconciler) ensureAgentClusterInstall(ctx context.Context, clusterDeployment *hivev1.ClusterDeployment, acp v1beta1.AgentControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if clusterDeployment.Spec.ClusterInstallRef != nil {
		log.Info(
			"Skipping reconciliation: cluster deployment already has a referenced agent cluster install",
			"cluster_deployment_name", clusterDeployment.Name,
			"cluster_deployment_namespace", clusterDeployment.Namespace,
			"agent_cluster_install", clusterDeployment.Spec.ClusterInstallRef.Name,
		)
		return ctrl.Result{}, nil
	}
	imageSet := &hivev1.ClusterImageSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDeployment.Name,
			Namespace: clusterDeployment.Namespace,
		},
		Spec: hivev1.ClusterImageSetSpec{
			ReleaseImage: acp.Spec.AgentConfigSpec.ReleaseImage,
		},
	}
	if err := r.Client.Create(ctx, imageSet); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Info("ImageSet already exists", "name", imageSet.Name, "namespace", imageSet.Namespace)
			return ctrl.Result{}, err
		}
	}

	log.Info("Creating agent cluster install for ClusterDeployment", "name", clusterDeployment.Name, "namespace", clusterDeployment.Namespace)
	agentClusterInstall := &hiveext.AgentClusterInstall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDeployment.Name,
			Namespace: clusterDeployment.Namespace,
		},
		Spec: hiveext.AgentClusterInstallSpec{
			ClusterDeploymentRef: corev1.LocalObjectReference{Name: clusterDeployment.Name},
			ProvisionRequirements: hiveext.ProvisionRequirements{
				ControlPlaneAgents: int(acp.Spec.Replicas),
			},
			SSHPublicKey: acp.Spec.AgentConfigSpec.SSHAuthorizedKey,
			ImageSetRef:  &hivev1.ClusterImageSetReference{Name: imageSet.Name},
		},
	}
	if err := r.Client.Create(ctx, agentClusterInstall); err != nil {
		return ctrl.Result{}, err
	}
	clusterDeployment.Spec.ClusterInstallRef = &hivev1.ClusterInstallLocalReference{
		Group:   hiveext.Group,
		Version: hiveext.Version,
		Kind:    "AgentClusterInstall",
		Name:    agentClusterInstall.Name,
	}
	err := r.Client.Update(ctx, clusterDeployment)
	return ctrl.Result{}, err
}
