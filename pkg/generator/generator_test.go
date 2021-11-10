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

package generator

import (
	"testing"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/utils/pointer"
)

func TestDefaultCloudInitGenerator_Generate(t *testing.T) {
	testCases := []struct {
		name              string
		osc               *osmv1alpha1.OperatingSystemConfig
		expectedCloudInit []byte
	}{
		{
			name: "generated cloud-init for ubuntu",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
						{
							Path:        "/opt/bin/setup.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
					UserSSHKeys: []string{
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3",
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4",
					},
				},
			},
			expectedCloudInit: []byte(`#cloud-config

ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test.service'
  permissions: '0700'
  encoding: b64
  content: |
    IyEvYmluL2Jhc2gKICAgIHNldCAteGV1byBwaXBlZmFpbAogICAgY2xvdWQtaW5pdCBjbGVhbgogICAgY2xvdWQtaW5pdCBpbml0CiAgICBzeXN0ZW1jdGwgc3RhcnQgcHJvdmlzaW9uLnNlcnZpY2U=
- path: '/opt/bin/setup.service'
  permissions: '0700'
  encoding: b64
  content: |
    IyEvYmluL2Jhc2gKICAgIHNldCAteGV1byBwaXBlZmFpbAogICAgY2xvdWQtaW5pdCBjbGVhbgogICAgY2xvdWQtaW5pdCBpbml0CiAgICBzeXN0ZW1jdGwgc3RhcnQgcHJvdmlzaW9uLnNlcnZpY2U=
runcmd:
- systemctl restart test.service
- systemctl restart setup.service
- systemctl daemon-reload`),
		},
		{
			name: "generated cloud-init for ubuntu without a service",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
					UserSSHKeys: []string{
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3",
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4",
					},
				},
			},
			expectedCloudInit: []byte(`#cloud-config
		
ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test'
permissions: '0700'
encoding: b64
content: |
IyEvYmluL2Jhc2gKICAgIHNldCAteGV1byBwaXBlZmFpbAogICAgY2xvdWQtaW5pdCBjbGVhbgogICAgY2xvdWQtaW5pdCBpbml0CiAgICBzeXN0ZW1jdGwgc3RhcnQgcHJvdmlzaW9uLnNlcnZpY2U=
runcmd:
- systemctl daemon-reload`),
		},
		{
			name: "generated cloud-init for ubuntu without a service and ssh keys",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			expectedCloudInit: []byte(`#cloud-config
		
ssh_pwauth: no
ssh_authorized_keys:
write_files:
- path: '/opt/bin/test'
permissions: '0700'
encoding: b64
content: |
IyEvYmluL2Jhc2gKICAgIHNldCAteGV1byBwaXBlZmFpbAogICAgY2xvdWQtaW5pdCBjbGVhbgogICAgY2xvdWQtaW5pdCBpbml0CiAgICBzeXN0ZW1jdGwgc3RhcnQgcHJvdmlzaW9uLnNlcnZpY2U=
runcmd:
- systemctl daemon-reload`),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			generator := NewDefaultCloudInitGenerator("")
			cloudInit, err := generator.Generate(testCase.osc)
			if err != nil {
				t.Fatalf("failed to generate cloud-init configs: %v", err)
			}

			if string(cloudInit) == string(testCase.expectedCloudInit) {
				t.Fatal("unexpected generated cloud-init")
			}
		})
	}
}
