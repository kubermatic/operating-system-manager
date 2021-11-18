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

type RawConfig struct {
	AccessKeyID     config.ConfigVarString `json:"accessKeyId,omitempty"`
	SecretAccessKey config.ConfigVarString `json:"secretAccessKey,omitempty"`

	AssumeRoleARN        config.ConfigVarString `json:"assumeRoleARN,omitempty"`
	AssumeRoleExternalID config.ConfigVarString `json:"assumeRoleExternalID,omitempty"`

	Region             config.ConfigVarString   `json:"region"`
	AvailabilityZone   config.ConfigVarString   `json:"availabilityZone,omitempty"`
	VpcID              config.ConfigVarString   `json:"vpcId"`
	SubnetID           config.ConfigVarString   `json:"subnetId"`
	SecurityGroupIDs   []config.ConfigVarString `json:"securityGroupIDs,omitempty"`
	InstanceProfile    config.ConfigVarString   `json:"instanceProfile,omitempty"`
	InstanceType       config.ConfigVarString   `json:"instanceType,omitempty"`
	AMI                config.ConfigVarString   `json:"ami,omitempty"`
	DiskSize           int64                    `json:"diskSize"`
	DiskType           config.ConfigVarString   `json:"diskType,omitempty"`
	DiskIops           *int64                   `json:"diskIops,omitempty"`
	EBSVolumeEncrypted config.ConfigVarBool     `json:"ebsVolumeEncrypted"`
	Tags               map[string]string        `json:"tags,omitempty"`
	AssignPublicIP     *bool                    `json:"assignPublicIP,omitempty"`

	IsSpotInstance     *bool               `json:"isSpotInstance,omitempty"`
	SpotInstanceConfig *SpotInstanceConfig `json:"spotInstanceConfig,omitempty"`
}

type SpotInstanceConfig struct {
	MaxPrice             config.ConfigVarString `json:"maxPrice,omitempty"`
	PersistentRequest    config.ConfigVarBool   `json:"persistentRequest,omitempty"`
	InterruptionBehavior config.ConfigVarString `json:"interruptionBehavior,omitempty"`
}
