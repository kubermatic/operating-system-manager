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

const AzureCloudProvider = "AZUREPUBLICCLOUD"

// RawConfig is a direct representation of an Azure machine object's configuration
type RawConfig struct {
	SubscriptionID types.ConfigVarString `json:"subscriptionID,omitempty"`
	TenantID       types.ConfigVarString `json:"tenantID,omitempty"`
	ClientID       types.ConfigVarString `json:"clientID,omitempty"`
	ClientSecret   types.ConfigVarString `json:"clientSecret,omitempty"`

	Location              types.ConfigVarString `json:"location"`
	ResourceGroup         types.ConfigVarString `json:"resourceGroup"`
	VNetResourceGroup     types.ConfigVarString `json:"vnetResourceGroup"`
	VNetName              types.ConfigVarString `json:"vnetName"`
	SubnetName            types.ConfigVarString `json:"subnetName"`
	LoadBalancerSku       types.ConfigVarString `json:"loadBalancerSku"`
	RouteTableName        types.ConfigVarString `json:"routeTableName"`
	AvailabilitySet       types.ConfigVarString `json:"availabilitySet"`
	AssignAvailabilitySet *bool                 `json:"assignAvailabilitySet"`
	SecurityGroupName     types.ConfigVarString `json:"securityGroupName"`
}
