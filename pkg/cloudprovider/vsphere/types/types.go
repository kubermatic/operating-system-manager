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

import "k8c.io/machine-controller/pkg/providerconfig/types"

type RawConfig struct {
	TemplateVMName types.ConfigVarString `json:"templateVMName"`
	VMNetName      types.ConfigVarString `json:"vmNetName"`
	Username       types.ConfigVarString `json:"username"`
	Password       types.ConfigVarString `json:"password"`
	VSphereURL     types.ConfigVarString `json:"vsphereURL"`
	Datacenter     types.ConfigVarString `json:"datacenter"`
	Cluster        types.ConfigVarString `json:"cluster"`
	Folder         types.ConfigVarString `json:"folder"`

	// Either Datastore or DatastoreCluster have to be provided.
	DatastoreCluster types.ConfigVarString `json:"datastoreCluster"`
	Datastore        types.ConfigVarString `json:"datastore"`

	AllowInsecure types.ConfigVarBool `json:"allowInsecure"`
}
