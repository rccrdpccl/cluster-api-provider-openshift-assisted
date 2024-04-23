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
	"github.com/openshift/hive/apis/hive/v1/agent"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/annotations"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	controlplanev1beta1 "github.com/openshift-assisted/cluster-api-agent/controlplane/api/v1beta1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
)

// AgentControlPlaneReconciler reconciles a AgentControlPlane object
type AgentControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hive.openshift.io,resources=clusterimagesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions.hive.openshift.io,resources=agentclusterinstalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions.hive.openshift.io,resources=agentclusterinstalls/status,verbs=get
// +kubebuilder:rbac:groups=hive.openshift.io,resources=clusterdeployments/status,verbs=get
// +kubebuilder:rbac:groups=hive.openshift.io,resources=clusterdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=agentcontrolplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=agentcontrolplanes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=agentcontrolplanes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AgentControlPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *AgentControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	acp := &controlplanev1beta1.AgentControlPlane{}
	if err := r.Client.Get(ctx, req.NamespacedName, acp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, acp.ObjectMeta)
	if err != nil {
		logger.Error(err, "Failed to retrieve owner Cluster from the API Server")
		return ctrl.Result{}, err
	}
	if cluster == nil {
		logger.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}
	//logger = logger.WithValues("Cluster", klog.KObj(cluster))
	//ctx = ctrl.LoggerInto(ctx, logger)

	if annotations.IsPaused(cluster, acp) {
		logger.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	// TODO: handle changes in pull-secret (for example)
	// create clusterdeployment if not set
	if acp.Spec.AgentConfigSpec.ClusterDeploymentRef == nil {
		err, clusterDeployment := r.createClusterDeployment(ctx, acp, cluster.Name)
		if err != nil {
			logger.Error(
				err,
				"couldn't create clusterDeployment",
				"name", clusterDeployment.Name,
				"namespace", clusterDeployment.Namespace,
			)
			return ctrl.Result{}, err
		}

		if clusterDeployment != nil {
			acp.Spec.AgentConfigSpec.ClusterDeploymentRef = &corev1.ObjectReference{
				Name:       clusterDeployment.Name,
				Namespace:  clusterDeployment.Namespace,
				Kind:       "ClusterDeployment",
				APIVersion: hivev1.SchemeGroupVersion.String(),
			}
			if err := r.Client.Update(ctx, acp); err != nil {
				logger.Error(err, "couldn't update AgentControlPlane", "name", acp.Name, "namespace", acp.Namespace)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentControlPlaneReconciler) createClusterDeployment(ctx context.Context, acp *controlplanev1beta1.AgentControlPlane, clusterName string) (error, *hivev1.ClusterDeployment) {
	var pullSecret *corev1.LocalObjectReference
	if acp.Spec.AgentConfigSpec.PullSecretRef != nil {
		pullSecret = acp.Spec.AgentConfigSpec.PullSecretRef
	}
	//TODO: create logic for placeholder pull secret

	// Get cluster clusterName instead of reference to ACP clusterName
	clusterDeployment := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      acp.Name,
			Namespace: acp.Namespace,
		},
		Spec: hivev1.ClusterDeploymentSpec{
			ClusterName: clusterName,
			BaseDomain:  acp.Spec.AgentConfigSpec.BaseDomain,
			Platform: hivev1.Platform{
				AgentBareMetal: &agent.BareMetalPlatform{
					//AgentSelector: metav1.LabelSelector{}, // TODO: What label should we select?
				},
			},
			//PreserveOnDelete: True, // TODO take from CR
			//ControlPlaneConfig: hivev1.ControlPlaneConfigSpec{}, //
			//Ingress: ( []hivev1.ClusterIngress)
			//CertificateBundles: ([]hivev1.CertificateBundleSpec)
			// ManageDNS: bool,
			//ClusterMetadata: *hivev1.ClusterMetadata.
			Installed: false,
			// Provisioning: *hivev1.Provisioning
			//ClusterInstallRef: // reference to AgentClusterInstall  *ClusterInstallLocalReference
			// ClusterPoolRef *ClusterPoolReference `json:"clusterPoolRef,omitempty"`,
			// ClusterPoolRef *ClusterPoolReference `json:"clusterPoolRef,omitempty"`
			PullSecretRef: pullSecret,
		},
	}
	return r.Create(ctx, clusterDeployment), clusterDeployment
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&controlplanev1beta1.AgentControlPlane{}).
		Complete(r)
}
