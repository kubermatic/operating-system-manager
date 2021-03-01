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
	"reflect"
	"testing"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resrources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register osmv1alpha1 with scheme: %v", err))
	}
}

func TestReconciler_Reconcile(t *testing.T) {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:   []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem: "Ubuntu",
	}
	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		t.Fatalf("failed to marshal machine deployment config")
	}
	var (
		fakeClient = fakectrlruntimeclient.NewFakeClient(
			&osmv1alpha1.OperatingSystemProfile{
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
			},
		)

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
					generator:      generator.NewDefaultCloudInitGenerator(""),
					clusterAddress: "http://127.0.0.1/configs",
				},
				md: &v1alpha1.MachineDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:      "ubuntu-20.04",
						Namespace: "kube-system",
						Annotations: map[string]string{
							resrources.MachineDeploymentOSPAnnotation: "ubuntu-20.04-profile",
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
				},
				expectedOSCs: []*osmv1alpha1.OperatingSystemConfig{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.BootstrapCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Spec: osmv1alpha1.OperatingSystemConfigSpec{
							OSName: "Ubuntu",
							Files: []osmv1alpha1.File{
								{
									Path:        fmt.Sprintf("/opt/bin/%s", resrources.BootstrapCloudInit),
									Permissions: pointer.Int32Ptr(0755),
									Content: osmv1alpha1.FileContent{
										Inline: &osmv1alpha1.FileContentInline{
											Encoding: "b64",
											Data:     "#!/bin/bash\nset -xeuo pipefail\nwget /ubuntu-20.04-osc-bootstrap.cfg --directory-prefix /etc/cloud/cloud.cfg.d/\ncloud-init clean\ncloud-init --file /etc/cloud/cloud.cfg.d/ubuntu-20.04-osc-bootstrap.cfg init\nsystemctl start provision.service",
										},
									},
								},
								{
									Path:        fmt.Sprintf("/etc/systemd/system/%s.service", resrources.BootstrapCloudInit),
									Permissions: pointer.Int32Ptr(0644),
									Content: osmv1alpha1.FileContent{
										Inline: &osmv1alpha1.FileContentInline{
											Encoding: "b64",
											Data:     "[Install]\nWantedBy=multi-user.target\n\n[Unit]\nRequires=network-online.target\nAfter=network-online.target\n[Service]\nType=oneshot\nRemainAfterExit=true\nExecStart=/opt/bin/bootstrap",
										},
									},
								},
							},
							UserSSHKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.ProvisioningCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Spec: osmv1alpha1.OperatingSystemConfigSpec{
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
							UserSSHKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
						},
					},
				},
				expectedSecrets: []*corev1.Secret{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.BootstrapCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Data: map[string][]byte{
							"cloud-init": []byte("#cloud-config\n\nssh_pwauth: no\nssh_authorized_keys:\n- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c'\nwrite_files:\n- path: '/opt/bin/bootstrap'\n  permissions: '0755'\n  encoding: b64\n  content: |\nIyEvYmluL2Jhc2gKc2V0IC14ZXVvIHBpcGVmYWlsCndnZXQgL3VidW50dS0yMC4wNC1vc2MtYm9vdHN0cmFwLmNmZyAtLWRpcmVjdG9yeS1wcmVmaXggL2V0Yy9jbG91ZC9jbG91ZC5jZmcuZC8KY2xvdWQtaW5pdCBjbGVhbgpjbG91ZC1pbml0IC0tZmlsZSAvZXRjL2Nsb3VkL2Nsb3VkLmNmZy5kL3VidW50dS0yMC4wNC1vc2MtYm9vdHN0cmFwLmNmZyBpbml0CnN5c3RlbWN0bCBzdGFydCBwcm92aXNpb24uc2VydmljZQ==\n\n- path: '/etc/systemd/system/bootstrap.service'\n  permissions: '0644'\n  encoding: b64\n  content: |\nW0luc3RhbGxdCldhbnRlZEJ5PW11bHRpLXVzZXIudGFyZ2V0CgpbVW5pdF0KUmVxdWlyZXM9bmV0d29yay1vbmxpbmUudGFyZ2V0CkFmdGVyPW5ldHdvcmstb25saW5lLnRhcmdldApbU2VydmljZV0KVHlwZT1vbmVzaG90ClJlbWFpbkFmdGVyRXhpdD10cnVlCkV4ZWNTdGFydD0vb3B0L2Jpbi9ib290c3RyYXA=\n\nruncmd:\n- systemctl restart bootstrap.service\n- systemctl daemon-reload\n"),
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name:            fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.ProvisioningCloudInit),
							Namespace:       "kube-system",
							ResourceVersion: "1",
						},
						Data: map[string][]byte{
							"cloud-init": []byte("#cloud-config\n\nssh_pwauth: no\nssh_authorized_keys:\n- 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c'\nwrite_files:\n- path: '/opt/bin/setup'\n  permissions: '0755'\n  encoding: b64\n  content: |\nIyEvYmluL2Jhc2gKc2V0IC14ZXVvIHBpcGVmYWlsCmNsb3VkLWluaXQgY2xlYW4Kc3lzdGVtY3RsIHN0YXJ0IHByb3Zpc2lvbi5zZXJ2aWNl\n\n- path: '/etc/systemd/system/setup.service'\n  permissions: '0644'\n  encoding: b64\n  content: |\nW0luc3RhbGxdCldhbnRlZEJ5PW11bHRpLXVzZXIudGFyZ2V0CgpbVW5pdF0KUmVxdWlyZXM9bmV0d29yay1vbmxpbmUudGFyZ2V0CkFmdGVyPW5ldHdvcmstb25saW5lLnRhcmdldApbU2VydmljZV0KVHlwZT1vbmVzaG90ClJlbWFpbkFmdGVyRXhpdD10cnVlCkV4ZWNTdGFydD0vb3B0L2Jpbi9zZXR1cA==\n\nruncmd:\n- systemctl restart setup.service\n- systemctl daemon-reload\n"),
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
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.BootstrapCloudInit)},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			if !reflect.DeepEqual(osc.ObjectMeta, testCase.expectedOSCs[0].ObjectMeta) ||
				!reflect.DeepEqual(osc.Spec, testCase.expectedOSCs[0].Spec) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}

			osc = &osmv1alpha1.OperatingSystemConfig{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.ProvisioningCloudInit)},
				osc); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}
			if !reflect.DeepEqual(osc.ObjectMeta, testCase.expectedOSCs[1].ObjectMeta) ||
				!reflect.DeepEqual(osc.Spec, testCase.expectedOSCs[1].Spec) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}

			secret := &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.BootstrapCloudInit)},
				secret); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			if !reflect.DeepEqual(secret.ObjectMeta, testCase.expectedSecrets[0].ObjectMeta) ||
				!reflect.DeepEqual(secret.Data, testCase.expectedSecrets[0].Data) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}

			secret = &corev1.Secret{}
			if err := fakeClient.Get(ctx, types.NamespacedName{
				Namespace: "kube-system",
				Name:      fmt.Sprintf("ubuntu-20.04-osc-%s", resrources.ProvisioningCloudInit)},
				secret); err != nil {
				t.Fatalf("failed to get osc: %v", err)
			}

			if !reflect.DeepEqual(secret.ObjectMeta, testCase.expectedSecrets[1].ObjectMeta) ||
				!reflect.DeepEqual(secret.Data, testCase.expectedSecrets[1].Data) {
				t.Fatal("operatingSystemConfig values are unexpected")
			}
		})
	}
}
