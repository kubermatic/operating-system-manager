/*
Copyright 2021 The Operating System Manager contributors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OperatingSystemProfileResourceName represents "Resource" defined in Kubernetes
	OperatingSystemProfileResourceName = "operatingsystemprofiles"

	// OperatingSystemProfileKindName represents "Kind" defined in Kubernetes
	OperatingSystemProfileKindName = "OperatingSystemProfile"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemProfile is the object that represents the OperatingSystemProfile
type OperatingSystemProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// OperatingSystemProfileSpec represents the operating system configuration spec.
	Spec OperatingSystemProfileSpec `json:"spec"`
}

// OperatingSystemProfileSpec represents the data in the newly created OperatingSystemProfile
type OperatingSystemProfileSpec struct {
	// OSType represent the operating system name e.g: ubuntu
	OSName OperatingSystem `json:"osName"`
	// OSVersion the version of the operating system
	OSVersion string `json:"osVersion"`
	// Version is the version of the operating System Profile
	// +kubebuilder:validation:Pattern=`v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`
	// SupportedCloudProviders represent the cloud providers that support the given operating system version
	SupportedCloudProviders []CloudProviderSpec `json:"supportedCloudProviders"`
	// Bootstrap config is used for initial configuration of machine and to fetch the kubernetes secret that contains the provisioning config.
	BootstrapConfig OSPConfig `json:"bootstrapConfig"`
	// Provisioning Config is used for provisioning the worker node.
	ProvisioningConfig OSPConfig `json:"provisioningConfig"`
}

type OSPConfig struct {
	// SupportedContainerRuntimes represents the container runtimes supported by the given OS
	SupportedContainerRuntimes []ContainerRuntimeSpec `json:"supportedContainerRuntimes,omitempty"`
	// Templates to be included in units and files
	Templates map[string]string `json:"templates,omitempty"`
	// Units a list of the systemd unit files which will run on the instance
	Units []Unit `json:"units,omitempty"`
	// Files is a list of files that should exist in the instance
	Files []File `json:"files,omitempty"`
	// CloudInitModules field contains the optional cloud-init modules which are supported by OSM
	// +optional
	CloudInitModules *CloudInitModule `json:"modules,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemProfileList is a list of OperatingSystemProfiles
type OperatingSystemProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OperatingSystemProfile `json:"items"`
}
