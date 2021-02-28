package resrources

import (
	"encoding/json"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
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

			osc.Spec = osmv1alpha1.OperatingSystemConfigSpec{
				OSName:      osp.Spec.OSName,
				OSVersion:   osp.Spec.OSVersion,
				Units:       osp.Spec.Units,
				Files:       osp.Spec.Files,
				UserSSHKeys: userSSHKeys.SSHPublicKeys,
			}

			return osc, nil
		}
	}
}
