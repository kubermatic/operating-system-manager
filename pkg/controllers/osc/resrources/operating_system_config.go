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
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
)

type CloudInitSecret string

const (
	BootstrapCloudInit    CloudInitSecret = "bootstrap"
	ProvisioningCloudInit CloudInitSecret = "provisioning"

	MachineDeploymentOSPAnnotation = "k8c.io/operating-system-profile"

	cniVersion = "v0.8.7"
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

			cloudProvider, err := GetCloudProviderFromMachineDeployment(md)
			if err != nil {
				return nil, fmt.Errorf("failed to get cloud provider from machine deployment: %v", err)
			}

			ospOriginal := osp.DeepCopy()

			data := filesData{
				KubeletVersion: md.Spec.Template.Spec.Versions.Kubelet,
				CNIVersion:     cniVersion,
			}
			populatedFiles, err := populateFilesList(ospOriginal.Spec.Files, data)
			if err != nil {
				return nil, fmt.Errorf("failed to populate OSP file template %v:", err)
			}

			osc.Spec = osmv1alpha1.OperatingSystemConfigSpec{
				OSName:        ospOriginal.Spec.OSName,
				OSVersion:     ospOriginal.Spec.OSVersion,
				Units:         ospOriginal.Spec.Units,
				Files:         populatedFiles,
				CloudProvider: *cloudProvider,
				UserSSHKeys:   userSSHKeys.SSHPublicKeys,
			}

			return osc, nil
		}
	}
}

type filesData struct {
	KubeletVersion string
	CNIVersion     string
}

func populateFilesList(files []osmv1alpha1.File, d filesData) ([]osmv1alpha1.File, error) {
	pfiles := []osmv1alpha1.File{}
	for _, file := range files {
		content := file.Content.Inline.Data
		tmpl, err := template.New(file.Path).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP file [%s] template: %v", file.Path, err)
		}

		buff := bytes.Buffer{}
		if err := tmpl.Execute(&buff, &d); err != nil {
			return nil, err
		}
		pfile := file.DeepCopy()
		pfile.Content.Inline.Data = buff.String()
		pfiles = append(pfiles, *pfile)
	}

	return pfiles, nil
}
