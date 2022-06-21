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
	// OperatingSystemConfigResourceName represents "Resource" defined in Kubernetes
	OperatingSystemConfigResourceName = "operatingsystemconfigs"

	// OperatingSystemConfigKindName represents "Kind" defined in Kubernetes
	OperatingSystemConfigKindName = "OperatingSystemConfig"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemConfig is the object that represents the OperatingSystemConfig
type OperatingSystemConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// OperatingSystemConfigSpec represents the operating system configuration spec.
	Spec OperatingSystemConfigSpec `json:"spec"`
}

// OperatingSystemConfigSpec represents the data in the newly created OperatingSystemConfig
type OperatingSystemConfigSpec struct {
	// OSType represent the operating system name e.g: ubuntu
	OSName OperatingSystem `json:"osName"`
	// OSVersion the version of the operating system
	OSVersion string `json:"osVersion"`
	// CloudProvider represent the cloud provider that support the given operating system version
	CloudProvider CloudProviderSpec `json:"cloudProvider"`
	// Bootstrap config is used for initial configuration of machine and to fetch the kubernetes secret that contains the provisioning config.
	BootstrapConfig OSCConfig `json:"bootstrapConfig"`
	// Provisioning Config is used for provisioning the worker node.
	ProvisioningConfig OSCConfig `json:"provisioningConfig"`
}

type OSCConfig struct {
	// Units a list of the systemd unit files which will run on the instance
	Units []Unit `json:"units,omitempty"`
	// Files is a list of files that should exist in the instance
	Files []File `json:"files,omitempty"`
	// UserSSHKeys is a list of attached user ssh keys
	UserSSHKeys []string `json:"userSSHKeys,omitempty"`
	// CloudInitModules contains the supported cloud-init modules
	// +optional
	CloudInitModules *CloudInitModule `json:"modules,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemConfigList is a list of OperatingSystemConfigs
type OperatingSystemConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OperatingSystemConfig `json:"items"`
}
