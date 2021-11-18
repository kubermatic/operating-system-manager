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

package types

import (
	"k8c.io/operating-system-manager/pkg/providerconfig/config/types"

	corev1 "k8s.io/api/core/v1"
)

// RawConfig is a direct representation of an Kubevirt machine object's configuration
type RawConfig struct {
	Kubeconfig       types.ConfigVarString `json:"kubeconfig,omitempty"`
	CPUs             types.ConfigVarString `json:"cpus,omitempty"`
	Memory           types.ConfigVarString `json:"memory,omitempty"`
	Namespace        types.ConfigVarString `json:"namespace,omitempty"`
	SourceURL        types.ConfigVarString `json:"sourceURL,omitempty"`
	PVCSize          types.ConfigVarString `json:"pvcSize,omitempty"`
	StorageClassName types.ConfigVarString `json:"storageClassName,omitempty"`
	DNSPolicy        types.ConfigVarString `json:"dnsPolicy,omitempty"`
	DNSConfig        *corev1.PodDNSConfig  `json:"dnsConfig,omitempty"`
}
