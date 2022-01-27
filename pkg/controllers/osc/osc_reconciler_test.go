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
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	namespace        string
	containerRuntime string
	kubeVersion      string
	clusterDNSIPs    []net.IP
}

func TestReconciler_Reconcile(t *testing.T) {
	var testCases = []struct {
		name              string
		ospFile           string
		ospName           string
		oscFile           string
		oscName           string
		operatingSystem   providerconfigtypes.OperatingSystem
		mdName            string
		secretFile        string
		config            testConfig
		cloudProvider     string
		cloudProviderSpec runtime.RawExtension
	}{
		{
			name:            "Ubuntu OS in AWS with Containerd",
			ospFile:         "osp-ubuntu-20.04.yaml",
			ospName:         "osp-ubuntu",
			operatingSystem: providerconfigtypes.OperatingSystemUbuntu,
			oscFile:         "osc-ubuntu-20.04-aws-containerd.yaml",
			oscName:         "ubuntu-20.04-aws-osc-provisioning",
			mdName:          "ubuntu-20.04-aws",
			secretFile:      "secret-ubuntu-20.04-aws-containerd.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "containerd",
				kubeVersion:      "1.22.1",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"zone": "eu-central-1b", "vpc": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:            "Ubuntu OS in AWS with Docker",
			ospFile:         "osp-ubuntu-20.04.yaml",
			ospName:         "osp-ubuntu",
			operatingSystem: providerconfigtypes.OperatingSystemUbuntu,
			oscFile:         "osc-ubuntu-20.04-aws-docker.yaml",
			oscName:         "ubuntu-20.04-aws-osc-provisioning",
			mdName:          "ubuntu-20.04-aws",
			secretFile:      "secret-ubuntu-20.04-aws-docker.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "docker",
				kubeVersion:      "1.22.1",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"zone": "eu-central-1b", "vpc": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:            "Flatcar OS in AWS with Containerd",
			ospFile:         "osp-flatcar.yaml",
			ospName:         "osp-flatcar",
			operatingSystem: providerconfigtypes.OperatingSystemFlatcar,
			oscFile:         "osc-flatcar-aws-containerd.yaml",
			oscName:         "flatcar-aws-containerd-osc-provisioning",
			mdName:          "flatcar-aws-containerd",
			secretFile:      "secret-flatcar-aws-containerd.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "containerd",
				kubeVersion:      "1.22.1",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"zone": "eu-central-1b", "vpc": "e-123f", "subnetID": "test-subnet"}`)},
		},
		{
			name:            "Flatcar OS in AWS with docker",
			ospFile:         "osp-flatcar.yaml",
			ospName:         "osp-flatcar",
			operatingSystem: providerconfigtypes.OperatingSystemFlatcar,
			oscFile:         "osc-flatcar-aws-docker.yaml",
			oscName:         "flatcar-aws-docker-osc-provisioning",
			mdName:          "flatcar-aws-docker",
			secretFile:      "secret-flatcar-aws-docker.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "docker",
				kubeVersion:      "1.22.1",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"cloud-config-key": "cloud-config-value"}`)},
		},
		{
			name:            "RHEL OS in AWS with Containerd",
			ospFile:         "osp-rhel.yaml",
			ospName:         "osp-rhel",
			operatingSystem: providerconfigtypes.OperatingSystemRHEL,
			oscFile:         "osc-rhel-8.x-containerd.yaml",
			oscName:         "osp-rhel-aws-osc-provisioning",
			mdName:          "osp-rhel-aws",
			secretFile:      "secret-rhel-8.x-containerd.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "containerd",
				kubeVersion:      "1.22.1",
				clusterDNSIPs:    []net.IP{net.IPv4(10, 0, 0, 0)},
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"zone": "eu-central-1b", "vpc": "e-123f", "subnetID": "test-subnet"}`)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		osp := &osmv1alpha1.OperatingSystemProfile{}
		if err := loadFile(osp, testCase.ospFile); err != nil {
			t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
		}

		fakeClient := fakectrlruntimeclient.
			NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(osp).
			Build()

		reconciler := buildReconciler(fakeClient, testCase.config)

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			md := generateMachineDeployment(t, testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.operatingSystem, testCase.cloudProvider, testCase.cloudProviderSpec)

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
				t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
			}

			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      testCase.oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			buff, err := yaml.Marshal(osc)
			if err != nil {
				t.Fatalf(err.Error())
			}
			testUtil.CompareOutput(t, testCase.oscFile, string(buff), *update)

			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "cloud-init-settings",
				Name:      testCase.oscName},
				secret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			testSecret := &corev1.Secret{}
			if err := loadFile(testSecret, testCase.secretFile); err != nil {
				t.Fatalf("failed loading secret %s from testdata: %v", testCase.secretFile, err)
			}

			buff, err = yaml.Marshal(secret)
			if err != nil {
				t.Fatalf(err.Error())
			}
			testUtil.CompareOutput(t, testCase.secretFile, string(buff), *update)
		})
	}
}

func TestMachineDeploymentDeletion(t *testing.T) {
	var testCases = []struct {
		name              string
		ospFile           string
		ospName           string
		operatingSystem   providerconfigtypes.OperatingSystem
		oscFile           string
		oscName           string
		mdName            string
		secretFile        string
		config            testConfig
		cloudProvider     string
		cloudProviderSpec runtime.RawExtension
	}{
		{

			name:            "test the deletion of machineDeployment",
			ospFile:         "osp-ubuntu-20.04.yaml",
			ospName:         "osp-ubuntu",
			operatingSystem: providerconfigtypes.OperatingSystemUbuntu,
			oscFile:         "osc-ubuntu-20.04-aws-containerd.yaml",
			oscName:         "ubuntu-20.04-aws-osc-provisioning",
			mdName:          "ubuntu-20.04-aws",
			secretFile:      "secret-ubuntu-20.04-aws-containerd.yaml",
			config: testConfig{
				namespace:        "cloud-init-settings",
				containerRuntime: "containerd",
			},
			cloudProvider:     "aws",
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"cloudProvider":"aws", "cloudProviderSpec":"test-provider-spec"}`)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		osp := &osmv1alpha1.OperatingSystemProfile{}
		if err := loadFile(osp, testCase.ospFile); err != nil {
			t.Fatalf("failed loading osp %s from testdata: %v", testCase.name, err)
		}

		md := generateMachineDeployment(t, testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.operatingSystem, testCase.cloudProvider, testCase.cloudProviderSpec)
		fakeClient := fakectrlruntimeclient.
			NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(osp, md).
			Build()

		reconciler := buildReconciler(fakeClient, testCase.config)

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			if err := reconciler.reconcile(ctx, md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was created
			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      testCase.oscName},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			// Ensure that corresponding secret was created
			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      testCase.oscName},
				secret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			// Retrieve MachineDeployment
			machineDeployment := &v1alpha1.MachineDeployment{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: testCase.config.namespace,
				Name:      testCase.mdName},
				machineDeployment); err != nil {
				t.Fatalf("failed to get machine deployment: %v", err)
			}

			// Add deletionTimestamp to Machinedeployment to queue it up for deletion
			machineDeployment.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			err := fakeClient.Update(ctx, machineDeployment)
			if err != nil {
				t.Fatalf("failed to update machine deployment with deletionTimestamp: %v", err)
			}

			// Reconcile to trigger delete workflow
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: md.Name, Namespace: md.Namespace}})
			if err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "cloud-init-settings",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudConfig)},
				osc); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to delete osc")
			}

			// Ensure that corresponding secret was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "cloud-init-settings",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudConfig)},
				secret); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to delete secret")
			}
		})
	}
}

func generateMachineDeployment(t *testing.T, name, namespace, osp string, os providerconfigtypes.OperatingSystem, cloudprovider string, cloudProviderSpec runtime.RawExtension) *v1alpha1.MachineDeployment {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem:   os,
		CloudProviderSpec: cloudProviderSpec,
		CloudProvider:     providerconfigtypes.CloudProvider(cloudprovider),
	}
	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		t.Fatalf("failed to generate machine deployment: %v", err)
	}

	md := &v1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				resources.MachineDeploymentOSPAnnotation: osp,
			},
		},
		Spec: v1alpha1.MachineDeploymentSpec{
			Template: v1alpha1.MachineTemplateSpec{
				Spec: v1alpha1.MachineSpec{
					Versions: v1alpha1.MachineVersionInfo{
						Kubelet: "1.22.1",
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
		return fmt.Errorf("failed to get absolute path to testdata %s: %v", name, err)
	}
	objBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read testdata file: %v", err)
	}

	err = yaml.Unmarshal(objBytes, obj)
	if err != nil {
		return err
	}
	return nil
}

func buildReconciler(fakeClient client.Client, config testConfig) Reconciler {
	return Reconciler{
		Client:       fakeClient,
		workerClient: fakeClient,

		log:                 testUtil.DefaultLogger,
		generator:           generator.NewDefaultCloudConfigGenerator(""),
		namespace:           config.namespace,
		caCert:              dummyCACert,
		containerRuntime:    config.containerRuntime,
		clusterDNSIPs:       config.clusterDNSIPs,
		kubeletFeatureGates: map[string]bool{"GracefulNodeShutdown": true, "IdentifyPodOS": false},
	}
}
