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
	"strings"

	"github.com/BurntSushi/toml"
)

type Containerd struct {
	insecureRegistries  []string
	registryMirrors     map[string][]string
	sandboxImage        string
	registryCredentials map[string]AuthConfig
	version             string
}

func (eng *Containerd) ConfigFileName() string {
	return "/etc/containerd/config.toml"
}

func (eng *Containerd) AuthConfig() (string, error) {
	return "", nil
}

func (eng *Containerd) AuthConfigFileName() string {
	return ""
}

func (eng *Containerd) KubeletFlags() []string {
	return []string{
		"--container-runtime-endpoint=unix:///run/containerd/containerd.sock",
	}
}

func (eng *Containerd) String() string {
	return containerdName
}

type containerdConfigManifest struct {
	Version int                    `toml:"version"`
	Metrics *containerdMetrics     `toml:"metrics"`
	Plugins map[string]interface{} `toml:"plugins"`
}

type containerdMetrics struct {
	Address string `toml:"address"`
}

type containerdCRIPlugin struct {
	Containerd   *containerdCRISettings `toml:"containerd"`
	Registry     *containerdCRIRegistry `toml:"registry"`
	SandboxImage string                 `toml:"sandbox_image,omitempty"`
}

type containerdCRISettings struct {
	Runtimes map[string]containerdCRIRuntime `toml:"runtimes"`
}

type containerdCRIRuntime struct {
	RuntimeType string      `toml:"runtime_type"`
	Options     interface{} `toml:"options"`
}

type containerdCRIRuncOptions struct {
	SystemdCgroup bool
}

type containerdCRIRegistry struct {
	Mirrors map[string]containerdRegistryMirror `toml:"mirrors"`
	Configs map[string]containerdRegistryConfig `toml:"configs"`
}

type containerdRegistryMirror struct {
	Endpoint []string `toml:"endpoint"`
}

type containerdRegistryConfig struct {
	TLS  *containerdRegistryTLSConfig `toml:"tls"`
	Auth *AuthConfig                  `toml:"auth"`
}

type containerdRegistryTLSConfig struct {
	InsecureSkipVerify bool `toml:"insecure_skip_verify"`
}

func (eng *Containerd) Config() (string, error) {
	criPlugin := containerdCRIPlugin{
		SandboxImage: eng.sandboxImage,
		Containerd: &containerdCRISettings{
			Runtimes: map[string]containerdCRIRuntime{
				"runc": {
					RuntimeType: "io.containerd.runc.v2",
					Options: containerdCRIRuncOptions{
						SystemdCgroup: true,
					},
				},
			},
		},
		Registry: &containerdCRIRegistry{
			Mirrors: map[string]containerdRegistryMirror{
				"docker.io": {
					Endpoint: []string{"https://registry-1.docker.io"},
				},
			},
		},
	}

	for registryName := range eng.registryMirrors {
		registry := criPlugin.Registry.Mirrors[registryName]
		registry.Endpoint = eng.registryMirrors[registryName]
		criPlugin.Registry.Mirrors[registryName] = registry
	}

	if len(eng.insecureRegistries) != 0 || len(eng.registryCredentials) != 0 {
		criPlugin.Registry.Configs = map[string]containerdRegistryConfig{}
	}

	for _, registry := range eng.insecureRegistries {
		criPlugin.Registry.Configs[registry] = containerdRegistryConfig{
			TLS: &containerdRegistryTLSConfig{
				InsecureSkipVerify: true,
			},
		}
	}

	for registry, auth := range eng.registryCredentials {
		regConfig := criPlugin.Registry.Configs[registry]
		auth := auth
		regConfig.Auth = &auth
		criPlugin.Registry.Configs[registry] = regConfig
	}

	cfg := containerdConfigManifest{
		Version: 2,
		Metrics: &containerdMetrics{
			// metrics available at http://127.0.0.1:1338/v1/metrics
			Address: "127.0.0.1:1338",
		},

		Plugins: map[string]interface{}{
			"io.containerd.grpc.v1.cri": criPlugin,
		},
	}

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	err := enc.Encode(cfg)

	return buf.String(), err
}
