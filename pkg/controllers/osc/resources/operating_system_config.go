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
	"time"

	"github.com/Masterminds/semver/v3"

	mcsdkcommon "k8c.io/machine-controller/sdk/apis/cluster/common"
	"k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	mcbootstrap "k8c.io/machine-controller/sdk/bootstrap"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/cloudprovider"
	"k8c.io/operating-system-manager/pkg/containerruntime"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"
	fm "k8c.io/operating-system-manager/pkg/util/funcmap"
	jsonutil "k8c.io/operating-system-manager/pkg/util/json"
	kubeconfigutil "k8c.io/operating-system-manager/pkg/util/kubeconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

const (
	ProvisioningCloudConfig mcbootstrap.CloudConfigSecret = "provisioning"

	OperatingSystemConfigNamePattern        = "%s-%s-config"
	MachineDeploymentOSPAnnotation          = "k8c.io/operating-system-profile"
	MachineDeploymentOSPNamespaceAnnotation = "k8c.io/operating-system-profile-namespace"

	defaultFilePermissions = 644
)

// GenerateOperatingSystemConfig return an OperatingSystemConfig generated against the input data
func GenerateOperatingSystemConfig(
	md *v1alpha1.MachineDeployment,
	osp *osmv1alpha1.OperatingSystemProfile,
	bootstrapKubeconfig *clientcmdapi.Config,
	bootstrapKubeconfigSecretName string,
	apiServerToken string,
	oscName string,
	namespace string,
	caCert string,
	hostCACert string,
	clusterDNSIPs []net.IP,
	containerRuntime string,
	externalCloudProvider bool,
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
	providerConfig := providerconfig.Config{}
	if err = jsonutil.StrictUnmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig); err != nil {
		return nil, fmt.Errorf("failed to decode provider configs: %w", err)
	}

	networkIPFamily := providerConfig.Network.GetIPFamily()

	// Ensure that Kubelet version is prefixed by "v"
	kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
	if err != nil {
		return nil, fmt.Errorf("invalid kubelet version: %w", err)
	}

	kubeletVersionStr := kubeletVersion.String()
	if !strings.HasPrefix(kubeletVersionStr, "v") {
		kubeletVersionStr = fmt.Sprintf("v%s", kubeletVersionStr)
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

	crEngine := containerRuntimeConfig.Engine()
	crConfig, err := crEngine.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to generate container runtime config: %w", err)
	}

	crAuthConfig, err := crEngine.AuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate container runtime auth config: %w", err)
	}

	bootstrapKubeconfigString, err := kubeconfigutil.StringifyKubeconfig(bootstrapKubeconfig)
	if err != nil {
		return nil, err
	}
	provisioningSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, ProvisioningCloudConfig)

	var clusterName string
	for key := range bootstrapKubeconfig.Clusters {
		clusterName = key
		break
	}
	serverURL := bootstrapKubeconfig.Clusters[clusterName].Server

	bc := bootstrapConfig{
		Token:                         apiServerToken,
		SecretName:                    provisioningSecretName,
		ServerURL:                     serverURL,
		BootstrapKubeconfigSecretName: bootstrapKubeconfigSecretName,
	}

	inTreeCCM, external, err := cloudprovider.KubeletCloudProviderConfig(providerConfig.CloudProvider, externalCloudProvider)
	if err != nil {
		return nil, err
	}

	var cloudConfig string
	if providerConfig.OverwriteCloudConfig != nil {
		cloudConfig = *providerConfig.OverwriteCloudConfig
	} else {
		cloudConfig, err = cloudprovider.GetCloudConfig(external, providerConfig, md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch cloud-config: %w", err)
		}
	}

	data := filesData{
		KubeVersion:                kubeletVersionStr,
		ClusterDNSIPs:              clusterDNSIPs,
		KubernetesCACert:           caCert,
		HostCACert:                 hostCACert,
		InTreeCCMAvailable:         inTreeCCM,
		CloudConfig:                cloudConfig,
		ContainerRuntime:           containerRuntime,
		CloudProviderName:          osmv1alpha1.CloudProvider(providerConfig.CloudProvider),
		ExternalCloudProvider:      external,
		InitialTaints:              initialTaints,
		ContainerRuntimeConfig:     crConfig,
		ContainerRuntimeAuthConfig: crAuthConfig,
		KubeletFeatureGates:        kubeletFeatureGates,
		kubeletConfig:              kubeletConfigs,
		BootstrapKubeconfig:        bootstrapKubeconfigString,
		bootstrapConfig:            bc,
		NetworkIPFamily:            string(networkIPFamily),
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

	if providerConfig.Network.IsStaticIPConfig() && providerConfig.OperatingSystem != providerconfig.OperatingSystemFlatcar {
		return nil, fmt.Errorf("static IP config is not supported with: %s", providerConfig.OperatingSystem)
	}

	err = setOperatingSystemConfig(providerConfig.OperatingSystem, providerConfig.OperatingSystemSpec, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to add operating system spec: %w", err)
	}

	if providerConfig.OperatingSystem == providerconfig.OperatingSystemRHEL {
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
	BootstrapKubeconfig        string
	InTreeCCMAvailable         bool
	CNIVersion                 string
	ClusterDNSIPs              []net.IP
	KubernetesCACert           string
	HostCACert                 string
	ServerAddress              string
	CloudConfig                string
	ContainerRuntime           string
	CloudProviderName          osmv1alpha1.CloudProvider
	NetworkConfig              *providerconfig.NetworkConfig
	ExternalCloudProvider      bool
	InitialTaints              string
	HTTPProxy                  *string
	NoProxy                    *string
	ContainerRuntimeConfig     string
	ContainerRuntimeAuthConfig string
	KubeletFeatureGates        map[string]bool
	RHSubscription             map[string]string
	NetworkIPFamily            string

	kubeletConfig
	operatingSystemConfig
	bootstrapConfig
}

type operatingSystemConfig struct {
	AmazonLinuxConfig amzn2.Config
	FlatcarConfig     flatcar.Config
	RhelConfig        rhel.Config
	UbuntuConfig      ubuntu.Config
	RockyLinuxConfig  rockylinux.Config
}

type kubeletConfig struct {
	KubeReserved                *map[string]string
	SystemReserved              *map[string]string
	EvictionHard                *map[string]string
	MaxPods                     *int32
	ContainerLogMaxSize         *string
	ContainerLogMaxFiles        *string
	ImageGCHighThresholdPercent *int32
	ImageGCLowThresholdPercent  *int32
	ImageMinimumGCAge           *metav1.Duration
	ImageMaximumGCAge           *metav1.Duration
}

type bootstrapConfig struct {
	Token                         string
	ServerURL                     string
	SecretName                    string
	BootstrapKubeconfigSecretName string
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
	funcMap := fm.ExtraTxtFuncMap()
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

		if pfile.Permissions == 0 {
			pfile.Permissions = defaultFilePermissions
		}
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
	funcMap := fm.ExtraTxtFuncMap()

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

func setOperatingSystemConfig(os providerconfig.OperatingSystem, operatingSystemSpec runtime.RawExtension, data *filesData) error {
	switch os {
	case providerconfig.OperatingSystemAmazonLinux2:
		config, err := amzn2.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.AmazonLinuxConfig = *config
		return nil
	case providerconfig.OperatingSystemFlatcar:
		config, err := flatcar.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.FlatcarConfig = *config
		return nil
	case providerconfig.OperatingSystemRHEL:
		config, err := rhel.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.RhelConfig = *config
		return nil
	case providerconfig.OperatingSystemUbuntu:
		config, err := ubuntu.LoadConfig(operatingSystemSpec)
		if err != nil {
			return err
		}
		data.UbuntuConfig = *config
		return nil
	case providerconfig.OperatingSystemRockyLinux:
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

	kubeletConfigs := getKubeletConfigMap(annotations)
	if len(kubeletConfigs) == 0 {
		return cfg, nil
	}

	if val, ok := kubeletConfigs[mcsdkcommon.KubeReservedKubeletConfig]; ok {
		cfg.KubeReserved = getKeyValueMap(val, "=")
	}

	if val, ok := kubeletConfigs[mcsdkcommon.SystemReservedKubeletConfig]; ok {
		cfg.SystemReserved = getKeyValueMap(val, "=")
	}

	if val, ok := kubeletConfigs[mcsdkcommon.EvictionHardKubeletConfig]; ok {
		cfg.EvictionHard = getKeyValueMap(val, "<")
	}

	if val, ok := kubeletConfigs[mcsdkcommon.MaxPodsKubeletConfig]; ok {
		mp, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("parsing maxPods: %w", err)
		}

		cfg.MaxPods = ptr.To(int32(mp))
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ContainerLogMaxSizeKubeletConfig]; ok {
		cfg.ContainerLogMaxSize = &val
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ContainerLogMaxFilesKubeletConfig]; ok {
		cfg.ContainerLogMaxFiles = &val
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ImageGCHighThresholdPercent]; ok {
		mp, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("parsing imageGCHighThresholdPercent: %w", err)
		}
		cfg.ImageGCHighThresholdPercent = ptr.To(int32(mp))
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ImageGCLowThresholdPercent]; ok {
		mp, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("parsing imageGCLowThresholdPercent: %w", err)
		}
		cfg.ImageGCLowThresholdPercent = ptr.To(int32(mp))
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ImageMinimumGCAge]; ok {
		dur, err := time.ParseDuration(val)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("parsing imageMinimumGCAge: %w", err)
		}

		cfg.ImageMinimumGCAge = &metav1.Duration{Duration: dur}
	}

	if val, ok := kubeletConfigs[mcsdkcommon.ImageMaximumGCAge]; ok {
		dur, err := time.ParseDuration(val)
		if err != nil {
			return kubeletConfig{}, fmt.Errorf("parsing imageMaximumGCAge: %w", err)
		}

		cfg.ImageMaximumGCAge = &metav1.Duration{Duration: dur}
	}

	return cfg, nil
}

func getKubeletConfigMap(annotations map[string]string) map[string]string {
	configs := map[string]string{}
	for name, value := range annotations {
		if strings.HasPrefix(name, mcsdkcommon.KubeletConfigAnnotationPrefixV1) {
			nameConfigValue := strings.SplitN(name, "/", 2)
			if len(nameConfigValue) != 2 {
				continue
			}
			configs[nameConfigValue[1]] = value
		}
	}
	return configs
}

func getKeyValueMap(value string, kvDelimiter string) *map[string]string {
	res := make(map[string]string)

	for pair := range strings.SplitSeq(value, ",") {
		kvPair := strings.SplitN(pair, kvDelimiter, 2)
		if len(kvPair) != 2 {
			continue
		}
		res[kvPair[0]] = kvPair[1]
	}

	return &res
}
