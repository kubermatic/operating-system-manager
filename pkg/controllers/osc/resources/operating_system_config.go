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
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletv1b1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"
	kyaml "sigs.k8s.io/yaml"
)

type CloudConfigSecret string

const (
	ProvisioningCloudConfig CloudConfigSecret = "provisioning"

	MachineDeploymentSubresourceNamePattern = "%s-osc-%s"

	MachineDeploymentOSPAnnotation = "k8c.io/operating-system-profile"

	cniVersion = "v0.8.7"
)

func OperatingSystemConfigCreator(
	md *v1alpha1.MachineDeployment,
	osp *osmv1alpha1.OperatingSystemProfile,
	kubeconfig string,
	clusterDNSIPs []net.IP,
) reconciling.NamedOperatingSystemConfigCreatorGetter {
	return func() (string, reconciling.OperatingSystemConfigCreator) {
		var oscName = fmt.Sprintf(MachineDeploymentSubresourceNamePattern, md.Name, ProvisioningCloudConfig)

		return oscName, func(osc *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error) {
			ospOriginal := osp.DeepCopy()
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

			CACert, err := resources.GetCACert(kubeconfig)
			if err != nil {
				return nil, err
			}
			kubeletConfig, err := kubeletConfiguration("cluster.local", clusterDNSIPs, map[string]bool{"RotateKubeletServerCertificate": true})
			if err != nil {
				return nil, err
			}
			kubeconfigStr, err := resources.StringifyKubeconfig(kubeconfig)
			if err != nil {
				return nil, err
			}

			kubeletSystemdUnit, err := KubeletSystemdUnit("docker", md.Spec.Template.Spec.Versions.Kubelet, cloudProvider.Name, "node-1", clusterDNSIPs, false,
				"", nil, KubeletFlags())
			if err != nil {
				return nil, err
			}

			safeDownloadBinariesScript, err := SafeDownloadBinariesScript(md.Spec.Template.Spec.Versions.Kubelet)
			if err != nil {
				return nil, err
			}

			data := filesData{
				KubeletVersion:       md.Spec.Template.Spec.Versions.Kubelet,
				KubeletConfiguration: kubeletConfig,
				KubeletSystemdUnit:   kubeletSystemdUnit,
				CNIVersion:           cniVersion,
				ClusterDNSIPs:        clusterDNSIPs,
				KubernetesCACert:     CACert,
				Kubeconfig:           kubeconfigStr,
				ContainerRuntime:     "docker",
				CloudProviderName:    cloudProvider.Name,
				Hostname:             "Node-1", // FIX this shit
				ExtraKubeletFlags:    KubeletFlags(),

				SafeDownloadBinariesScript: safeDownloadBinariesScript,
			}

			populatedFiles, err := populateFilesList(ospOriginal.Spec.Files, data)
			if err != nil {
				return nil, fmt.Errorf("failed to populate OSP file template: %v", err)
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
	KubeletVersion       string
	KubeletConfiguration string
	KubeletSystemdUnit   string
	CNIVersion           string
	ClusterDNSIPs        []net.IP
	KubernetesCACert     string
	ServerAddress        string
	Kubeconfig           string
	ContainerRuntime     string
	CloudProviderName    string
	Hostname             string
	ExtraKubeletFlags    []string

	SafeDownloadBinariesScript string
}

func populateFilesList(files []osmv1alpha1.File, d filesData) ([]osmv1alpha1.File, error) {
	var pfiles []osmv1alpha1.File
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

// kubeletConfiguration returns marshaled kubelet.config.k8s.io/v1beta1 KubeletConfiguration
func kubeletConfiguration(clusterDomain string, clusterDNS []net.IP, featureGates map[string]bool) (string, error) {
	clusterDNSstr := make([]string, 0, len(clusterDNS))
	for _, ip := range clusterDNS {
		clusterDNSstr = append(clusterDNSstr, ip.String())
	}

	cfg := kubeletv1b1.KubeletConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeletConfiguration",
			APIVersion: kubeletv1b1.SchemeGroupVersion.String(),
		},
		Authentication: kubeletv1b1.KubeletAuthentication{
			X509: kubeletv1b1.KubeletX509Authentication{
				ClientCAFile: "/etc/kubernetes/pki/ca.crt",
			},
			Webhook: kubeletv1b1.KubeletWebhookAuthentication{
				Enabled: pointer.BoolPtr(true),
			},
			Anonymous: kubeletv1b1.KubeletAnonymousAuthentication{
				Enabled: pointer.BoolPtr(false),
			},
		},
		Authorization: kubeletv1b1.KubeletAuthorization{
			Mode: kubeletv1b1.KubeletAuthorizationModeWebhook,
		},
		CgroupDriver:          "systemd",
		ClusterDNS:            clusterDNSstr,
		ClusterDomain:         clusterDomain,
		FeatureGates:          featureGates,
		ProtectKernelDefaults: true,
		ReadOnlyPort:          0,
		RotateCertificates:    true,
		ServerTLSBootstrap:    true,
		StaticPodPath:         "/etc/kubernetes/manifests",
		KubeReserved:          map[string]string{"cpu": "200m", "memory": "200Mi", "ephemeral-storage": "1Gi"},
		SystemReserved:        map[string]string{"cpu": "200m", "memory": "200Mi", "ephemeral-storage": "1Gi"},
		VolumePluginDir:       "/var/lib/kubelet/volumeplugins",
	}

	buf, err := kyaml.Marshal(cfg)
	return string(buf), err
}

// KubeletSystemdUnit returns the systemd unit for the kubelet
func KubeletSystemdUnit(containerRuntime, kubeletVersion, cloudProvider, hostname string, dnsIPs []net.IP, external bool, pauseImage string, initialTaints []corev1.Taint, extraKubeletFlags []string) (string, error) {
	tmpl, err := template.New("kubelet-systemd-unit").Parse(kubeletSystemdUnitTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse kubelet-systemd-unit template: %v", err)
	}

	data := struct {
		ContainerRuntime  string
		KubeletVersion    string
		CloudProvider     string
		Hostname          string
		ClusterDNSIPs     []net.IP
		IsExternal        bool
		PauseImage        string
		InitialTaints     []corev1.Taint
		ExtraKubeletFlags []string
	}{
		ContainerRuntime:  containerRuntime,
		KubeletVersion:    kubeletVersion,
		CloudProvider:     cloudProvider,
		Hostname:          hostname,
		ClusterDNSIPs:     dnsIPs,
		IsExternal:        external,
		PauseImage:        pauseImage,
		InitialTaints:     initialTaints,
		ExtraKubeletFlags: extraKubeletFlags,
	}

	var buf strings.Builder
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute kubelet-systemd-unit template: %w", err)
	}

	return buf.String(), nil
}

func KubeletFlags() []string {
	return []string{
		"--container-runtime=docker",
		"--container-runtime-endpoint=unix:///var/run/dockershim.sock",
	}
}

func SafeDownloadBinariesScript(kubeVersion string) (string, error) {
	tmpl, err := template.New("download-binaries").Parse(safeDownloadBinariesTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse download-binaries template: %v", err)
	}

	const CNIVersion = "v0.8.7"

	// force v in case if it's not there
	if !strings.HasPrefix(kubeVersion, "v") {
		kubeVersion = "v" + kubeVersion
	}

	data := struct {
		KubeVersion string
		CNIVersion  string
	}{
		KubeVersion: kubeVersion,
		CNIVersion:  CNIVersion,
	}

	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute download-binaries template: %v", err)
	}

	return b.String(), nil
}

const (
	kubeletSystemdUnitTpl = `
[Unit]
After={{ .ContainerRuntime }}.service
Requires={{ .ContainerRuntime }}.service

Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs/home/

[Service]
Restart=always
StartLimitInterval=0
RestartSec=10
CPUAccounting=true
MemoryAccounting=true

Environment="PATH=/opt/bin:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin/"
EnvironmentFile=-/etc/environment

ExecStartPre=/bin/bash /opt/load-kernel-modules.sh
ExecStartPre=/bin/bash /opt/bin/setup_net_env.sh
ExecStart=/opt/bin/kubelet $KUBELET_EXTRA_ARGS \
  --bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
  --kubeconfig=/var/lib/kubelet/kubeconfig \
  --config=/etc/kubernetes/kubelet.conf \
  --network-plugin=cni \
  --cert-dir=/etc/kubernetes/pki \
  {{- if .IsExternal }}
  --cloud-provider=external \
  {{- end }}
  {{- if and (.Hostname) (ne .CloudProvider "aws") }}
  --hostname-override=%H \
  {{- end }}
  --dynamic-config-dir=/etc/kubernetes/dynamic-config-dir \
  --exit-on-lock-contention \
  --lock-file=/tmp/kubelet.lock \
  {{- if .PauseImage }}
  --pod-infra-container-image={{ .PauseImage }} \
  {{- end }}
  {{- if .InitialTaints }}
  --register-with-taints={{- .InitialTaints }} \
  {{- end }}
  {{- range .ExtraKubeletFlags }}
  {{ . }} \
  {{- end }}
  --node-ip ${KUBELET_NODE_IP}

[Install]
WantedBy=multi-user.target`

	safeDownloadBinariesTpl = `
{{- /*setup some common directories */ -}}
opt_bin=/opt/bin
cni_bin_dir=/opt/cni/bin

{{- /* create all the necessary dirs */}}
mkdir -p /etc/cni/net.d /etc/kubernetes/dynamic-config-dir /etc/kubernetes/manifests "$opt_bin" "$cni_bin_dir"

{{- /* HOST_ARCH can be defined outside of machine-controller (in kubeone for example) */}}
arch=${HOST_ARCH-}
if [ -z "$arch" ]
then
case $(uname -m) in
x86_64)
    arch="amd64"
    ;;
aarch64)
    arch="arm64"
    ;;
*)
    echo "unsupported CPU architecture, exiting"
    exit 1
    ;;
esac
fi

{{- /* # CNI variables */}}
CNI_VERSION="${CNI_VERSION:-{{ .CNIVersion }}}"
cni_base_url="https://github.com/containernetworking/plugins/releases/download/$CNI_VERSION"
cni_filename="cni-plugins-linux-$arch-$CNI_VERSION.tgz"

{{- /* download CNI */}}
curl -Lfo "$cni_bin_dir/$cni_filename" "$cni_base_url/$cni_filename"

{{- /* download CNI checksum */}}
cni_sum=$(curl -Lf "$cni_base_url/$cni_filename.sha256")
cd "$cni_bin_dir"

{{- /* verify CNI checksum */}}
sha256sum -c <<<"$cni_sum"

{{- /* unpack CNI */}}
tar xvf "$cni_filename"
rm -f "$cni_filename"
cd -

{{- /* kubelet */}}
KUBE_VERSION="${KUBE_VERSION:-{{ .KubeVersion }}}"
kube_dir="$opt_bin/kubernetes-$KUBE_VERSION"
kube_base_url="https://storage.googleapis.com/kubernetes-release/release/$KUBE_VERSION/bin/linux/$arch"
kube_sum_file="$kube_dir/sha256"

{{- /* create versioned kube dir */}}
mkdir -p "$kube_dir"
: >"$kube_sum_file"

for bin in kubelet kubeadm kubectl; do
    {{- /* download kube binary */}}
    curl -Lfo "$kube_dir/$bin" "$kube_base_url/$bin"
    chmod +x "$kube_dir/$bin"

    {{- /* download kube binary checksum */}}
    sum=$(curl -Lf "$kube_base_url/$bin.sha256")

    {{- /* save kube binary checksum */}}
    echo "$sum  $kube_dir/$bin" >>"$kube_sum_file"
done

{{- /* check kube binaries checksum */}}
sha256sum -c "$kube_sum_file"

for bin in kubelet kubeadm kubectl; do
    {{- /* link kube binaries from verioned dir to $opt_bin */}}
    ln -sf "$kube_dir/$bin" "$opt_bin"/$bin
done

if [[ ! -x /opt/bin/health-monitor.sh ]]; then
    curl -Lfo /opt/bin/health-monitor.sh https://raw.githubusercontent.com/kubermatic/machine-controller/7967a0af2b75f29ad2ab227eeaa26ea7b0f2fbde/pkg/userdata/scripts/health-monitor.sh
    chmod +x /opt/bin/health-monitor.sh
fi
`
)
