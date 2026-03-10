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
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Containerd struct {
	insecureRegistries                 []string
	registryMirrors                    map[string][]string
	sandboxImage                       string
	registryCredentials                map[string]AuthConfig
	version                            string
	deviceOwnershipFromSecurityContext bool
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
	Version int                `toml:"version"`
	Metrics *containerdMetrics `toml:"metrics"`
	Plugins map[string]any     `toml:"plugins"`
}

type containerdMetrics struct {
	Address string `toml:"address"`
}

// containerdCRIImagesPlugin represents the "io.containerd.cri.v1.images" plugin in containerd 2.x.
type containerdCRIImagesPlugin struct {
	DiscardUnpackedLayers bool                    `toml:"discard_unpacked_layers"`
	PinnedImages          *containerdPinnedImages `toml:"pinned_images,omitempty"`
	Registry              *containerdCRIRegistry  `toml:"registry"`
}

// containerdPinnedImages represents the pinned_images config in containerd 2.x.
type containerdPinnedImages struct {
	Sandbox string `toml:"sandbox,omitempty"`
}

// containerdCRIRuntimePlugin represents the "io.containerd.cri.v1.runtime" plugin in containerd 2.x.
type containerdCRIRuntimePlugin struct {
	Containerd                         *containerdCRISettings  `toml:"containerd"`
	DeviceOwnershipFromSecurityContext bool                    `toml:"device_ownership_from_security_context"`
	CNI                                *containerdCRICNIConfig `toml:"cni"`
}

// containerdCRICNIConfig represents the CNI config under the runtime plugin in containerd 2.x.
type containerdCRICNIConfig struct {
	BinDirs []string `toml:"bin_dirs"`
	ConfDir string   `toml:"conf_dir"`
}

type containerdCRISettings struct {
	Runtimes map[string]containerdCRIRuntime `toml:"runtimes"`
}

type containerdCRIRuntime struct {
	RuntimeType string `toml:"runtime_type"`
	Options     any    `toml:"options"`
}

type containerdCRIRuncOptions struct {
	SystemdCgroup bool `toml:"SystemdCgroup"`
}

type containerdCRIRegistry struct {
	ConfigPath string `toml:"config_path"`
}

// registryHostConfig holds the parsed mirror configuration for a single registry,
// used internally when building hosts.toml files.
type registryHostConfig struct {
	endpoints    []string
	overridePath bool
	insecure     bool
	auth         *AuthConfig
}

// hostsTomlConfig represents the top-level structure of a hosts.toml file.
type hostsTomlConfig struct {
	Server string                     `toml:"server"`
	Host   map[string]hostEntryConfig `toml:"host,omitempty"`
}

// hostEntryConfig represents a single host entry in a hosts.toml file.
type hostEntryConfig struct {
	Capabilities []string          `toml:"capabilities"`
	SkipVerify   bool              `toml:"skip_verify,omitempty"`
	OverridePath bool              `toml:"override_path,omitempty"`
	Header       map[string]string `toml:"header,omitempty"`
}

func (eng *Containerd) Config() (string, error) {
	criImagesPlugin := containerdCRIImagesPlugin{
		DiscardUnpackedLayers: false,
		Registry: &containerdCRIRegistry{
			ConfigPath: "/etc/containerd/certs.d",
		},
	}

	if eng.sandboxImage != "" {
		criImagesPlugin.PinnedImages = &containerdPinnedImages{
			Sandbox: eng.sandboxImage,
		}
	}

	criRuntimePlugin := containerdCRIRuntimePlugin{
		DeviceOwnershipFromSecurityContext: eng.deviceOwnershipFromSecurityContext,
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
		CNI: &containerdCRICNIConfig{
			BinDirs: []string{"/opt/cni/bin"},
			ConfDir: "/etc/cni/net.d",
		},
	}

	cfg := containerdConfigManifest{
		Version: 3,
		Metrics: &containerdMetrics{
			// metrics available at http://127.0.0.1:1338/v1/metrics
			Address: "127.0.0.1:1338",
		},

		Plugins: map[string]interface{}{
			"io.containerd.cri.v1.images":  criImagesPlugin,
			"io.containerd.cri.v1.runtime": criRuntimePlugin,
		},
	}

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	err := enc.Encode(cfg)

	return buf.String(), err
}

// buildRegistryHostConfigs processes the registry mirrors, insecure registries,
// and registry credentials using the same logic that was previously used to
// build the inline mirrors config, and returns a per-registry configuration.
func (eng *Containerd) buildRegistryHostConfigs() map[string]*registryHostConfig {
	configs := make(map[string]*registryHostConfig)

	// Start with default docker.io entry
	configs["docker.io"] = &registryHostConfig{
		endpoints: []string{"https://registry-1.docker.io"},
	}

	// Process registry mirrors — same logic as the original Config() method
	for registryName := range eng.registryMirrors {
		if _, ok := configs[registryName]; !ok {
			configs[registryName] = &registryHostConfig{}
		}
		rc := configs[registryName]
		rc.endpoints = eng.registryMirrors[registryName]

		var overridePath bool
		for i, endpoint := range rc.endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}

			endpointQS := endpointURL.Query()
			if kubermaticParams := endpointQS.Get("kubermatic"); endpointQS.Has("kubermatic") {
				endpointQS.Del("kubermatic")
				endpointURL.RawQuery = endpointQS.Encode()
				rc.endpoints[i] = endpointURL.String()
				params, err := url.QueryUnescape(kubermaticParams)
				if err != nil {
					continue
				}

				paramsValues, err := url.ParseQuery(params)
				if err != nil {
					continue
				}

				if !overridePath {
					overridePath, _ = strconv.ParseBool(paramsValues.Get("override_path"))
				}
			}
		}
		rc.overridePath = overridePath
	}

	// Process insecure registries
	for _, registry := range eng.insecureRegistries {
		if _, ok := configs[registry]; !ok {
			configs[registry] = &registryHostConfig{}
		}
		configs[registry].insecure = true
	}

	// Process registry credentials
	for registry, auth := range eng.registryCredentials {
		if _, ok := configs[registry]; !ok {
			configs[registry] = &registryHostConfig{}
		}
		auth := auth
		configs[registry].auth = &auth
	}

	return configs
}

// RegistryHostConfigs returns a map of file path to file content for containerd
// registry host configuration files. Each key is a path like
// "/etc/containerd/certs.d/<registry>/hosts.toml" and the value is the TOML content.
// This preserves all the existing logic for kubermatic params, override_path,
// insecure registries, and registry credentials.
func (eng *Containerd) RegistryHostConfigs() map[string]string {
	result := make(map[string]string)
	configs := eng.buildRegistryHostConfigs()

	// Sort registry names for deterministic output
	registryNames := make([]string, 0, len(configs))
	for name := range configs {
		registryNames = append(registryNames, name)
	}
	sort.Strings(registryNames)

	for _, registryName := range registryNames {
		rc := configs[registryName]

		// Determine the server URL (the upstream registry)
		serverURL := fmt.Sprintf("https://%s", registryName)
		if registryName == "docker.io" {
			serverURL = "https://registry-1.docker.io"
		}

		cfg := hostsTomlConfig{
			Server: serverURL,
			Host:   make(map[string]hostEntryConfig),
		}

		// Add mirror host entries
		for _, endpoint := range rc.endpoints {
			if !strings.HasPrefix(endpoint, "http") {
				endpoint = "https://" + endpoint
			}

			// Check if there is a registry credential, matching the host
			mirrorEndpoint, err := url.Parse(endpoint)
			if err != nil {
				continue
			}

			var foundAuthConfig *AuthConfig

			for otherRegistry, authConfig := range eng.registryCredentials {
				otherParsedUrl, err := url.Parse(otherRegistry)
				if otherRegistry == mirrorEndpoint.Host || err == nil && otherParsedUrl.Host == mirrorEndpoint.Host {
					foundAuthConfig = &authConfig
					break
				}
			}

			var header map[string]string
			if foundAuthConfig != nil {
				header = eng.buildHeaders(foundAuthConfig)
			}

			cfg.Host[endpoint] = hostEntryConfig{
				Capabilities: []string{"pull", "resolve"},
				OverridePath: rc.overridePath,
				SkipVerify:   rc.insecure,
				Header:       header,
			}
		}

		// If insecure registry has no endpoints, add its own endpoint
		if rc.insecure && len(rc.endpoints) == 0 {
			cfg.Host[serverURL] = hostEntryConfig{
				Capabilities: []string{"pull", "resolve", "push"},
				SkipVerify:   true,
				Header:       eng.buildHeaders(rc.auth),
			}
		}

		if rc.auth != nil {
			if _, ok := cfg.Host[serverURL]; !ok {
				cfg.Host[serverURL] = hostEntryConfig{
					Capabilities: []string{"pull", "resolve", "push"},
					SkipVerify:   false,
					Header:       eng.buildHeaders(rc.auth),
				}
			}
		}

		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.Indent = ""
		_ = enc.Encode(cfg)

		// Remove empty parent table header that TOML encoder generates for nested maps
		output := strings.ReplaceAll(buf.String(), "[host]\n", "")

		filePath := fmt.Sprintf("/etc/containerd/certs.d/%s/hosts.toml", registryName)
		result[filePath] = output
	}

	return result
}

func (eng *Containerd) buildHeaders(authConfig *AuthConfig) map[string]string {
	if authConfig == nil {
		return nil
	}
	credentials := fmt.Sprintf("%s:%s", authConfig.Username, authConfig.Password)
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))

	return map[string]string{
		"Authorization": fmt.Sprintf("Basic %s", encodedCredentials),
	}
}
