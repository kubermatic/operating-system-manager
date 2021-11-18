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
	VMSize                types.ConfigVarString `json:"vmSize"`
	VNetName              types.ConfigVarString `json:"vnetName"`
	SubnetName            types.ConfigVarString `json:"subnetName"`
	LoadBalancerSku       types.ConfigVarString `json:"loadBalancerSku"`
	RouteTableName        types.ConfigVarString `json:"routeTableName"`
	AvailabilitySet       types.ConfigVarString `json:"availabilitySet"`
	AssignAvailabilitySet *bool                 `json:"assignAvailabilitySet"`
	SecurityGroupName     types.ConfigVarString `json:"securityGroupName"`
	Zones                 []string              `json:"zones"`
	ImagePlan             *ImagePlan            `json:"imagePlan,omitempty"`
	ImageReference        *ImageReference       `json:"imageReference,omitempty"`

	ImageID        types.ConfigVarString `json:"imageID"`
	OSDiskSize     int32                 `json:"osDiskSize"`
	DataDiskSize   int32                 `json:"dataDiskSize"`
	AssignPublicIP types.ConfigVarBool   `json:"assignPublicIP"`
	Tags           map[string]string     `json:"tags,omitempty"`
}

// ImagePlan contains azure OS Plan fields for the marketplace images
type ImagePlan struct {
	Name      string `json:"name,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Product   string `json:"product,omitempty"`
}

// ImageReference specifies information about the image to use.
type ImageReference struct {
	Publisher string `json:"publisher,omitempty"`
	Offer     string `json:"offer,omitempty"`
	Sku       string `json:"sku,omitempty"`
	Version   string `json:"version,omitempty"`
}
