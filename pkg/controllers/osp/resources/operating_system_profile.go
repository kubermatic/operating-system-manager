package resources

import (
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	bootstrapBinContentTemplate = `
#!/bin/bash
set -xeuo pipefail
wget %s/%s.cfg --directory-prefix /etc/cloud/cloud.cfg.d/
cloud-init clean
cloud-init --file /etc/cloud/cloud.cfg.d/%s.cfg init
systemctl start provision.service
`

	bootstrapServiceContentTemplate = `
[Install]
WantedBy=multi-user.target

[Unit]
Requires=network-online.target
After=network-online.target
[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/opt/bin/bootstrap
`
)

// BootstrapOSP returns the default bootstrap script.
func BootstrapOSP(apiServerAddress string, md *clusterv1alpha1.MachineDeployment) *v1alpha1.OperatingSystemProfile {
	configFileName := fmt.Sprintf("%s-osc-bootstrap", md.Name)
	bootstrapBin := fmt.Sprintf(bootstrapBinContentTemplate, apiServerAddress, configFileName, configFileName)
	return &v1alpha1.OperatingSystemProfile{
		ObjectMeta: v1.ObjectMeta{
			Name: "BootstrapOSP",
		},
		Spec: v1alpha1.OperatingSystemProfileSpec{
			Files: []v1alpha1.File{
				{
					Path:        "/opt/bin/bootstrap",
					Permissions: pointer.Int32Ptr(0755),
					Content: v1alpha1.FileContent{
						Inline: &v1alpha1.FileContentInline{
							Data:     bootstrapBin,
							Encoding: "b64",
						},
					},
				},
				{
					Path:        "/etc/systemd/system/bootstrap.service",
					Permissions: pointer.Int32Ptr(0644),
					Content: v1alpha1.FileContent{
						Inline: &v1alpha1.FileContentInline{
							Data:     bootstrapServiceContentTemplate,
							Encoding: "b64",
						},
					},
				},
			},
		},
	}
}
