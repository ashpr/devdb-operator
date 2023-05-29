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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MSSQLDatabaseSpec defines the desired state of MSSQLDatabase
type MSSQLDatabaseSpec struct {
	Server         string `json:"server,omitempty"`
	CloneDirectory string `json:"cloneDirectory,omitempty"`
}

// MSSQLDatabaseStatus defines the observed state of MSSQLDatabase
type MSSQLDatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MSSQLDatabase is the Schema for the mssqldatabases API
type MSSQLDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MSSQLDatabaseSpec   `json:"spec,omitempty"`
	Status MSSQLDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MSSQLDatabaseList contains a list of MSSQLDatabase
type MSSQLDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MSSQLDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MSSQLDatabase{}, &MSSQLDatabaseList{})
}
