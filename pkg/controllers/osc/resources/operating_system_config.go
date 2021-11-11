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

package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
)

type CloudInitSecret string

const (
	ProvisioningCloudInit CloudInitSecret = "provisioning"

	MachineDeploymentSubresourceNamePattern = "%s-osc-%s"
	MachineDeploymentOSPAnnotation          = "k8c.io/operating-system-profile"
)

func OperatingSystemConfigCreator(
	md *v1alpha1.MachineDeployment,
	osp *osmv1alpha1.OperatingSystemProfile,
	kubeconfig string,
	clusterDNSIPs string,
	containerRuntime string,
	externalCloudProvider bool,
	pauseImage string,
	initialTaints string,
	cniVersion string,
	containerdVersion string,
	cloudConfig string,
) reconciling.NamedOperatingSystemConfigCreatorGetter {
	return func() (string, reconciling.OperatingSystemConfigCreator) {
		var oscName = fmt.Sprintf(MachineDeploymentSubresourceNamePattern, md.Name, ProvisioningCloudInit)

		return oscName, func(osc *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error) {
			var (
				ospOriginal  = osp.DeepCopy()
				providerSpec = types.Config{}
			)

			if err := json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerSpec); err != nil {
				return nil, fmt.Errorf("failed to get user ssh keys: %v", err)
			}

			cloudProvider, err := GetCloudProviderFromMachineDeployment(md)
			if err != nil {
				return nil, fmt.Errorf("failed to get cloud provider from machine deployment: %v", err)
			}

			CACert, err := resources.GetCACert(kubeconfig)
			if err != nil {
				return nil, err
			}

			kubeconfigStr, err := resources.StringifyKubeconfig(kubeconfig)
			if err != nil {
				return nil, err
			}

			// ensure that Kubelet version is prefixed by "v"
			kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
			if err != nil {
				return nil, fmt.Errorf("invalid kubelet version: %w", err)
			}
			kubeletVersionStr := kubeletVersion.String()
			if !strings.HasPrefix(kubeletVersionStr, "v") {
				kubeletVersionStr = fmt.Sprintf("v%s", kubeletVersionStr)
			}

			data := filesData{
				KubeVersion:           kubeletVersionStr,
				CNIVersion:            cniVersion,
				ClusterDNSIPs:         clusterDNSIPs,
				KubernetesCACert:      CACert,
				Kubeconfig:            kubeconfigStr,
				CloudConfig:           cloudConfig,
				ContainerRuntime:      containerRuntime,
				ContainerdVersion:     containerdVersion,
				CloudProviderName:     providerSpec.CloudProvider,
				ExternalCloudProvider: externalCloudProvider,
				PauseImage:            pauseImage,
				InitialTaints:         initialTaints,
			}

			osp.Spec.Files = append(osp.Spec.Files, selectAdditionalFiles(osp, containerRuntime)...)
			additionalTemplates, err := selectAdditionalTemplates(osp, containerRuntime, data)
			if err != nil {
				return nil, fmt.Errorf("failed to add OSP templates: %v", err)
			}
			populatedFiles, err := populateFilesList(osp.Spec.Files, additionalTemplates, data)
			if err != nil {
				return nil, fmt.Errorf("failed to populate OSP file template: %v", err)
			}

			osc.Spec = osmv1alpha1.OperatingSystemConfigSpec{
				OSName:        ospOriginal.Spec.OSName,
				OSVersion:     ospOriginal.Spec.OSVersion,
				Units:         ospOriginal.Spec.Units,
				Files:         populatedFiles,
				CloudProvider: *cloudProvider,
				UserSSHKeys:   providerSpec.SSHPublicKeys,
			}

			return osc, nil
		}
	}
}

type filesData struct {
	KubeVersion           string
	KubeletConfiguration  string
	KubeletSystemdUnit    string
	CNIVersion            string
	ClusterDNSIPs         string
	KubernetesCACert      string
	ServerAddress         string
	Kubeconfig            string
	CloudConfig           string
	ContainerRuntime      string
	ContainerdVersion     string
	CloudProviderName     types.CloudProvider
	ExtraKubeletFlags     []string
	ExternalCloudProvider bool
	PauseImage            string
	InitialTaints         string
}

func populateFilesList(files []osmv1alpha1.File, additionalTemplates []string, d filesData) ([]osmv1alpha1.File, error) {
	var pfiles []osmv1alpha1.File
	for _, file := range files {
		content := file.Content.Inline.Data
		tmpl, err := template.New(file.Path).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP file [%s] template: %v", file.Path, err)
		}

		for _, at := range additionalTemplates {
			if tmpl, err = tmpl.Parse(at); err != nil {
				return nil, err
			}
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

func selectAdditionalFiles(osp *osmv1alpha1.OperatingSystemProfile, containerRuntime string) []osmv1alpha1.File {
	filesToAdd := make([]osmv1alpha1.File, 0)
	// select container runtime files
	for _, cr := range osp.Spec.SupportedContainerRuntimes {
		if cr.Name == containerRuntime {
			filesToAdd = append(filesToAdd, cr.Files...)
			break
		}
	}

	return filesToAdd
}

func selectAdditionalTemplates(osp *osmv1alpha1.OperatingSystemProfile, containerRuntime string, d filesData) ([]string, error) {
	templatesToRender := make(map[string]string)

	// select container runtime scripts
	for _, cr := range osp.Spec.SupportedContainerRuntimes {
		if cr.Name == containerRuntime {
			for name, temp := range cr.Templates {
				templatesToRender[name] = temp
			}
			break
		}
	}

	// select templates from templates field
	for name, temp := range osp.Spec.Templates {
		templatesToRender[name] = temp
	}

	templates := make([]string, 0)
	// render templates
	for name, t := range templatesToRender {
		tmpl, err := template.New(name).Parse(t)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP template [%s]: %v", name, err)
		}

		buff := bytes.Buffer{}
		if err := tmpl.Execute(&buff, &d); err != nil {
			return nil, err
		}
		templates = append(templates, addTemplatingSequence(name, buff.String()))
	}

	return templates, nil
}

func addTemplatingSequence(templateName, template string) string {
	return fmt.Sprintf("\n{{- define \"%s\" }}\n%s\n{{- end }}", templateName, template)
}
