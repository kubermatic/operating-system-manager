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
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/Masterminds/sprig/v3"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/common"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/cloudprovider"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/centos"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/sles"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"
	jsonutil "k8c.io/operating-system-manager/pkg/util/json"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CloudConfigSecret string

const (
	ProvisioningCloudConfig CloudConfigSecret = "provisioning"
	BootstrapCloudConfig    CloudConfigSecret = "bootstrap"

	OperatingSystemConfigNamePattern = "%s-%s-osc"
	CloudConfigSecretNamePattern     = "%s-%s-%s-secret"
	MachineDeploymentOSPAnnotation   = "k8c.io/operating-system-profile"
)

// GenerateOperatingSystemConfig return an OperatingSystemConfig generated against the input data
func GenerateOperatingSystemConfig(
	md *v1alpha1.MachineDeployment,
	osp *osmv1alpha1.OperatingSystemProfile,
	oscName string,
	namespace string,
	caCert string,
	clusterDNSIPs []net.IP,
	containerRuntime string,
	externalCloudProvider bool,
	pauseImage string,
	initialTaints string,
	nodeHTTPProxy string,
	nodeNoProxy string,
	containerRuntimeConfig containerruntime.Config,
	kubeletFeatureGates map[string]bool,
) (*osmv1alpha1.OperatingSystemConfig, error) {
	var err error
	ospOriginal := osp.DeepCopy()

	// Set metadata for OSC
	osc := &osmv1alpha1.OperatingSystemConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oscName,
			Namespace: namespace,
		},
	}

	// Get providerConfig from machineDeployment
	if len(md.Spec.Template.Spec.ProviderSpec.Value.Raw) == 0 {
		return nil, fmt.Errorf("providerSpec cannot be empty")
	}
	providerConfig := providerconfigtypes.Config{}
	if err = jsonutil.StrictUnmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig); err != nil {
		return nil, fmt.Errorf("failed to decode provider configs: %w", err)
	}

	var cloudConfig string
	if providerConfig.OverwriteCloudConfig != nil {
		cloudConfig = *providerConfig.OverwriteCloudConfig
	} else {
		cloudConfig, err = cloudprovider.GetCloudConfig(providerConfig, md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch cloud-config: %w", err)
		}
	}

	// Ensure that Kubelet version is prefixed by "v"
	kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
	if err != nil {
		return nil, fmt.Errorf("invalid kubelet version: %w", err)
	}

	kubeletVersionStr := kubeletVersion.String()
	if !strings.HasPrefix(kubeletVersionStr, "v") {
		kubeletVersionStr = fmt.Sprintf("v%s", kubeletVersionStr)
	}

	inTreeCCM, external, err := cloudprovider.KubeletCloudProviderConfig(providerConfig.CloudProvider)
	if err != nil {
		return nil, err
	}

	// Handling for kubelet configuration
	kubeletConfigs, err := getKubeletConfigs(md.Annotations)
	if err != nil {
		return nil, err
	}
	if kubeletConfigs.ContainerLogMaxSize != nil && len(*kubeletConfigs.ContainerLogMaxSize) > 0 {
		containerRuntimeConfig.ContainerLogMaxSize = *kubeletConfigs.ContainerLogMaxSize
	}

	if kubeletConfigs.ContainerLogMaxFiles != nil && len(*kubeletConfigs.ContainerLogMaxFiles) > 0 {
		containerRuntimeConfig.ContainerLogMaxFiles = *kubeletConfigs.ContainerLogMaxFiles
	}

	crEngine := containerRuntimeConfig.Engine(kubeletVersion)
	crConfig, err := crEngine.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to generate container runtime config: %w", err)
	}

	crAuthConfig, err := crEngine.AuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate container runtime auth config: %w", err)
	}

	if external {
		externalCloudProvider = true
	}

	data := filesData{
		KubeVersion:                kubeletVersionStr,
		ClusterDNSIPs:              clusterDNSIPs,
		KubernetesCACert:           caCert,
		InTreeCCMAvailable:         inTreeCCM,
		CloudConfig:                cloudConfig,
		ContainerRuntime:           containerRuntime,
		CloudProviderName:          osmv1alpha1.CloudProvider(providerConfig.CloudProvider),
		ExternalCloudProvider:      externalCloudProvider,
		PauseImage:                 pauseImage,
		InitialTaints:              initialTaints,
		ContainerRuntimeConfig:     crConfig,
		ContainerRuntimeAuthConfig: crAuthConfig,
		KubeletFeatureGates:        kubeletFeatureGates,
		kubeletConfig:              kubeletConfigs,
	}

	if len(nodeHTTPProxy) > 0 {
		data.HTTPProxy = &nodeHTTPProxy
	}
	if len(nodeNoProxy) > 0 {
		data.NoProxy = &nodeNoProxy
	}
	if providerConfig.Network != nil {
		data.NetworkConfig = providerConfig.Network
	}

	if providerConfig.Network.IsStaticIPConfig() && providerConfig.OperatingSystem != providerconfigtypes.OperatingSystemFlatcar {
		return nil, fmt.Errorf("static IP config is not supported with: %s", providerConfig.OperatingSystem)
	}

	err = setOperatingSystemConfig(providerConfig.OperatingSystem, providerConfig.OperatingSystemSpec, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to add operating system spec: %w", err)
	}

	if providerConfig.OperatingSystem == providerconfigtypes.OperatingSystemRHEL {
		rhSubscription := rhel.RHSubscription(data.RhelConfig)

		if osp.Spec.BootstrapConfig.CloudInitModules == nil {
			osp.Spec.BootstrapConfig.CloudInitModules = &osmv1alpha1.CloudInitModule{}
		}
		osp.Spec.BootstrapConfig.CloudInitModules.RHSubscription = rhSubscription

		if osp.Spec.ProvisioningConfig.CloudInitModules == nil {
			osp.Spec.ProvisioningConfig.CloudInitModules = &osmv1alpha1.CloudInitModule{}
		}
		osp.Spec.ProvisioningConfig.CloudInitModules.RHSubscription = rhSubscription
	}

	// Render files for bootstrapping config
	renderedBootstrappingFiles, err := renderedFiles(osp.Spec.BootstrapConfig, containerRuntime, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render bootstrapping file templates: %w", err)
	}
	// Render files for provisioning config
	renderedProvisioningFiles, err := renderedFiles(osp.Spec.ProvisioningConfig, containerRuntime, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render bootstrapping file templates: %w", err)
	}

	osc.Spec = osmv1alpha1.OperatingSystemConfigSpec{
		OSName:    ospOriginal.Spec.OSName,
		OSVersion: ospOriginal.Spec.OSVersion,
		CloudProvider: osmv1alpha1.CloudProviderSpec{
			Name: osmv1alpha1.CloudProvider(providerConfig.CloudProvider),
			Spec: providerConfig.CloudProviderSpec,
		},
		BootstrapConfig: osmv1alpha1.OSCConfig{
			Units:            ospOriginal.Spec.BootstrapConfig.Units,
			Files:            renderedBootstrappingFiles,
			UserSSHKeys:      providerConfig.SSHPublicKeys,
			CloudInitModules: osp.Spec.BootstrapConfig.CloudInitModules,
		},
		ProvisioningConfig: osmv1alpha1.OSCConfig{
			Units:            ospOriginal.Spec.ProvisioningConfig.Units,
			Files:            renderedProvisioningFiles,
			UserSSHKeys:      providerConfig.SSHPublicKeys,
			CloudInitModules: osp.Spec.ProvisioningConfig.CloudInitModules,
		},
	}
	return osc, nil
}

type filesData struct {
	KubeVersion                string
	KubeletConfiguration       string
	KubeletSystemdUnit         string
	InTreeCCMAvailable         bool
	CNIVersion                 string
	ClusterDNSIPs              []net.IP
	KubernetesCACert           string
	ServerAddress              string
	CloudConfig                string
	ContainerRuntime           string
	CloudProviderName          osmv1alpha1.CloudProvider
	NetworkConfig              *providerconfigtypes.NetworkConfig
	ExternalCloudProvider      bool
	PauseImage                 string
	InitialTaints              string
	HTTPProxy                  *string
	NoProxy                    *string
	ContainerRuntimeConfig     string
	ContainerRuntimeAuthConfig string
	KubeletFeatureGates        map[string]bool
	RHSubscription             map[string]string

	kubeletConfig
	OperatingSystemConfig
}

type OperatingSystemConfig struct {
	AmazonLinuxConfig amzn2.Config
	CentOSConfig      centos.Config
	FlatcarConfig     flatcar.Config
	RhelConfig        rhel.Config
	SlesConfig        sles.Config
	UbuntuConfig      ubuntu.Config
	RockyLinuxConfig  rockylinux.Config
}

type kubeletConfig struct {
	KubeReserved         *map[string]string
	SystemReserved       *map[string]string
	EvictionHard         *map[string]string
	MaxPods              *int32
	ContainerLogMaxSize  *string
	ContainerLogMaxFiles *string
}

func renderedFiles(config osmv1alpha1.OSPConfig, containerRuntime string, data filesData) ([]osmv1alpha1.File, error) {
	config.Files = append(config.Files, selectAdditionalFiles(config, containerRuntime)...)
	additionalTemplates, err := selectAdditionalTemplates(config, containerRuntime, data)
	if err != nil {
		return nil, fmt.Errorf("failed to add OSP templates: %w", err)
	}
	populatedFiles, err := populateFilesList(config.Files, additionalTemplates, data)
	if err != nil {
		return nil, fmt.Errorf("failed to populate OSP file template: %w", err)
	}
	return populatedFiles, nil
}

func populateFilesList(files []osmv1alpha1.File, additionalTemplates []string, d filesData) ([]osmv1alpha1.File, error) {
	funcMap := sprig.TxtFuncMap()
	var pfiles []osmv1alpha1.File
	for _, file := range files {
		content := file.Content.Inline.Data
		tmpl, err := template.New(file.Path).Funcs(funcMap).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP file [%s] template: %w", file.Path, err)
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

func selectAdditionalFiles(config osmv1alpha1.OSPConfig, containerRuntime string) []osmv1alpha1.File {
	filesToAdd := make([]osmv1alpha1.File, 0)
	// select container runtime files
	for _, cr := range config.SupportedContainerRuntimes {
		if cr.Name == osmv1alpha1.ContainerRuntime(containerRuntime) {
			filesToAdd = append(filesToAdd, cr.Files...)
			break
		}
	}

	return filesToAdd
}

func selectAdditionalTemplates(config osmv1alpha1.OSPConfig, containerRuntime string, d filesData) ([]string, error) {
	templatesToRender := make(map[string]string)

	// select container runtime scripts
	for _, cr := range config.SupportedContainerRuntimes {
		if cr.Name == osmv1alpha1.ContainerRuntime(containerRuntime) {
			for name, temp := range cr.Templates {
				templatesToRender[name] = temp
			}
			break
		}
	}

	// select templates from templates field
	for name, temp := range config.Templates {
		templatesToRender[name] = temp
	}

	templates := make([]string, 0)
	funcMap := sprig.TxtFuncMap()

	// render templates
	for name, t := range templatesToRender {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(t)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP template [%s]: %w", name, err)
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

func setOperatingSystemConfig(os providerconfigtypes.OperatingSystem, operatingSystemSpec runtime.RawExtension, data *filesData) error {
	switch os {
	case providerconfigtypes.OperatingSystemAmazonLinux2:
		config, err := amzn2.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.AmazonLinuxConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemCentOS:
		config, err := centos.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.CentOSConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemFlatcar:
		config, err := flatcar.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.FlatcarConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemRHEL:
		config, err := rhel.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.RhelConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemSLES:
		config, err := sles.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.SlesConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemUbuntu:
		config, err := ubuntu.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.UbuntuConfig = *config
		return nil
	case providerconfigtypes.OperatingSystemRockyLinux:
		config, err := rockylinux.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.RockyLinuxConfig = *config
		return nil
	}

	return errors.New("unknown OperatingSystem")
}

func getKubeletConfigs(annotations map[string]string) (kubeletConfig, error) {
	var cfg kubeletConfig
	kubeletConfigs := common.GetKubeletConfigs(annotations)
	if len(kubeletConfigs) == 0 {
		return cfg, nil
	}

	if val, ok := kubeletConfigs[common.KubeReservedKubeletConfig]; ok {
		cfg.KubeReserved = getKeyValueMap(val, "=")
	}

	if val, ok := kubeletConfigs[common.SystemReservedKubeletConfig]; ok {
		cfg.SystemReserved = getKeyValueMap(val, "=")
	}

	if val, ok := kubeletConfigs[common.EvictionHardKubeletConfig]; ok {
		cfg.EvictionHard = getKeyValueMap(val, "<")
	}

	if val, ok := kubeletConfigs[common.MaxPodsKubeletConfig]; ok {
		mp, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("failed to parse maxPods")
		}
		cfg.MaxPods = pointer.Int32Ptr(int32(mp))
	}

	if val, ok := kubeletConfigs[common.ContainerLogMaxSizeKubeletConfig]; ok {
		cfg.ContainerLogMaxSize = &val
	}

	if val, ok := kubeletConfigs[common.ContainerLogMaxFilesKubeletConfig]; ok {
		cfg.ContainerLogMaxFiles = &val
	}
	return cfg, nil
}

func getKeyValueMap(value string, kvDelimeter string) *map[string]string {
	res := make(map[string]string)
	for _, pair := range strings.Split(value, ",") {
		kvPair := strings.SplitN(pair, kvDelimeter, 2)
		if len(kvPair) != 2 {
			continue
		}
		res[kvPair[0]] = kvPair[1]
	}
	return &res
}
