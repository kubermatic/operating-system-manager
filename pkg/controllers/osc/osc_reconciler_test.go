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

package osc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	"k8s.io/utils/pointer"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	osmProfile = &osmv1alpha1.OperatingSystemProfile{
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
	// Path for a dummy kubeconfig; not using a real kubeconfig for this use case
	kubeconfigPath    = os.Getenv("PWD") + "/testdata/kube-config.yaml"
	cloudProviderSpec = runtime.RawExtension{Raw: []byte(`{"cloudProvider":"test-value", "cloudProviderSpec":"test-value"}`)}
)

func init() {
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register osmv1alpha1 with scheme: %v", err))
	}
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register v1alpha1 with scheme: %v", err))
	}
}

func TestReconciler_Reconcile(t *testing.T) {
	machineDeployment := generateMachineDeployment("ubuntu-20.04", "kube-system")

	// Encode cloud provider spec in JSON
	cloudProviderSpecJSON, err := json.Marshal(cloudProviderSpec)
	if err != nil {
		t.Fatalf("failed to marshal cloud provider spec")
	}
	var (
		fakeClient = fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(osmProfile).
				Build()

		testCases = []struct {
			name            string
			reconciler      Reconciler
			md              *v1alpha1.MachineDeployment
			osp             *osmv1alpha1.OperatingSystemProfile
			expectedOSCs    []*osmv1alpha1.OperatingSystemConfig
			expectedSecrets []*corev1.Secret
		}{
			{
				name: "test the creation of operating system config",
				reconciler: Reconciler{
					Client:         fakeClient,
					namespace:      "kube-system",
					generator:      generator.NewDefaultCloudConfigGenerator(""),
					log:            testUtil.DefaultLogger,
					clusterAddress: "http://127.0.0.1/configs",
					kubeconfig:     kubeconfigPath,
				},
				md: machineDeployment,
				expectedOSCs: []*osmv1alpha1.OperatingSystemConfig{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resources.ProvisioningCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Spec: osmv1alpha1.OperatingSystemConfigSpec{
							OSName:    "Ubuntu",
							OSVersion: "20.04",
							CloudProvider: osmv1alpha1.CloudProviderSpec{
								Spec: runtime.RawExtension{
									Raw: cloudProviderSpecJSON,
								},
							},
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
							UserSSHKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
						},
					},
				},
				expectedSecrets: []*corev1.Secret{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resources.ProvisioningCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Data: map[string][]byte{
							"cloud-init": []byte("#cloud-config\nssh_pwauth: no\nssh_authorized_keys:\n- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c'\nwrite_files:\n- path: '/opt/bin/setup'\n  permissions: '0755'\n  content: |-\n    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    systemctl start provision.service\n\n- path: '/etc/systemd/system/setup.service'\n  permissions: '0644'\n  content: |-\n    [Install]\n    WantedBy=multi-user.target\n    \n    [Unit]\n    Requires=network-online.target\n    After=network-online.target\n    [Service]\n    Type=oneshot\n    RemainAfterExit=true\n    ExecStart=/opt/bin/setup\n\nruncmd:\n- systemctl restart setup.service\n- systemctl daemon-reload\n"),
						},
					},
				},
			},
		}
	)

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			if err := testCase.reconciler.reconcile(ctx, testCase.md); err != nil {
				t.Fatalf("failed to reconcile: %v", err)
			}

			osc := &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resources.ProvisioningCloudInit)},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			if !reflect.DeepEqual(osc.ObjectMeta, testCase.expectedOSCs[0].ObjectMeta) ||
				!reflect.DeepEqual(osc.Spec, testCase.expectedOSCs[0].Spec) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}

			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resources.ProvisioningCloudInit)},
				secret); err != nil {
				t.Fatalf("failed to get secret: %v", err)
			}

			if !reflect.DeepEqual(secret.ObjectMeta, testCase.expectedSecrets[0].ObjectMeta) ||
				!reflect.DeepEqual(secret.Data, testCase.expectedSecrets[0].Data) {
				t.Fatal("secret values are unexpected")
			}
		})
	}
}

func TestMachineDeploymentDeletion(t *testing.T) {
	machineDeploymentName := "ubuntu-20.04-lts"
	machineDeployment := generateMachineDeployment(machineDeploymentName, "kube-system")

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
				Client:         fakeClient,
				namespace:      "kube-system",
				generator:      generator.NewDefaultCloudConfigGenerator(""),
				log:            testUtil.DefaultLogger,
				clusterAddress: "http://127.0.0.1/configs",
				kubeconfig:     kubeconfigPath,
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

func generateMachineDeployment(name, namespace string) *v1alpha1.MachineDeployment {
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
				resources.MachineDeploymentOSPAnnotation: "ubuntu-20.04-profile",
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
