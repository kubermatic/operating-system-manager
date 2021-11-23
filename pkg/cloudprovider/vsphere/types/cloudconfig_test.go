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
	"testing"
)

func TestCloudConfigToString(t *testing.T) {
	tests := []struct {
		name     string
		config   *CloudConfig
		expected string
	}{
		{
			name: "conversion-test",
			config: &CloudConfig{
				Global: GlobalOpts{
					User:         "username",
					Password:     "password",
					InsecureFlag: true,
					VCenterPort:  "9090",
					ClusterID:    "test",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: WorkspaceOpts{
					Datacenter:       "datacenter",
					VCenterIP:        "hostname",
					DefaultDatastore: "datastore",
					Folder:           "workingDir",
				},
				VirtualCenter: map[string]*VirtualCenterConfig{
					"hostname": {
						VCenterPort: "9090",
						Datacenters: "datacenter",
						User:        "username",
						Password:    "password",
					},
				},
			},
			expected: `[Global]
user              = "username"
password          = "password"
port              = "9090"
insecure-flag     = true

[Disk]
scsicontrollertype = "pvscsi"

[Workspace]
server            = "hostname"
datacenter        = "datacenter"
folder            = "workingDir"
default-datastore = "datastore"


[VirtualCenter "hostname"]
user = "username"
password = "password"
port = 9090
datacenters = "datacenter"

`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := test.config.ToString()
			if err != nil {
				t.Fatalf("failed to convert to string: %v", err)
			}

			if s != test.expected {
				t.Fatalf("output is not as expected")
			}
		})
	}
}
