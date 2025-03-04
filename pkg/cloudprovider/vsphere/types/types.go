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
	TemplateVMName providerconfig.ConfigVarString `json:"templateVMName"`
	VMNetName      providerconfig.ConfigVarString `json:"vmNetName"`
	Username       providerconfig.ConfigVarString `json:"username"`
	Password       providerconfig.ConfigVarString `json:"password"`
	VSphereURL     providerconfig.ConfigVarString `json:"vsphereURL"`
	Datacenter     providerconfig.ConfigVarString `json:"datacenter"`
	Cluster        providerconfig.ConfigVarString `json:"cluster"`
	Folder         providerconfig.ConfigVarString `json:"folder"`

	// Either Datastore or DatastoreCluster have to be provided.
	DatastoreCluster providerconfig.ConfigVarString `json:"datastoreCluster"`
	Datastore        providerconfig.ConfigVarString `json:"datastore"`

	AllowInsecure providerconfig.ConfigVarBool `json:"allowInsecure"`
}
