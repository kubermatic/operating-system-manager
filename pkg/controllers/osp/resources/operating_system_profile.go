/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package resources

import (
	"encoding/json"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	oscresources "k8c.io/operating-system-manager/pkg/controllers/osc/resrources"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	bootstrapBinContentTemplate = `#!/bin/bash
set -xeuo pipefail
wget %s/%s.cfg --directory-prefix /etc/cloud/cloud.cfg.d/
cloud-init clean
cloud-init --file /etc/cloud/cloud.cfg.d/%s.cfg init
systemctl start provision.service`

	bootstrapServiceContentTemplate = `[Install]
WantedBy=multi-user.target

[Unit]
Requires=network-online.target
After=network-online.target
[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/opt/bin/bootstrap`

	BootstrapOSPName = "BootstrapOSP"
)

// BootstrapOSP returns the default bootstrap script.
func BootstrapOSP(apiServerAddress string, md *clusterv1alpha1.MachineDeployment) (*v1alpha1.OperatingSystemProfile, error) {
	configFileName := fmt.Sprintf("%s-osc-bootstrap", md.Name)
	bootstrapBin := fmt.Sprintf(bootstrapBinContentTemplate, apiServerAddress, configFileName, configFileName)
	osName := &struct {
		OperatingSystem string `json:"operatingSystem"`
	}{}
	if err := json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, osName); err != nil {
		return nil, fmt.Errorf("failed to get operating system from machine deployment: %v", err)
	}
	cloudProvider, err := oscresources.GetCloudProviderFromMachineDeployment(md)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud provider from machine deployment: %v", err)
	}

	return &v1alpha1.OperatingSystemProfile{
		ObjectMeta: v1.ObjectMeta{
			Name: BootstrapOSPName,
		},
		Spec: v1alpha1.OperatingSystemProfileSpec{
			OSName: osName.OperatingSystem,
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
			SupportedCloudProviders: []v1alpha1.CloudProviderSpec{
				*cloudProvider,
			},
		},
	}, nil
}
