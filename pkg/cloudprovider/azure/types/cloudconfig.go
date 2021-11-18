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

type CloudConfig struct {
	Cloud           string `json:"cloud"`
	TenantID        string `json:"tenantId"`
	SubscriptionID  string `json:"subscriptionId"`
	AADClientID     string `json:"aadClientId"`
	AADClientSecret string `json:"aadClientSecret"`

	ResourceGroup              string `json:"resourceGroup"`
	Location                   string `json:"location"`
	VNetName                   string `json:"vnetName"`
	SubnetName                 string `json:"subnetName"`
	RouteTableName             string `json:"routeTableName"`
	SecurityGroupName          string `json:"securityGroupName" yaml:"securityGroupName"`
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName"`
	VnetResourceGroup          string `json:"vnetResourceGroup"`
	UseInstanceMetadata        bool   `json:"useInstanceMetadata"`
	LoadBalancerSku            string `json:"loadBalancerSku"`
}
