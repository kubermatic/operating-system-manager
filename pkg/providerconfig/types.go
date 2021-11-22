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

package providerconfig

import "k8s.io/apimachinery/pkg/runtime"

type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderAzure        CloudProvider = "azure"
	CloudProviderDigitalocean CloudProvider = "digitalocean"
	CloudProviderGoogle       CloudProvider = "gce"
	CloudProviderHetzner      CloudProvider = "hetzner"
	CloudProviderKubeVirt     CloudProvider = "kubevirt"
	CloudProviderLinode       CloudProvider = "linode"
	CloudProviderOpenstack    CloudProvider = "openstack"
	CloudProviderPacket       CloudProvider = "packet"
	CloudProviderVsphere      CloudProvider = "vsphere"
	CloudProviderFake         CloudProvider = "fake"
	CloudProviderAlibaba      CloudProvider = "alibaba"
	CloudProviderAnexia       CloudProvider = "anexia"
	CloudProviderScaleway     CloudProvider = "scaleway"
	CloudProviderBaremetal    CloudProvider = "baremetal"
	CloudProviderExternal     CloudProvider = "external"
)

type Config struct {
	SSHPublicKeys []string `json:"sshPublicKeys"`

	CloudProvider     CloudProvider        `json:"cloudProvider"`
	CloudProviderSpec runtime.RawExtension `json:"cloudProviderSpec"`
}
