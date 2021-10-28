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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	testUtil "k8c.io/operating-system-manager/pkg/test/util"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var (
	// Path for a dummy kubeconfig; not using a real kubeconfig for this use case
	kubeconfigPath string
)

func init() {
	var err error
	kubeconfigPath, err = filepath.Abs(filepath.Join("testdata", "kube-config.yaml"))
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path to testdata kubeconfig: %v", err))
	}
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register osmv1alpha1 with scheme: %v", err))
	}
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register v1alpha1 with scheme: %v", err))
	}
}

type testConfig struct {
	namespace        string
	clusterAddress   string
	containerRuntime string
}

func TestReconciler_Reconcile(t *testing.T) {
	var testCases = []struct {
		name              string
		ospFile           string
		ospName           string
		oscFile           string
		oscName           string
		mdName            string
		secretFile        string
		config            testConfig
		cloudProviderSpec runtime.RawExtension
	}{
		{
			name:       "test the creation of operating system config",
			ospFile:    "osp-ubuntu-20.04.yaml",
			ospName:    "osp-ubuntu-aws",
			oscFile:    "osc-ubuntu-20.04-aws-containerd.yaml",
			oscName:    "ubuntu-20.04-aws-osc-provisioning",
			mdName:     "ubuntu-20.04-aws",
			secretFile: "secret-ubuntu-20.04-aws-containerd.yaml",
			config: testConfig{
				namespace:        "kube-system",
				clusterAddress:   "http://127.0.0.1/configs",
				containerRuntime: "containerd",
			},
			cloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"cloudProvider":"aws", "cloudProviderSpec":"test-provider-spec"}`)},
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

		reconciler := Reconciler{
			Client:           fakeClient,
			log:              testUtil.DefaultLogger,
			generator:        generator.NewDefaultCloudInitGenerator(""),
			namespace:        testCase.config.namespace,
			clusterAddress:   testCase.config.clusterAddress,
			kubeconfig:       kubeconfigPath,
			containerRuntime: testCase.config.containerRuntime,
		}

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			md := generateMachineDeployment(testCase.mdName, testCase.config.namespace, testCase.ospName, testCase.cloudProviderSpec)
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

			if !reflect.DeepEqual(osc.ObjectMeta, testOSC.ObjectMeta) ||
				!reflect.DeepEqual(osc.Spec, testOSC.Spec) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}

			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      testCase.oscName},
				secret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			aa, _ := yaml.Marshal(secret)
			klog.Info(string(aa))

			testSecret := &corev1.Secret{}
			if err := loadFile(testSecret, testCase.secretFile); err != nil {
				t.Fatalf("failed loading secret %s from testdata: %v", testCase.secretFile, err)
			}

			if !reflect.DeepEqual(secret.ObjectMeta, testSecret.ObjectMeta) ||
				!reflect.DeepEqual(secret.Data, testSecret.Data) {
				t.Fatal("secret values are unexpected")
			}
		})
	}
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

var osmProfile = &osmv1alpha1.OperatingSystemProfile{
	ObjectMeta: v1.ObjectMeta{
		Name:      "ubuntu-20.04-profile",
		Namespace: "kube-system",
	},
	Spec: osmv1alpha1.OperatingSystemProfileSpec{
		OSName:    "Ubuntu",
		OSVersion: "20.04",
		Files: []osmv1alpha1.File{
			{
				Path:        "/opt/bin/setup",
				Permissions: pointer.Int32Ptr(0755),
				Content: osmv1alpha1.FileContent{
					Inline: &osmv1alpha1.FileContentInline{
						Encoding: "b64",
						Data:     "#!/bin/bash\nset -xeuo pipefail\ncloud-init clean\nsystemctl start provision.service",
					},
				},
			},
			{
				Path:        "/etc/systemd/system/setup.service",
				Permissions: pointer.Int32Ptr(0644),
				Content: osmv1alpha1.FileContent{
					Inline: &osmv1alpha1.FileContentInline{
						Encoding: "b64",
						Data:     "[Install]\nWantedBy=multi-user.target\n\n[Unit]\nRequires=network-online.target\nAfter=network-online.target\n[Service]\nType=oneshot\nRemainAfterExit=true\nExecStart=/opt/bin/setup",
					},
				},
			},
		},
	},
}

func TestMachineDeploymentDeletion(t *testing.T) {
	machineDeploymentName := "ubuntu-20.04-lts"
	machineDeployment := generateMachineDeployment(machineDeploymentName, "kube-system", "", runtime.RawExtension{})

	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(osmProfile, machineDeployment).
		Build()

	testCases := []struct {
		name       string
		reconciler Reconciler
		md         *v1alpha1.MachineDeployment
	}{
		{
			name: "test the deletion of machine deployment",
			reconciler: Reconciler{
				Client:           fakeClient,
				namespace:        "kube-system",
				generator:        generator.NewDefaultCloudInitGenerator(""),
				log:              testUtil.DefaultLogger,
				clusterAddress:   "http://127.0.0.1/configs",
				kubeconfig:       kubeconfigPath,
				containerRuntime: "containerd",
			},
			md: machineDeployment,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			// First time reconcile to trigger create workflow
			_, err := testCase.reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: testCase.md.Name, Namespace: testCase.md.Namespace}})
			if err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was created
			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudInit)},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			// Ensure that corresponding secret was created
			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudInit)},
				secret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			// Retrieve MachineDeployment
			machineDeployment := &v1alpha1.MachineDeployment{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      machineDeploymentName},
				machineDeployment); err != nil {
				t.Fatalf("failed to get machine deployment: %v", err)
			}

			// Add deletionTimestamp to Machinedeployment to queue it up for deletion
			machineDeployment.ObjectMeta.DeletionTimestamp = &v1.Time{Time: time.Now()}
			if err := fakeClient.Update(ctx, machineDeployment); err != nil {
				t.Fatalf("failed to update machine deployment with deletionTimestamp: %v", err)
			}

			// Reconcile to trigger delete workflow
			_, err = testCase.reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: testCase.md.Name, Namespace: testCase.md.Namespace}})
			if err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			// Ensure that OperatingSystemConfig was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudInit)},
				osc); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to delete osc")
			}

			// Ensure that corresponding secret was deleted
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-lts-osc-%s", resources.ProvisioningCloudInit)},
				secret); err == nil || !kerrors.IsNotFound(err) {
				t.Fatalf("failed to delete secret")
			}
		})
	}
}

func generateMachineDeployment(name, namespace, osp string, cloudProviderSpec runtime.RawExtension) *v1alpha1.MachineDeployment {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem:   "Ubuntu",
		CloudProviderSpec: cloudProviderSpec,
	}
	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		return &v1alpha1.MachineDeployment{}
	}

	md := &v1alpha1.MachineDeployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				resources.MachineDeploymentOSPAnnotation: osp,
			},
		},
		Spec: v1alpha1.MachineDeploymentSpec{
			Template: v1alpha1.MachineTemplateSpec{
				Spec: v1alpha1.MachineSpec{
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
