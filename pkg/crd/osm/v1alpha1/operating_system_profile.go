/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// OperatingSystemProfileResourceName represents "Resource" defined in Kubernetes
	OperatingSystemProfileResourceName = "operatingsystemprofile"

	// OperatingSystemProfileKindName represents "Kind" defined in Kubernetes
	OperatingSystemProfileKindName = "OperatingSystemProfile"
)

//+genclient
//+genclient:Namespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemProfile is the object that represents the OperatingSystemProfile
type OperatingSystemProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// OperatingSystemProfileSpec represents the operating system confuration spec.
	Spec OperatingSystemProfileSpec `json:"spec"`
}

// OperatingSystemProfileSpec represents the data in the newly created OperatingSystemProfile
type OperatingSystemProfileSpec struct {
	// OSType represent the operating system name e.g: ubuntu
	OSName string `json:"osName"`
	// OSVersion the version of the operating system
	OSVersion string `json:"osVersion"`
	// SupportedCloudProviders represent the cloud providers that support the given operating system version
	SupportedCloudProviders []SupportedCloudProvider `json:"supportedCloudProviders"`
	// Units a list of the systemd unit files which will run on the instance
	Units []Unit `json:"units,omitempty"`
	// Files is a list of files that should exist in the instance
	Files []File `json:"files,omitempty"`
}

// SupportedCloudProvider
type SupportedCloudProvider struct {
	// Name represents the name of the supported cloud provider
	Name string `json:"name"`
	// Spec represents the os/image reference in the supported cloud provider
	Spec runtime.RawExtension `json:"spec"`
}

// Unit is a systemd unit used for the operating system config.
type Unit struct {
	// Name is the name of a unit.
	Name string `json:"name"`
	// Enable describes whether the unit is enabled or not.
	Enable *bool `json:"enable,omitempty"`
	// Content is the unit's content.
	Content *string `json:"content,omitempty"`
	// DropIns is a list of drop-ins for this unit.
	DropIns []DropIn `json:"dropIns,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

// DropIn is a drop-in configuration for a systemd unit.
type DropIn struct {
	// Name is the name of the drop-in.
	Name string `json:"name"`
	// Content is the content of the drop-in.
	Content string `json:"content"`
}

// File is a file that should get written to the host's file system. The content can either be inlined or
// referenced from a secret in the same namespace.
type File struct {
	// Path is the path of the file system where the file should get written to.
	Path string `json:"path"`
	// Permissions describes with which permissions the file should get written to the file system.
	// Should be defaulted to octal 0644.
	Permissions *int32 `json:"permissions,omitempty"`
	// Content describe the file's content.
	Content FileContent `json:"content"`
}

// FileContent can either reference a secret or contain inline configuration.
type FileContent struct {
	// Inline is a struct that contains information about the inlined data.
	Inline *FileContentInline `json:"inline,omitempty"`
}

// FileContentInline contains keys for inlining a file content's data and encoding.
type FileContentInline struct {
	// Encoding is the file's encoding (e.g. base64).
	Encoding string `json:"encoding"`
	// Data is the file's data.
	Data string `json:"data"`
}
