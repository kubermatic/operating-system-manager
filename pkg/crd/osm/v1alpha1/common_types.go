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
	"k8s.io/apimachinery/pkg/runtime"
)

// OperatingSystem represents a supported operating system.
type OperatingSystem string

const (
	OperatingSystemFlatcar      OperatingSystem = "flatcar"
	OperatingSystemRHEL         OperatingSystem = "rhel"
	OperatingSystemCentOS       OperatingSystem = "centos"
	OperatingSystemUbuntu       OperatingSystem = "ubuntu"
	OperatingSystemSLES         OperatingSystem = "sles"
	OperatingSystemAmazonLinux2 OperatingSystem = "amzn2"
)

// CloudProviderSpec contains the os/image reference for a specific supported cloud provider
type CloudProviderSpec struct {
	// Name represents the name of the supported cloud provider
	Name string `json:"name"`
	// Spec represents the os/image reference in the supported cloud provider
	// +kubebuilder:pruning:PreserveUnknownFields
	Spec runtime.RawExtension `json:"spec,omitempty"`
}

// Unit is a systemd unit used for the operating system config.
type Unit struct {
	// Name is the name of a unit.
	Name string `json:"name"`
	// Enable describes whether the unit is enabled or not.
	Enable *bool `json:"enable,omitempty"`
	// Mask describes whether the unit is masked or not.
	Mask *bool `json:"mask,omitempty"`
	// Content is the unit's content.
	Content *string `json:"content,omitempty"`
	// DropIns is a list of drop-ins for this unit.
	DropIns []DropIn `json:"dropIns,omitempty"`
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

// ContainerRuntimeSpec aggregates information about a specific container runtime
type ContainerRuntimeSpec struct {
	// Name of the Container runtime
	Name string `json:"name"`
	// Files to add to the main files list when the containerRuntime is selected
	Files []File `json:"files"`
	// Templates to add to the available templates when the containerRuntime is selected
	Templates map[string]string `json:"templates,omitempty"`
}

// FileContent can either reference a secret or contain inline configuration.
type FileContent struct {
	// Inline is a struct that contains information about the inlined data.
	Inline *FileContentInline `json:"inline,omitempty"`
}

// FileContentInline contains keys for inlining a file content's data and encoding.
type FileContentInline struct {
	// Encoding is the file's encoding (e.g. base64).
	Encoding string `json:"encoding,omitempty"`
	// Data is the file's data.
	Data string `json:"data"`
}
