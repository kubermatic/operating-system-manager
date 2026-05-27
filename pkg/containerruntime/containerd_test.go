/*
Copyright 2024 The Operating System Manager contributors.

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

package containerruntime

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"testing"

	testUtil "k8c.io/operating-system-manager/pkg/test/util"
)

var update = flag.Bool("update", false, "update testdata files")

func TestContainerd_Configs(t *testing.T) {
	tests := []struct {
		name string
		eng  *Containerd
	}{
		{
			name: "simple",
			eng:  &Containerd{},
		},
		{
			name: "override path",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {"https://override.tld?kubermatic=override_path%3Dtrue"},
				},
			},
		},
		{
			name: "empty kubermatic param",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {"https://empty.tld?kubermatic="},
				},
			},
		},
		{
			name: "broken kubermatic param",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {"https://broken.tld?kubermatic=override_path%3Dzzzz"},
				},
			},
		},
		{
			name: "first endpoint with override_path",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {
						"https://host1.reg.tld?kubermatic=override_path%3Dtrue",
						"https://host2.reg.tld",
					},
				},
			},
		},
		{
			name: "second endpoint with override_path",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {
						"https://host1.reg.tld",
						"https://host2.reg.tld?kubermatic=override_path%3Dtrue",
					},
				},
			},
		},
		{
			name: "docker.io with override_path mirror",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"docker.io": {"https://harbor.example.com/v2/proxy-docker-io?kubermatic=override_path%3Dtrue"},
				},
			},
		},
		{
			name: "registry in subpath",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"gitlab.com": {"https://mirror.gitlab.com/project/repo?kubermatic=override_path%3Dtrue"},
				},
			},
		},
		{
			name: "registry in subpath without override_path",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"gitlab.com": {"https://mirror.gitlab.com/project/repo"},
				},
			},
		},
		{
			name: "insecure registry",
			eng: &Containerd{
				insecureRegistries: []string{"insecure.example.com"},
			},
		},
		{
			name: "insecure registry with mirror",
			eng: &Containerd{
				insecureRegistries: []string{"insecure.example.com"},
				registryMirrors: map[string][]string{
					"insecure.example.com": {"https://mirror.insecure.example.com"},
				},
			},
		},
		{
			name: "insecure and override_path",
			eng: &Containerd{
				insecureRegistries: []string{"insecure.example.com"},
				registryMirrors: map[string][]string{
					"insecure.example.com": {"https://mirror.example.com/v2/proxy?kubermatic=override_path%3Dtrue"},
				},
			},
		},
		{
			name: "wildcard mirror",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"*": {"https://mirror.example.com"},
				},
			},
		},
		{
			name: "default mirror",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"_default": {"https://mirror.example.com"},
				},
			},
		},
		{
			name: "registry credentials",
			eng: &Containerd{
				registryCredentials: map[string]AuthConfig{
					"gcr.io": {Username: "user", Password: "pass"},
				},
			},
		},
		{
			name: "registry credentials with url scheme",
			eng: &Containerd{
				registryCredentials: map[string]AuthConfig{
					"https://my-registry.example.com": {Username: "user", Password: "pass"},
				},
			},
		},
		{
			name: "mirror without scheme",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {"mirror.example.com"},
				},
			},
		},
		{
			name: "harbor subpath mirrors with override_path",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"docker.io":       {"https://harbor.example.com/v2/proxy-docker-io?kubermatic=override_path%3Dtrue"},
					"gcr.io":          {"https://harbor.example.com/v2/proxy-gcr-io?kubermatic=override_path%3Dtrue"},
					"ghcr.io":         {"https://harbor.example.com/v2/proxy-ghcr-io?kubermatic=override_path%3Dtrue"},
					"quay.io":         {"https://harbor.example.com/v2/proxy-quay-io?kubermatic=override_path%3Dtrue"},
					"registry.k8s.io": {"https://harbor.example.com/v2/proxy-k8s-io?kubermatic=override_path%3Dtrue"},
				},
			},
		},
		{
			name: "two mirrors same registry one with override_path one without",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"ghcr.io": {
						"https://harbor.example.com/v2/proxy-ghcr-io?kubermatic=override_path%3Dtrue",
						"https://harbor.example.com",
					},
				},
			},
		},
		{
			name: "mixed registries",
			eng: &Containerd{
				insecureRegistries: []string{"insecure.example.com"},
				registryMirrors: map[string][]string{
					"docker.io":            {"https://harbor.example.com/v2/proxy-docker-io?kubermatic=override_path%3Dtrue"},
					"quay.io":              {"https://harbor.example.com/v2/proxy-quay-io?kubermatic=override_path%3Dtrue"},
					"reg.tld":              {"https://simple-mirror.example.com"},
					"insecure.example.com": {"https://mirror.example.com"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder

			// Main containerd config.toml
			config, err := tt.eng.Config()
			if err != nil {
				t.Fatalf("Config() error = %v", err)
			}
			fmt.Fprintf(&buf, "# %s\n", tt.eng.ConfigFileName())
			buf.WriteString(config)

			// Per-registry hosts.toml files
			hostConfigs, err := tt.eng.RegistryHostConfigs()
			if err != nil {
				t.Fatalf("RegistryHostConfigs() error = %v", err)
			}

			// Idempotency: second call must produce identical output.
			hostConfigs2, err := tt.eng.RegistryHostConfigs()
			if err != nil {
				t.Fatalf("RegistryHostConfigs() second call error = %v", err)
			}
			if len(hostConfigs) != len(hostConfigs2) {
				t.Errorf("idempotency: first call returned %d entries, second call returned %d", len(hostConfigs), len(hostConfigs2))
			}
			for path, content := range hostConfigs {
				if content2, ok := hostConfigs2[path]; !ok {
					t.Errorf("idempotency: path %q missing from second call", path)
				} else if content != content2 {
					t.Errorf("idempotency: path %q differs between calls:\nfirst:\n%s\nsecond:\n%s", path, content, content2)
				}
			}

			paths := make([]string, 0, len(hostConfigs))
			for path := range hostConfigs {
				paths = append(paths, path)
			}
			sort.Strings(paths)

			for _, path := range paths {
				buf.WriteString("---\n")
				fmt.Fprintf(&buf, "# %s\n", path)
				buf.WriteString(hostConfigs[path])
			}

			testUtil.CompareOutput(t, testUtil.FSGoldenName(t), buf.String(), *update)
		})
	}
}

// TestRegistryHostConfigs_SourceNotMutated is a regression test for
// https://github.com/kubermatic/kubermatic/issues/15886.
//
// RegistryHostConfigs() must not modify eng.registryMirrors. Previously the
// function stripped ?kubermatic= from the backing slice in-place, so a second
// MachineDeployment reconciliation in the same OSM pod saw already-stripped
// URLs, never parsed override_path=true, and silently omitted override_path = true
// from the generated hosts.toml.
func TestRegistryHostConfigs_SourceNotMutated(t *testing.T) {
	const mirror = "https://harbor.example.com/v2/proxy-quay-io?kubermatic=override_path%3Dtrue"

	eng := &Containerd{
		registryMirrors: map[string][]string{
			"quay.io": {mirror},
		},
	}

	const wantPath = "/etc/containerd/certs.d/quay.io/hosts.toml"
	const wantSubstr = "override_path = true"

	for i := range 3 {
		configs, err := eng.RegistryHostConfigs()
		if err != nil {
			t.Fatalf("call %d: RegistryHostConfigs() error = %v", i+1, err)
		}

		content, ok := configs[wantPath]
		if !ok {
			t.Fatalf("call %d: path %q not found", i+1, wantPath)
		}
		if !strings.Contains(content, wantSubstr) {
			t.Errorf("call %d: path %q missing %q — source was mutated between calls:\n%s", i+1, wantPath, wantSubstr, content)
		}

		if got := eng.registryMirrors["quay.io"][0]; got != mirror {
			t.Fatalf("call %d: source mirror was mutated: want %q, got %q", i+1, mirror, got)
		}
	}
}
