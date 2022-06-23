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

func TestDefaultCloudConfigGenerator_Generate(t *testing.T) {
	testCases := []struct {
		name                string
		osc                 *osmv1alpha1.OperatingSystemConfig
		expectedCloudConfig []byte
	}{
		{
			name: "generated cloud-init for ubuntu",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "azure",
					},
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
						CloudInitModules: &osmv1alpha1.CloudInitModule{
							BootCMD:        []string{"echo hello-world", "echo hello-osm"},
							RHSubscription: map[string]string{"username": "test_username", "password": "test_password"},
							RunCMD:         []string{"systemctl restart test.service", "systemctl restart setup.service", "systemctl daemon-reload"},
						},
					},
				},
			},
			expectedCloudConfig: []byte(`#cloud-config
hostname: <MACHINE_NAME>

ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

- path: '/opt/bin/setup.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

bootcmd:
- echo hello-world
- echo hello-osm

runcmd:
- systemctl restart test.service
- systemctl restart setup.service
- systemctl daemon-reload

rh_subscription:
    password: test_password
    username: test_username
`),
		},
		{
			name: "generated cloud-init for ubuntu on aws",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "aws",
					},
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
						CloudInitModules: &osmv1alpha1.CloudInitModule{
							BootCMD:        []string{"echo hello-world", "echo hello-osm"},
							RHSubscription: map[string]string{"username": "test_username", "password": "test_password"},
							RunCMD:         []string{"systemctl restart test.service", "systemctl restart setup.service", "systemctl daemon-reload"},
						},
					},
				},
			},
			expectedCloudConfig: []byte(`#cloud-config
ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

- path: '/opt/bin/setup.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

bootcmd:
- echo hello-world
- echo hello-osm

runcmd:
- systemctl restart test.service
- systemctl restart setup.service
- systemctl daemon-reload

rh_subscription:
    password: test_password
    username: test_username
`),
		},
		{
			name: "generated cloud-init for ubuntu without a service",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "azure",
					},
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
						CloudInitModules: &osmv1alpha1.CloudInitModule{
							RunCMD: []string{"systemctl daemon-reload"},
						},
					},
				},
			},
			expectedCloudConfig: []byte(`#cloud-config
hostname: <MACHINE_NAME>

ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

runcmd:
- systemctl daemon-reload
`),
		},
		{
			name: "generated cloud-init for ubuntu without a service and ssh keys",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "ubuntu",
					OSVersion: "20.04",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "azure",
					},
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
						CloudInitModules: &osmv1alpha1.CloudInitModule{
							RunCMD: []string{"systemctl daemon-reload"},
						},
					},
				},
			},
			expectedCloudConfig: []byte(`#cloud-config
hostname: <MACHINE_NAME>

ssh_pwauth: no
ssh_authorized_keys:
write_files:
- path: '/opt/bin/test'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

runcmd:
- systemctl daemon-reload
`),
		},
		{
			name: "generated ignition config for flatcar for aws",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName: "flatcar",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "aws",
					},
					OSVersion: "2605.22.1",
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
			},
			expectedCloudConfig: []byte(`{"ignition":{"config":{},"security":{"tls":{}},"timeouts":{},"version":"2.3.0"},"networkd":{},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3","ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4"]}]},"storage":{"files":[{"filesystem":"root","path":"/opt/bin/test.service","contents":{"source":"data:,%23!%2Fbin%2Fbash%0Aset%20-xeuo%20pipefail%0Acloud-init%20clean%0Acloud-init%20init%0Asystemctl%20start%20provision.service%0A","verification":{}},"mode":448},{"filesystem":"root","path":"/opt/bin/setup.service","contents":{"source":"data:,%23!%2Fbin%2Fbash%0Aset%20-xeuo%20pipefail%0Acloud-init%20clean%0Acloud-init%20init%0Asystemctl%20start%20provision.service%0A","verification":{}},"mode":448}]},"systemd":{}}`),
		},
		{
			name: "generated ignition config for flatcar for azure",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName: "flatcar",
					CloudProvider: osmv1alpha1.CloudProviderSpec{
						Name: "azure",
					},
					OSVersion: "2605.22.1",
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
			},
			expectedCloudConfig: []byte(`{"ignition":{"config":{},"security":{"tls":{}},"timeouts":{},"version":"2.3.0"},"networkd":{},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3","ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4"]}]},"storage":{"files":[{"filesystem":"root","path":"/etc/hostname","contents":{"source":"data:,%3CMACHINE_NAME%3E","verification":{}},"mode":384},{"filesystem":"root","path":"/opt/bin/test.service","contents":{"source":"data:,%23!%2Fbin%2Fbash%0Aset%20-xeuo%20pipefail%0Acloud-init%20clean%0Acloud-init%20init%0Asystemctl%20start%20provision.service%0A","verification":{}},"mode":448},{"filesystem":"root","path":"/opt/bin/setup.service","contents":{"source":"data:,%23!%2Fbin%2Fbash%0Aset%20-xeuo%20pipefail%0Acloud-init%20clean%0Acloud-init%20init%0Asystemctl%20start%20provision.service%0A","verification":{}},"mode":448}]},"systemd":{}}`),
		},
		{
			name: "generated cloud-init modules for rhel",
			osc: &osmv1alpha1.OperatingSystemConfig{
				Spec: osmv1alpha1.OperatingSystemConfigSpec{
					OSName:    "rhel",
					OSVersion: "8.5",
					ProvisioningConfig: osmv1alpha1.OSCConfig{
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
						CloudInitModules: &osmv1alpha1.CloudInitModule{
							BootCMD:        []string{"echo hello-world", "echo hello-osm"},
							RHSubscription: map[string]string{"username": "test_username", "password": "test_password"},
							RunCMD:         []string{"systemctl restart test.service", "systemctl restart setup.service", "systemctl daemon-reload"},
							YumRepoDir:     "/store/custom/yum.repos.d",
							YumRepos:       map[string]map[string]string{"cloud-init-daily": {"name": "@cloud-init", "baseurl": "https://k8c.io", "type": "rpm-md"}},
						},
					},
				},
			},
			expectedCloudConfig: []byte(`#cloud-config
hostname: <MACHINE_NAME>

ssh_pwauth: no
ssh_authorized_keys:
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR3'
- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR4'
write_files:
- path: '/opt/bin/test.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

- path: '/opt/bin/setup.service'
  permissions: '0700'
  content: |-
        #!/bin/bash
        set -xeuo pipefail
        cloud-init clean
        cloud-init init
        systemctl start provision.service

bootcmd:
- echo hello-world
- echo hello-osm

runcmd:
- systemctl restart test.service
- systemctl restart setup.service
- systemctl daemon-reload

rh_subscription:
    password: test_password
    username: test_username

yum_repos:
    cloud-init-daily:
       baseurl: https://k8c.io
       name: @cloud-init
       type: rpm-md

yum_repo_dir: /store/custom/yum.repos.d`),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			generator := NewDefaultCloudConfigGenerator("")
			userData, err := generator.Generate(&testCase.osc.Spec.ProvisioningConfig, testCase.osc.Spec.OSName, testCase.osc.Spec.CloudProvider.Name)
			if err != nil {
				t.Fatalf("failed to generate cloud config: %v", err)
			}

			if string(userData) != string(testCase.expectedCloudConfig) {
				t.Fatal("unexpected generated cloud config")
			}
		})
	}
}
