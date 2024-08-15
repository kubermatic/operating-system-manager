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

package osc

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	mcbootstrap "github.com/kubermatic/machine-controller/pkg/bootstrap"
	cloudproviderutil "github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	machinecontrollerutil "github.com/kubermatic/machine-controller/pkg/controller/util"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/bootstrap"
	"k8c.io/operating-system-manager/pkg/clusterinfo"
	"k8c.io/operating-system-manager/pkg/containerruntime"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	testUtil "k8c.io/operating-system-manager/pkg/test/util"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const dummyCACert = `-----BEGIN CERTIFICATE-----
MIIEWjCCA0KgAwIBAgIJALfRlWsI8YQHMA0GCSqGSIb3DQEBBQUAMHsxCzAJBgNV
BAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNjbzEUMBIG
A1UEChMLQnJhZGZpdHppbmMxEjAQBgNVBAMTCWxvY2FsaG9zdDEdMBsGCSqGSIb3
DQEJARYOYnJhZEBkYW5nYS5jb20wHhcNMTQwNzE1MjA0NjA1WhcNMTcwNTA0MjA0
NjA1WjB7MQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xFDASBgNVBAoTC0JyYWRmaXR6aW5jMRIwEAYDVQQDEwlsb2NhbGhv
c3QxHTAbBgkqhkiG9w0BCQEWDmJyYWRAZGFuZ2EuY29tMIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEAt5fAjp4fTcekWUTfzsp0kyih1OYbsGL0KX1eRbSS
R8Od0+9Q62Hyny+GFwMTb4A/KU8mssoHvcceSAAbwfbxFK/+s51TobqUnORZrOoT
ZjkUygbyXDSK99YBbcR1Pip8vwMTm4XKuLtCigeBBdjjAQdgUO28LENGlsMnmeYk
JfODVGnVmr5Ltb9ANA8IKyTfsnHJ4iOCS/PlPbUj2q7YnoVLposUBMlgUb/CykX3
mOoLb4yJJQyA/iST6ZxiIEj36D4yWZ5lg7YJl+UiiBQHGCnPdGyipqV06ex0heYW
caiW8LWZSUQ93jQ+WVCH8hT7DQO1dmsvUmXlq/JeAlwQ/QIDAQABo4HgMIHdMB0G
A1UdDgQWBBRcAROthS4P4U7vTfjByC569R7E6DCBrQYDVR0jBIGlMIGigBRcAROt
hS4P4U7vTfjByC569R7E6KF/pH0wezELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNB
MRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtCcmFkZml0emluYzES
MBAGA1UEAxMJbG9jYWxob3N0MR0wGwYJKoZIhvcNAQkBFg5icmFkQGRhbmdhLmNv
bYIJALfRlWsI8YQHMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBAG6h
U9f9sNH0/6oBbGGy2EVU0UgITUQIrFWo9rFkrW5k/XkDjQm+3lzjT0iGR4IxE/Ao
eU6sQhua7wrWeFEn47GL98lnCsJdD7oZNhFmQ95Tb/LnDUjs5Yj9brP0NWzXfYU4
UK2ZnINJRcJpB8iRCaCxE8DdcUF0XqIEq6pA272snoLmiXLMvNl3kYEdm+je6voD
58SNVEUsztzQyXmJEhCpwVI0A6QCjzXj+qvpmw3ZZHi8JwXei8ZZBLTSFBki8Z7n
sH9BBH38/SzUmAN4QHSPy1gjqm00OAE8NaYDkh/bzE4d7mLGGMWp/WE3KPSu82HF
kPe6XoSbiLm/kxk32T0=
-----END CERTIFICATE-----`

const (
	defaultOSPPathPrefix  = "../../../../deploy/osps/default/"
	defaultKubeletVersion = "1.31.0"
	ospUbuntu             = "osp-ubuntu"
)

const (
	clusterInfoKubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHRENDQWdDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREE5TVRzd09RWURWUVFERXpKeWIyOTAKTFdOaExuUTNjV3R4ZURWeGRDNWxkWEp2Y0dVdGQyVnpkRE10WXk1a1pYWXVhM1ZpWlhKdFlYUnBZeTVwYnpBZQpGdzB4T0RBeU1ERXhNelUyTURoYUZ3MHlPREF4TXpBeE16VTJNRGhhTUQweE96QTVCZ05WQkFNVE1uSnZiM1F0ClkyRXVkRGR4YTNGNE5YRjBMbVYxY205d1pTMTNaWE4wTXkxakxtUmxkaTVyZFdKbGNtMWhkR2xqTG1sdk1JSUIKSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXA2SDZWNTZiWUh2Q2V6TGtyZkl6TTgxYgppbzcvWmF3L0xLRXcwZUYrTE12NEUrL1EvZkZoc0hDK21oZUxnMUhXVVBGUFJrNFBRODVtQS80dGppbWpTUEZECms2U0ltektGTFlRZ3dDZ2dpVzhOMmhPKzl6ckJVQUxKRkdCNjRvT2NiQmo2RXIvK05sUEdJM1JSV1dkaUVUV0YKV1lDNGpmSmpiRjVQYnl5WEhuc0dmdFNOWVpCTDcxVzdoOWpMV3B5VVdLTDZaWUFOd0RPTjJSYnA3dHB1dzBYNgprayswQVZ3VnprMzArTU56bWY1MHF3K284MThiZkxVRGthTk1mTFM2STB3UW03UkdnK01nVlJEeTNDdVlxZklXClkyeng2YzdQcXpGc1ZWZklyYTBiMHFhdE5sMVhIajh0K0dOcWRiaTIvRlFqQ3hpbFROdW50VDN2eTJlT0hRSUQKQVFBQm95TXdJVEFPQmdOVkhROEJBZjhFQkFNQ0FxUXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QU5CZ2txaGtpRwo5dzBCQVFzRkFBT0NBUUVBSW1FbklYVjNEeW1DcTlxUDdwK3VKNTV1Zlhka1IyZ2hEVVlyVFRjUHdqUjJqVEhhCmlaQStnOG42UXJVb0NENnN6RytsaGFsN2hQNnhkV3VSalhGSE83Yk52NjNJcUVHelJEZ3c1Z3djcVVUWkV2d2cKZ216NzU5dy9hRmYxVjEyaDFhZlBtQTlFRzVOZEh4c3g5QWxIK0Y2dHlzcHBXaFU4WEVRVUFLQ1BqbndVbUs0cAo3Z3ZUWnIyeno0bndoWm8zTDg5MDNxcHRjcTFsWjRPWXNEb1hvbDF1emFRSDgyeHl3ZVNLQ0tYcE9iaXplNVowCndwbmpkRHVIODd4NHI0TGpNWnB1M3ZYNkxqQkRNUFdrSEhQTjVBaW0xSkx0Ny9STFBnVHRqc0pNclRBUzdoZ1oKZktMTDlRTVFsNnMxckhKNEtrL2U3S0c4SEE0aEVORWhrOVlEZlE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://foo.bar:6443
  name: ""
contexts: null
current-context: ""
kind: Config
preferences: {}
users: null
`
)

var (
	update = flag.Bool("update", false, "update testdata files")
)

func init() {
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register osmv1alpha1 with scheme: %v", err))
	}
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register v1alpha1 with scheme: %v", err))
	}
}

type testConfig struct {
	namespace             string
	containerRuntime      string
	clusterDNSIPs         []net.IP
	featureGates          map[string]bool
	externalCloudProvider bool
}

func TestReconciler_Reconcile(t *testing.T) {
	var testCases = []struct {
		name                   string
		kubeletVersion         string
		ospFile                string
		ospName                string
		oscFile                string
		operatingSystem        providerconfigtypes.OperatingSystem
		mdName                 string
		ipFamily               cloudproviderutil.IPFamily
		bootstrapSecretFile    string
		provisioningSecretFile string
		config                 testConfig
		cloudProvider          string
		cloudProviderSpec      runtime.RawExtension
		additionalAnnotations  map[string]string
	}{
		{
			name:                   "Ubuntu OS in AWS with Containerd",
			ospFile:                defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:                ospUbuntu,
			operatingSystem:        providerconfigtypes.OperatingSystemUbuntu,
			oscFile:                "osc-ubuntu-aws-containerd.yaml",
			mdName:                 "ubuntu-aws",
			kubeletVersion:         "1.29.0",
			provisioningSecretFile: "secret-ubuntu-aws-containerd-provisioning.yaml",
			bootstrapSecretFile:    "secret-ubuntu-aws-containerd-bootstrap.yaml",
			config: testConfig{
				namespace:             "kube-system",
				containerRuntime:      "containerd",
				externalCloudProvider: true,
				clusterDNSIPs:         []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:                   "Ubuntu OS in AWS with Dualstack Networking",
			ospFile:                defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:                ospUbuntu,
			operatingSystem:        providerconfigtypes.OperatingSystemUbuntu,
			oscFile:                "osc-ubuntu-aws-dualstack.yaml",
			mdName:                 "ubuntu-aws",
			ipFamily:               cloudproviderutil.IPFamilyIPv4IPv6,
			kubeletVersion:         "1.29.0",
			provisioningSecretFile: "secret-ubuntu-aws-dualstack-provisioning.yaml",
			bootstrapSecretFile:    "secret-ubuntu-aws-dualstack-bootstrap.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:                   "Ubuntu OS in AWS with Dualstack IPv6+IPv4 Networking",
			ospFile:                defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:                ospUbuntu,
			operatingSystem:        providerconfigtypes.OperatingSystemUbuntu,
			oscFile:                "osc-ubuntu-aws-dualstack-IPv6+IPv4.yaml",
			mdName:                 "ubuntu-aws",
			ipFamily:               cloudproviderutil.IPFamilyIPv6IPv4,
			kubeletVersion:         "1.29.0",
			provisioningSecretFile: "secret-ubuntu-aws-dualstack-IPv6+IPv4-provisioning.yaml",
			bootstrapSecretFile:    "secret-ubuntu-aws-dualstack-IPv6+IPv4-bootstrap.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:                   "Flatcar OS in AWS with Containerd",
			ospFile:                defaultOSPPathPrefix + "osp-flatcar.yaml",
			ospName:                "osp-flatcar",
			operatingSystem:        providerconfigtypes.OperatingSystemFlatcar,
			oscFile:                "osc-flatcar-aws-containerd.yaml",
			mdName:                 "flatcar-aws-containerd",
			kubeletVersion:         defaultKubeletVersion,
			provisioningSecretFile: "secret-flatcar-aws-containerd-provisioning.yaml",
			bootstrapSecretFile:    "secret-flatcar-aws-containerd-bootstrap.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:                   "RHEL OS in AWS with Containerd",
			ospFile:                "osp-rhel-aws-cloud-init-modules.yaml",
			ospName:                "osp-rhel-cloud-init-modules",
			operatingSystem:        providerconfigtypes.OperatingSystemRHEL,
			oscFile:                "osc-rhel-8.x-cloud-init-modules.yaml",
			provisioningSecretFile: "secret-osc-rhel-8.x-cloud-init-modules-provisioning.yaml",
			bootstrapSecretFile:    "secret-osc-rhel-8.x-cloud-init-modules-bootstrap.yaml",
			mdName:                 "osp-rhel-aws",
			kubeletVersion:         defaultKubeletVersion,
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:                   "RHEL OS on Azure with Containerd",
			ospFile:                defaultOSPPathPrefix + "osp-rhel.yaml",
			ospName:                "osp-rhel",
			operatingSystem:        providerconfigtypes.OperatingSystemRHEL,
			oscFile:                "osc-rhel-8.x-azure-containerd.yaml",
			mdName:                 "osp-rhel-azure",
			kubeletVersion:         defaultKubeletVersion,
			provisioningSecretFile: "secret-rhel-8.x-azure-containerd-provisioning.yaml",
			bootstrapSecretFile:    "secret-rhel-8.x-azure-containerd-bootstrap.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "azure",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"securityGroupName": "fake-sg"}`)},
		},
		{
			name:                   "Kubelet configuration with containerd",
			ospFile:                defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:                ospUbuntu,
			operatingSystem:        providerconfigtypes.OperatingSystemUbuntu,
			oscFile:                "osc-kubelet-configuration-containerd.yaml",
			mdName:                 "kubelet-configuration",
			kubeletVersion:         defaultKubeletVersion,
			provisioningSecretFile: "secret-kubelet-configuration-containerd-provisioning.yaml",
			bootstrapSecretFile:    "secret-kubelet-configuration-containerd-bootstrap.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
			additionalAnnotations: map[string]string{
				"v1.kubelet-config.machine-controller.kubermatic.io/ContainerLogMaxSize":  "300Mi",
				"v1.kubelet-config.machine-controller.kubermatic.io/ContainerLogMaxFiles": "30",
				"v1.kubelet-config.machine-controller.kubermatic.io/MaxPods":              "110",
				"v1.kubelet-config.machine-controller.kubermatic.io/SystemReserved":       "ephemeral-storage=30Gi,cpu=30m",
				"v1.kubelet-config.machine-controller.kubermatic.io/KubeReserved":         "ephemeral-storage=30Gi,cpu=30m",
				"v1.kubelet-config.machine-controller.kubermatic.io/EvictionHard":         "memory.available<30Mi",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		osp := &osmv1alpha1.OperatingSystemProfile{}
		if err := loadFile(osp, testCase.ospFile); err != nil {
			t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
		}
		objects := []controllerruntimeclient.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-info",
					Namespace: "kube-public",
				},
				Data: map[string]string{"kubeconfig": clusterInfoKubeconfig},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-init-getter-token",
					Namespace: "cloud-init-settings",
				},
				Data: map[string][]byte{
					"token": []byte("top-secret"),
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bootstrap-token",
					Namespace: "kube-system",
					Labels:    map[string]string{"machinedeployment.k8s.io/name": fmt.Sprintf("%s-%s", testCase.config.namespace, testCase.mdName)},
				},
				Data: map[string][]byte{
					"token-id":     []byte("test"),
					"token-secret": []byte("test"),
					"expiration":   []byte(metav1.Now().Add(10 * time.Hour).Format(time.RFC3339)),
				},
			},
		}

		objects = append(objects, osp)

		fakeClient := fakectrlruntimeclient.
			NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(objects...).
			Build()

		reconciler := buildReconciler(fakeClient, testCase.config)

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			md := generateMachineDeployment(t, testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.kubeletVersion, testCase.operatingSystem, testCase.cloudProvider, testCase.cloudProviderSpec, testCase.additionalAnnotations, testCase.ipFamily)

			// Configure containerRuntimeConfig
			containerRuntimeOpts := containerruntime.Opts{
				ContainerRuntime:   testCase.config.containerRuntime,
				InsecureRegistries: "192.168.100.100:5000, 10.0.0.1:5000",
				PauseImage:         "192.168.100.100:5000/kubernetes/pause:v3.1",
				RegistryMirrors:    "https://registry.docker-cn.com",
			}
			containerRuntimeConfig, err := containerruntime.BuildConfig(containerRuntimeOpts)
			if err != nil {
				t.Fatalf("failed to generate container runtime config: %v", err)
			}

			reconciler.containerRuntimeConfig = containerRuntimeConfig

			if err := reconciler.reconcile(ctx, md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			testOSC := &osmv1alpha1.OperatingSystemConfig{}
			if err := loadFile(testOSC, testCase.oscFile); err != nil {
				t.Fatalf("failed loading osc %s from testdata: %v", testCase.name, err)
			}

			oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			osc.TypeMeta = metav1.TypeMeta{
				Kind:       "OperatingSystemConfig",
				APIVersion: osmv1alpha1.SchemeGroupVersion.String(),
			}

			buff, err := yaml.Marshal(osc)
			if err != nil {
				t.Fatal(err)
			}
			testUtil.CompareOutput(t, testCase.oscFile, string(buff), *update)

			bootstrapSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, mcbootstrap.BootstrapCloudConfig)
			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      bootstrapSecretName},
				secret); err != nil {
				t.Fatalf("failed to get bootstrap secret: %v", err)
			}

			testSecret := &corev1.Secret{}
			if err := loadFile(testSecret, testCase.bootstrapSecretFile); err != nil {
				t.Fatalf("failed loading bootstrap secret %s from testdata: %v", testCase.bootstrapSecretFile, err)
			}

			secret.TypeMeta = metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			}

			buff, err = yaml.Marshal(secret)
			if err != nil {
				t.Fatal(err)
			}
			testUtil.CompareOutput(t, testCase.bootstrapSecretFile, string(buff), *update)

			provisioningSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
			secret = &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      provisioningSecretName},
				secret); err != nil {
				t.Fatalf("failed to get provisioning secret: %v", err)
			}

			secret.TypeMeta = metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			}

			if err := loadFile(testSecret, testCase.provisioningSecretFile); err != nil {
				t.Fatalf("failed loading secret %s from testdata: %v", testCase.provisioningSecretFile, err)
			}

			buff, err = yaml.Marshal(secret)
			if err != nil {
				t.Fatal(err)
			}
			testUtil.CompareOutput(t, testCase.provisioningSecretFile, string(buff), *update)
		})
	}
}

func TestOSCAndSecretRotation(t *testing.T) {
	var testCases = []struct {
		name              string
		kubeletVersion    string
		ospFile           string
		ospName           string
		operatingSystem   providerconfigtypes.OperatingSystem
		oscFile           string
		mdName            string
		secretFile        string
		config            testConfig
		cloudProvider     string
		cloudProviderSpec runtime.RawExtension
	}{
		{
			name:            "test updates of machineDeployment",
			ospFile:         defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:         ospUbuntu,
			operatingSystem: providerconfigtypes.OperatingSystemUbuntu,
			oscFile:         "osc-ubuntu-aws-containerd.yaml",
			mdName:          "ubuntu-aws",
			kubeletVersion:  defaultKubeletVersion,
			secretFile:      "secret-ubuntu-aws-containerd.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		osp := &osmv1alpha1.OperatingSystemProfile{}
		if err := loadFile(osp, testCase.ospFile); err != nil {
			t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
		}

		objects := []controllerruntimeclient.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-info",
					Namespace: "kube-public",
				},
				Data: map[string]string{"kubeconfig": clusterInfoKubeconfig},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-init-getter-token",
					Namespace: "cloud-init-settings",
				},
				Data: map[string][]byte{
					"token": []byte("top-secret"),
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bootstrap-token",
					Namespace: "kube-system",
					Labels:    map[string]string{"machinedeployment.k8s.io/name": fmt.Sprintf("%s-%s", testCase.config.namespace, testCase.mdName)},
				},
				Data: map[string][]byte{
					"token-id":     []byte("test"),
					"token-secret": []byte("test"),
					"expiration":   []byte(metav1.Now().Add(10 * time.Hour).Format(time.RFC3339)),
				},
			},
		}

		md := generateMachineDeployment(t, testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.kubeletVersion, testCase.operatingSystem, testCase.cloudProvider, testCase.cloudProviderSpec, nil, cloudproviderutil.IPFamilyIPv4)

		objects = append(objects, osp)
		objects = append(objects, md)

		fakeClient := fakectrlruntimeclient.
			NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(objects...).
			Build()

		reconciler := buildReconciler(fakeClient, testCase.config)

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			if err := reconciler.reconcile(ctx, md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
			// Ensure that OperatingSystemConfig was created
			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			// Ensure that corresponding secrets were created
			bootstrapSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, mcbootstrap.BootstrapCloudConfig)
			bootstrapSecret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      bootstrapSecretName},
				bootstrapSecret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			provisioningSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
			provisioningSecret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      provisioningSecretName},
				provisioningSecret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			oscRevision := osc.Annotations[mcbootstrap.MachineDeploymentRevision]
			bootstrapSecretRevision := bootstrapSecret.Annotations[mcbootstrap.MachineDeploymentRevision]
			provisioningSecretRevision := provisioningSecret.Annotations[mcbootstrap.MachineDeploymentRevision]
			revision := md.Annotations[machinecontrollerutil.RevisionAnnotation]

			if revision != oscRevision {
				t.Fatal("revision for machine deployment and OSC didn't match")
			}

			if revision != bootstrapSecretRevision {
				t.Fatal("revision for machine deployment and bootstrap secret didn't match")
			}

			if revision != provisioningSecretRevision {
				t.Fatal("revision for machine deployment and provisioning secret didn't match")
			}

			// Change the revision manually to trigger OSC and secret rotation
			md.Annotations[machinecontrollerutil.RevisionAnnotation] = "2"

			// Reconcile to trigger delete workflow
			if err := reconciler.reconcile(ctx, md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig exists
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			// Ensure that corresponding secret exists
			bootstrapSecretName = fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, mcbootstrap.BootstrapCloudConfig)
			bootstrapSecret = &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      bootstrapSecretName},
				bootstrapSecret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			provisioningSecretName = fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
			provisioningSecret = &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      provisioningSecretName},
				provisioningSecret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			oscRevision = osc.Annotations[mcbootstrap.MachineDeploymentRevision]
			bootstrapSecretRevision = bootstrapSecret.Annotations[mcbootstrap.MachineDeploymentRevision]
			provisioningSecretRevision = provisioningSecret.Annotations[mcbootstrap.MachineDeploymentRevision]
			updatedRevision := md.Annotations[machinecontrollerutil.RevisionAnnotation]

			if updatedRevision == revision {
				t.Fatal("revision for machine deployment was not updated")
			}

			if updatedRevision != oscRevision {
				t.Fatal("revision for machine deployment and OSC didn't match")
			}

			if updatedRevision != bootstrapSecretRevision {
				t.Fatal("revision for machine deployment and bootstrap secret didn't match")
			}

			if updatedRevision != provisioningSecretRevision {
				t.Fatal("revision for machine deployment and provisioning secret didn't match")
			}
		})
	}
}

func TestMachineDeploymentDeletion(t *testing.T) {
	var testCases = []struct {
		name              string
		kubeletVersion    string
		ospFile           string
		ospName           string
		operatingSystem   providerconfigtypes.OperatingSystem
		oscFile           string
		mdName            string
		secretFile        string
		config            testConfig
		cloudProvider     string
		cloudProviderSpec runtime.RawExtension
	}{
		{
			name:            "test the deletion of machineDeployment",
			ospFile:         defaultOSPPathPrefix + fmt.Sprintf("%s.yaml", ospUbuntu),
			ospName:         ospUbuntu,
			operatingSystem: providerconfigtypes.OperatingSystemUbuntu,
			oscFile:         "osc-ubuntu-aws-containerd.yaml",
			mdName:          "ubuntu-aws",
			kubeletVersion:  defaultKubeletVersion,
			secretFile:      "secret-ubuntu-aws-containerd.yaml",
			config: testConfig{
				namespace:        "kube-system",
				containerRuntime: "containerd",
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"availabilityZone": "eu-central-1b", "vpcId": "e-123f", "subnetID": "test-subnet"}`)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		osp := &osmv1alpha1.OperatingSystemProfile{}
		if err := loadFile(osp, testCase.ospFile); err != nil {
			t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
		}

		objects := []controllerruntimeclient.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-info",
					Namespace: "kube-public",
				},
				Data: map[string]string{"kubeconfig": clusterInfoKubeconfig},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-init-getter-token",
					Namespace: "cloud-init-settings",
				},
				Data: map[string][]byte{
					"token": []byte("top-secret"),
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bootstrap-token",
					Namespace: "kube-system",
					Labels:    map[string]string{"machinedeployment.k8s.io/name": fmt.Sprintf("%s-%s", testCase.config.namespace, testCase.mdName)},
				},
				Data: map[string][]byte{
					"token-id":     []byte("test"),
					"token-secret": []byte("test"),
					"expiration":   []byte(metav1.Now().Add(10 * time.Hour).Format(time.RFC3339)),
				},
			},
		}

		md := generateMachineDeployment(t, testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.kubeletVersion, testCase.operatingSystem, testCase.cloudProvider, testCase.cloudProviderSpec, nil, cloudproviderutil.IPFamilyIPv4)

		objects = append(objects, osp)
		objects = append(objects, md)

		fakeClient := fakectrlruntimeclient.
			NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(objects...).
			Build()

		reconciler := buildReconciler(fakeClient, testCase.config)

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			if err := reconciler.reconcile(ctx, md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was created
			oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			// Ensure that corresponding secrets were created
			secret := &corev1.Secret{}
			bootstrapSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, mcbootstrap.BootstrapCloudConfig)
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      bootstrapSecretName},
				secret); err != nil {
				t.Fatalf("failed to get bootstrap secret: %v", err)
			}

			provisioningSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      provisioningSecretName},
				secret); err != nil {
				t.Fatalf("failed to get provisioning secret: %v", err)
			}

			// Reconcile to trigger delete workflow
			_, err := reconciler.handleMachineDeploymentCleanup(ctx, md)
			if err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      oscName},
				osc); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to ensure that osc is deleted: %v", err)
			}

			// Ensure that corresponding secret was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      bootstrapSecretName},
				secret); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to ensure that secret is deleted: %s", err)
			}

			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: mcbootstrap.CloudInitSettingsNamespace,
				Name:      provisioningSecretName},
				secret); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to ensure that secret is deleted: %s", err)
			}
		})
	}
}

func generateMachineDeployment(t *testing.T, name, namespace, osp, kubeletVersion string, os providerconfigtypes.OperatingSystem, cloudprovider string, cloudProviderSpec runtime.RawExtension, additionalAnnotations map[string]string, ipFamily cloudproviderutil.IPFamily) *v1alpha1.MachineDeployment {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem:   os,
		CloudProviderSpec: cloudProviderSpec,
		CloudProvider:     providerconfigtypes.CloudProvider(cloudprovider),
		Network: &providerconfigtypes.NetworkConfig{
			IPFamily: ipFamily,
		},
	}

	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		t.Fatalf("failed to generate machine deployment: %v", err)
	}

	annotations := make(map[string]string)
	annotations[machinecontrollerutil.RevisionAnnotation] = "1"
	annotations[resources.MachineDeploymentOSPAnnotation] = osp
	for k, v := range additionalAnnotations {
		annotations[k] = v
	}

	md := &v1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: v1alpha1.MachineDeploymentSpec{
			Template: v1alpha1.MachineTemplateSpec{
				Spec: v1alpha1.MachineSpec{
					Versions: v1alpha1.MachineVersionInfo{
						Kubelet: kubeletVersion,
					},
					ProviderSpec: v1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: mdConfig,
						},
					},
				},
			},
		},
	}
	return md
}

func loadFile(obj runtime.Object, name string) error {
	path, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		return fmt.Errorf("failed to get absolute path to testdata %s: %w", name, err)
	}
	objBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read testdata file: %w", err)
	}

	err = yaml.Unmarshal(objBytes, obj)
	if err != nil {
		return err
	}
	return nil
}

func buildReconciler(fakeClient controllerruntimeclient.Client, config testConfig) Reconciler {
	kubeconfigProvider := clusterinfo.New(fakeClient, "foobar")
	bootstrappingManager := bootstrap.New(fakeClient, kubeconfigProvider, nil, "")
	featureGates := map[string]bool{"GracefulNodeShutdown": true, "IdentifyPodOS": false}
	if config.featureGates != nil {
		featureGates = config.featureGates
	}

	return Reconciler{
		Client:       fakeClient,
		workerClient: fakeClient,

		log:                   testUtil.DefaultLogger,
		generator:             generator.NewDefaultCloudConfigGenerator(""),
		namespace:             config.namespace,
		caCert:                dummyCACert,
		containerRuntime:      config.containerRuntime,
		clusterDNSIPs:         config.clusterDNSIPs,
		kubeletFeatureGates:   featureGates,
		bootstrappingManager:  bootstrappingManager,
		externalCloudProvider: config.externalCloudProvider,
		nodeHTTPProxy:         "http://test-http-proxy.com",
		nodeNoProxy:           "http://test-no-proxy.com",
	}
}
