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

package resrources

import (
	"encoding/json"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
)

type CloudInitSecret string

const (
	BootstrapCloudInit    CloudInitSecret = "bootstrap"
	ProvisioningCloudInit CloudInitSecret = "provisioning"

	MachineDeploymentOSPAnnotation = "k8c.io/operating-system-profile"
)

func OperatingSystemConfigCreator(provision bool, md *v1alpha1.MachineDeployment, osp *osmv1alpha1.OperatingSystemProfile) reconciling.NamedOperatingSystemConfigCreatorGetter {
	return func() (string, reconciling.OperatingSystemConfigCreator) {
		var oscName string
		if provision {
			oscName = fmt.Sprintf("%s-osc-%s", md.Name, ProvisioningCloudInit)
		} else {
			oscName = fmt.Sprintf("%s-osc-%s", md.Name, BootstrapCloudInit)
		}

		return oscName, func(osc *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error) {
			userSSHKeys := struct {
				SSHPublicKeys []string `json:"sshPublicKeys"`
			}{}
			if err := json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &userSSHKeys); err != nil {
				return nil, fmt.Errorf("failed to get user ssh keys: %v", err)
			}

			ospOriginal := osp.DeepCopy()
			osc.Spec = osmv1alpha1.OperatingSystemConfigSpec{
				OSName:      ospOriginal.Spec.OSName,
				OSVersion:   ospOriginal.Spec.OSVersion,
				Units:       ospOriginal.Spec.Units,
				Files:       ospOriginal.Spec.Files,
				UserSSHKeys: userSSHKeys.SSHPublicKeys,
			}

			return osc, nil
		}
	}
}
