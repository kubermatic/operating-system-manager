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
		expectedCloudInit string
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
									Data: "#!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
						{
							Path:        "/opt/bin/setup.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "#!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
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
			expectedCloudInit: `
#cloud-config

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
- systemctl daemon-reload
`,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			generator, err := NewDefaultCloudInitGenerator("")
			if err != nil {
				t.Fatalf("failed to create cloud-init generator: %v", err)
			}

			cloudInit, err := generator.Generate(testCase.osc)
			if err != nil {
				t.Fatalf("failed to generate cloud-init configs: %v", err)
			}

			if string(cloudInit) != (testCase.expectedCloudInit) {
				t.Fatal("unexpected generated cloud-init")
			}
		})
	}
}
