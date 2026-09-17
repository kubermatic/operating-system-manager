/*
Copyright 2026 The Operating System Manager contributors.

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
	"net"
	"slices"
	"strings"
	"testing"

	"github.com/coreos/go-systemd/unit"

	mcnet "k8c.io/machine-controller/sdk/net"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/containerruntime"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

func TestFlatcarMetadataDependencies(t *testing.T) {
	osp := &osmv1alpha1.OperatingSystemProfile{}
	if err := loadFile(osp, defaultOSPPathPrefix+"osp-flatcar.yaml"); err != nil {
		t.Fatalf("failed to load Flatcar profile: %v", err)
	}
	kubeconfig, err := clientcmd.Load([]byte(clusterInfoKubeconfig))
	if err != nil {
		t.Fatalf("failed to load bootstrap kubeconfig: %v", err)
	}
	containerRuntimeConfig, err := containerruntime.BuildConfig(containerruntime.Opts{
		ContainerRuntime: "containerd",
		PauseImage:       "registry.k8s.io/pause:3.10",
	})
	if err != nil {
		t.Fatalf("failed to build container runtime config: %v", err)
	}

	for _, cloudProvider := range []string{"aws", "openstack"} {
		t.Run(cloudProvider, func(t *testing.T) {
			providerSpec := runtime.RawExtension{Raw: []byte(`{}`)}
			if cloudProvider == "aws" {
				providerSpec.Raw = []byte(`{"availabilityZone":"eu-central-1b","vpcId":"test-vpc","subnetID":"test-subnet"}`)
			}
			md := generateMachineDeployment(t, "flatcar-metadata", "kube-system", osp.Name, defaultKubeletVersion,
				providerconfig.OperatingSystemFlatcar, cloudProvider, providerSpec, nil, mcnet.IPFamilyIPv4IPv6)
			osc, err := resources.GenerateOperatingSystemConfig(md, osp.DeepCopy(), kubeconfig,
				"bootstrap-token", "test-token", "flatcar-metadata-config", "kube-system", dummyCACert, "",
				[]net.IP{net.ParseIP("10.0.0.10")}, "containerd", true, "", "", "", containerRuntimeConfig, nil)
			if err != nil {
				t.Fatalf("failed to render Flatcar configuration: %v", err)
			}

			// Exercise the actual Ignition conversion: OSP unit contents are copied
			// verbatim, whereas file contents (including drop-ins) are templated.
			_, err = generator.NewDefaultCloudConfigGenerator("").Generate(
				&osc.Spec.ProvisioningConfig, osc.Spec.ProvisioningUtility, osc.Spec.OSName,
				osmv1alpha1.CloudProvider(cloudProvider), *md, resources.ProvisioningCloudConfig)
			if err != nil {
				t.Fatalf("failed to generate Flatcar Ignition configuration: %v", err)
			}
			for _, path := range []string{
				"/etc/systemd/system/setup.service.d/10-aws-metadata.conf",
				"/etc/systemd/system/kubelet.service",
			} {
				index := slices.IndexFunc(osc.Spec.ProvisioningConfig.Files, func(file osmv1alpha1.File) bool {
					return file.Path == path
				})
				if index == -1 {
					if cloudProvider != "aws" && strings.HasSuffix(path, ".conf") {
						continue // Non-AWS profiles do not need the metadata drop-in.
					}
					t.Fatalf("rendered configuration is missing %s", path)
				}
				file := osc.Spec.ProvisioningConfig.Files[index]
				if file.Content.Inline == nil {
					t.Fatalf("rendered file %s has no inline content", path)
				}
				assertFlatcarMetadataDependency(t, path, file.Content.Inline.Data, cloudProvider == "aws")
			}
		})
	}
}

func assertFlatcarMetadataDependency(t *testing.T, path, content string, want bool) {
	t.Helper()

	options, err := unit.Deserialize(strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to parse systemd unit %s: %v", path, err)
	}
	for _, directive := range []string{"Requires", "After"} {
		var dependencies []string
		for _, option := range options {
			if option.Section != "Unit" || option.Name != directive {
				continue
			}
			values := strings.Fields(option.Value)
			if len(values) == 0 {
				dependencies = nil // An empty assignment resets earlier dependencies.
			}
			dependencies = append(dependencies, values...)
		}
		if got := slices.Contains(dependencies, "coreos-metadata.service"); got != want {
			t.Errorf("%s: [Unit] %s contains coreos-metadata.service = %t, want %t", path, directive, got, want)
		}
	}
}
