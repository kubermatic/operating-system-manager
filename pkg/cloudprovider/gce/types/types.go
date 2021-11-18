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
)

// RawConfig is a direct representation of an GCE machine object's configuration
type RawConfig struct {
	ServiceAccount        types.ConfigVarString `json:"serviceAccount,omitempty"`
	Zone                  types.ConfigVarString `json:"zone"`
	MachineType           types.ConfigVarString `json:"machineType"`
	DiskSize              int64                 `json:"diskSize"`
	DiskType              types.ConfigVarString `json:"diskType"`
	Network               types.ConfigVarString `json:"network"`
	Subnetwork            types.ConfigVarString `json:"subnetwork"`
	Preemptible           types.ConfigVarBool   `json:"preemptible"`
	Labels                map[string]string     `json:"labels,omitempty"`
	Tags                  []string              `json:"tags,omitempty"`
	AssignPublicIPAddress *types.ConfigVarBool  `json:"assignPublicIPAddress,omitempty"`
	MultiZone             types.ConfigVarBool   `json:"multizone"`
	Regional              types.ConfigVarBool   `json:"regional"`
	CustomImage           types.ConfigVarString `json:"customImage,omitempty"`
}
