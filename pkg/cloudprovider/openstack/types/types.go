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

type RawConfig struct {
	// Auth details
	IdentityEndpoint            types.ConfigVarString `json:"identityEndpoint,omitempty"`
	Username                    types.ConfigVarString `json:"username,omitempty"`
	Password                    types.ConfigVarString `json:"password,omitempty"`
	ApplicationCredentialID     types.ConfigVarString `json:"applicationCredentialID,omitempty"`
	ApplicationCredentialSecret types.ConfigVarString `json:"applicationCredentialSecret,omitempty"`
	DomainName                  types.ConfigVarString `json:"domainName,omitempty"`
	ProjectName                 types.ConfigVarString `json:"projectName,omitempty"`
	ProjectID                   types.ConfigVarString `json:"projectID,omitempty"`
	TenantName                  types.ConfigVarString `json:"tenantName,omitempty"`
	TenantID                    types.ConfigVarString `json:"tenantID,omitempty"`
	TokenID                     types.ConfigVarString `json:"tokenId,omitempty"`
	Region                      types.ConfigVarString `json:"region,omitempty"`

	TrustDevicePath       bool  `json:"trustDevicePath"`
	NodeVolumeAttachLimit *uint `json:"nodeVolumeAttachLimit"`
}
