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
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

const AzureCloudProvider = "AZUREPUBLICCLOUD"

// RawConfig is a direct representation of an Azure machine object's configuration
type RawConfig struct {
	SubscriptionID config.ConfigVarString `json:"subscriptionID,omitempty"`
	TenantID       config.ConfigVarString `json:"tenantID,omitempty"`
	ClientID       config.ConfigVarString `json:"clientID,omitempty"`
	ClientSecret   config.ConfigVarString `json:"clientSecret,omitempty"`

	Location              config.ConfigVarString `json:"location"`
	ResourceGroup         config.ConfigVarString `json:"resourceGroup"`
	VNetResourceGroup     config.ConfigVarString `json:"vnetResourceGroup"`
	VMSize                config.ConfigVarString `json:"vmSize"`
	VNetName              config.ConfigVarString `json:"vnetName"`
	SubnetName            config.ConfigVarString `json:"subnetName"`
	LoadBalancerSku       config.ConfigVarString `json:"loadBalancerSku"`
	RouteTableName        config.ConfigVarString `json:"routeTableName"`
	AvailabilitySet       config.ConfigVarString `json:"availabilitySet"`
	AssignAvailabilitySet *bool                               `json:"assignAvailabilitySet"`
	SecurityGroupName     config.ConfigVarString `json:"securityGroupName"`
	Zones                 []string                            `json:"zones"`
	ImagePlan             *ImagePlan                          `json:"imagePlan,omitempty"`
	ImageReference        *ImageReference                     `json:"imageReference,omitempty"`

	ImageID        config.ConfigVarString `json:"imageID"`
	OSDiskSize     int32                               `json:"osDiskSize"`
	DataDiskSize   int32                               `json:"dataDiskSize"`
	AssignPublicIP config.ConfigVarBool   `json:"assignPublicIP"`
	Tags           map[string]string                   `json:"tags,omitempty"`
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
