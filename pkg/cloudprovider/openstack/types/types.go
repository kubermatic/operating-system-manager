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

import "k8c.io/machine-controller/sdk/providerconfig"

type RawConfig struct {
	// Auth details
	IdentityEndpoint            providerconfig.ConfigVarString `json:"identityEndpoint,omitempty"`
	Username                    providerconfig.ConfigVarString `json:"username,omitempty"`
	Password                    providerconfig.ConfigVarString `json:"password,omitempty"`
	ApplicationCredentialID     providerconfig.ConfigVarString `json:"applicationCredentialID,omitempty"`
	ApplicationCredentialSecret providerconfig.ConfigVarString `json:"applicationCredentialSecret,omitempty"`
	DomainName                  providerconfig.ConfigVarString `json:"domainName,omitempty"`
	ProjectName                 providerconfig.ConfigVarString `json:"projectName,omitempty"`
	ProjectID                   providerconfig.ConfigVarString `json:"projectID,omitempty"`
	TenantName                  providerconfig.ConfigVarString `json:"tenantName,omitempty"`
	TenantID                    providerconfig.ConfigVarString `json:"tenantID,omitempty"`
	TokenID                     providerconfig.ConfigVarString `json:"tokenId,omitempty"`
	Region                      providerconfig.ConfigVarString `json:"region,omitempty"`

	TrustDevicePath       providerconfig.ConfigVarBool `json:"trustDevicePath"`
	NodeVolumeAttachLimit *uint                        `json:"nodeVolumeAttachLimit"`
}
