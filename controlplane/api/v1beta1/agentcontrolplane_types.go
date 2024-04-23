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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type AgentControlPlaneMachineTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

	// InfrastructureRef is a required reference to a custom resource
	// offered by an infrastructure provider.
	InfrastructureRef corev1.ObjectReference `json:"infrastructureRef"`
}

// AgentControlPlaneSpec defines the desired state of AgentControlPlane
type AgentControlPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AgentConfigSpec specs for the AgentConfig
	AgentConfigSpec AgentControlPlaneConfigSpec      `json:"agentConfigSpec"`
	MachineTemplate AgentControlPlaneMachineTemplate `json:"machineTemplate"`
	Replicas        int32                            `json:"replicas,omitempty"`
	ReleaseImage    string                           `json:"releaseImage"`
}

// AgentControlPlaneConfigSpec defines configuration for the agent-provisioned cluster
type AgentControlPlaneConfigSpec struct {
	// PullSecretRef references pull secret necessary for the cluster installation
	PullSecretRef *corev1.LocalObjectReference `json:"pullSecretRef,omitempty"`

	// ClusterDeploymentRef references the ClusterDeployment used to create the cluster
	ClusterDeploymentRef *corev1.ObjectReference `json:"clusterDeploymentRef,omitempty"`

	// Base domain for install cluster
	BaseDomain string `json:"baseDomain"`
}

// AgentControlPlaneStatus defines the observed state of AgentControlPlane
type AgentControlPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector

// AgentControlPlane is the Schema for the agentcontrolplanes API
type AgentControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentControlPlaneSpec   `json:"spec,omitempty"`
	Status AgentControlPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentControlPlaneList contains a list of AgentControlPlane
type AgentControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentControlPlane{}, &AgentControlPlaneList{})
}
