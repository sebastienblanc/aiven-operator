// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClickhouseUserSpec defines the desired state of ClickhouseUser
type ClickhouseUserSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project to link the user to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service to link the user to
	ServiceName string `json:"serviceName"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication string `json:"authentication,omitempty"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef,omitempty"`
}

// ClickhouseUserStatus defines the observed state of ClickhouseUser
type ClickhouseUserStatus struct {
	// Clickhouse user UUID
	UUID string `json:"uuid"`

	// Conditions represent the latest available observations of an ClickhouseUser state
	// +kubebuilder:validation:type=array
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClickhouseUser is the Schema for the clickhouseusers API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connection Information Secret",type="string",JSONPath=".spec.connInfoSecretTarget.name"
type ClickhouseUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseUserSpec   `json:"spec,omitempty"`
	Status ClickhouseUserStatus `json:"status,omitempty"`
}

func (u ClickhouseUser) AuthSecretRef() AuthSecretReference {
	return u.Spec.AuthSecretRef
}

//+kubebuilder:object:root=true

// ClickhouseUserList contains a list of ClickhouseUser
type ClickhouseUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClickhouseUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClickhouseUser{}, &ClickhouseUserList{})
}
