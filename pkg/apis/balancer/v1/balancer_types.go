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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Protocol string
type Port int32

const (
	TCP Protocol = "TCP"
	UDP Protocol = "UDP"
)

// BalancerSpec defines the desired state of Balancer
type BalancerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinItems=1
	Backends []BackendSpec `json:"backends"`

	Selector map[string]string `json:"selector,omitempty"`

	Ports []BalancerPort `json:"ports"`
}

// BackendSpec defines the desired status of endpoints of Balancer
// +k8s:openapi-gen=true
type BackendSpec struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=1
	Weight int32 `json:"weight"`

	Selector map[string]string `json:"selector,omitempty"`
}

// BalancerPort contains the endpoints and exposed ports.
// +k8s:openapi-gen=true
type BalancerPort struct {
	// The name of this port within the manager. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. This maps to
	// the 'Name' field in EndpointPort objects.
	// Optional if only one BalancerPort is defined on this service.
	// +required
	Name string `json:"name,omitempty"`

	// +optional
	Protocol Protocol `json:"protocol,omitempty"`

	// the port that will be exposed by the manager
	Port Port `json:"port"`

	// the port that used by the container
	// +optional
	TargetPort intstr.IntOrString `json:"targetPort,omitempty"`
}

// BalancerStatus defines the observed state of Balancer
type BalancerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +optional
	ActiveBackendsNum int32 `json:"activeBackendsNum,omitempty"`

	// +optional
	ObsoleteBackendsNum int32 `json:"obsoleteBackendsNum,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Balancer is the Schema for the balancers API
type Balancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BalancerSpec   `json:"spec,omitempty"`
	Status BalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// BalancerList contains a list of Balancer
type BalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Balancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Balancer{}, &BalancerList{})
}
