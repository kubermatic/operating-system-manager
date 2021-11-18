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
	AccessKeyID     types.ConfigVarString `json:"accessKeyId,omitempty"`
	SecretAccessKey types.ConfigVarString `json:"secretAccessKey,omitempty"`

	AssumeRoleARN        types.ConfigVarString `json:"assumeRoleARN,omitempty"`
	AssumeRoleExternalID types.ConfigVarString `json:"assumeRoleExternalID,omitempty"`

	Region             types.ConfigVarString   `json:"region"`
	AvailabilityZone   types.ConfigVarString   `json:"availabilityZone,omitempty"`
	VpcID              types.ConfigVarString   `json:"vpcId"`
	SubnetID           types.ConfigVarString   `json:"subnetId"`
	SecurityGroupIDs   []types.ConfigVarString `json:"securityGroupIDs,omitempty"`
	InstanceProfile    types.ConfigVarString   `json:"instanceProfile,omitempty"`
	InstanceType       types.ConfigVarString   `json:"instanceType,omitempty"`
	AMI                types.ConfigVarString   `json:"ami,omitempty"`
	DiskSize           int64                   `json:"diskSize"`
	DiskType           types.ConfigVarString   `json:"diskType,omitempty"`
	DiskIops           *int64                  `json:"diskIops,omitempty"`
	EBSVolumeEncrypted types.ConfigVarBool     `json:"ebsVolumeEncrypted"`
	Tags               map[string]string       `json:"tags,omitempty"`
	AssignPublicIP     *bool                   `json:"assignPublicIP,omitempty"`

	IsSpotInstance     *bool               `json:"isSpotInstance,omitempty"`
	SpotInstanceConfig *SpotInstanceConfig `json:"spotInstanceConfig,omitempty"`
}

type SpotInstanceConfig struct {
	MaxPrice             types.ConfigVarString `json:"maxPrice,omitempty"`
	PersistentRequest    types.ConfigVarBool   `json:"persistentRequest,omitempty"`
	InterruptionBehavior types.ConfigVarString `json:"interruptionBehavior,omitempty"`
}
