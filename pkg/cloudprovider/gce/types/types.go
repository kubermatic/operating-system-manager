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

import "k8c.io/operating-system-manager/pkg/providerconfig/config/types"

// RawConfig is a direct representation of an GCE machine object's configuration
type RawConfig struct {
	ServiceAccount types.ConfigVarString `json:"serviceAccount,omitempty"`
	Zone           string                `json:"zone"`
	Network        string                `json:"network"`
	Subnetwork     string                `json:"subnetwork"`
	Tags           []string              `json:"tags,omitempty"`
	MultiZone      bool                  `json:"multizone"`
	Regional       bool                  `json:"regional"`
}
