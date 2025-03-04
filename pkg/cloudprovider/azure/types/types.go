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

const AzureCloudProvider = "AZUREPUBLICCLOUD"

// RawConfig is a direct representation of an Azure machine object's configuration
type RawConfig struct {
	SubscriptionID providerconfig.ConfigVarString `json:"subscriptionID,omitempty"`
	TenantID       providerconfig.ConfigVarString `json:"tenantID,omitempty"`
	ClientID       providerconfig.ConfigVarString `json:"clientID,omitempty"`
	ClientSecret   providerconfig.ConfigVarString `json:"clientSecret,omitempty"`

	Location              providerconfig.ConfigVarString `json:"location"`
	ResourceGroup         providerconfig.ConfigVarString `json:"resourceGroup"`
	VNetResourceGroup     providerconfig.ConfigVarString `json:"vnetResourceGroup"`
	VNetName              providerconfig.ConfigVarString `json:"vnetName"`
	SubnetName            providerconfig.ConfigVarString `json:"subnetName"`
	LoadBalancerSku       providerconfig.ConfigVarString `json:"loadBalancerSku"`
	RouteTableName        providerconfig.ConfigVarString `json:"routeTableName"`
	AvailabilitySet       providerconfig.ConfigVarString `json:"availabilitySet"`
	AssignAvailabilitySet *bool                          `json:"assignAvailabilitySet"`
	SecurityGroupName     providerconfig.ConfigVarString `json:"securityGroupName"`
}
