/*
Copyright 2023.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MSSQLServerSpec defines the desired state of MSSQLServer
type MSSQLServerSpec struct {
	// Volumes allows configuration of additional volumes on the output Deployment definition.
	// Volumes specified will be appended to other volumes that are generated as a result of
	// StorageSpec objects.
	// +optional
	Clones []corev1.Volume `json:"clones,omitempty"`
}

// MSSQLServerStatus defines the observed state of MSSQLServer
type MSSQLServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MSSQLServer is the Schema for the mssqlservers API
type MSSQLServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MSSQLServerSpec   `json:"spec,omitempty"`
	Status MSSQLServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MSSQLServerList contains a list of MSSQLServer
type MSSQLServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MSSQLServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MSSQLServer{}, &MSSQLServerList{})
}
