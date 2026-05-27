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
	ConfigPath string                              `toml:"config_path"`
	Configs    map[string]containerdRegistryConfig `toml:"configs,omitempty"`
}

type containerdRegistryConfig struct {
	Auth *AuthConfig `toml:"auth,omitempty"`
}

// registryEndpoint holds a single mirror URL and its per-endpoint settings
// parsed from the kubermatic sideband query parameter.
type registryEndpoint struct {
	url          string
	overridePath bool
}

// registryHostConfig holds the parsed mirror configuration for a single registry,
// used internally when building hosts.toml files.
type registryHostConfig struct {
	endpoints []registryEndpoint
	insecure  bool
}

// hostsTomlConfig represents the top-level structure of a hosts.toml file.
type hostsTomlConfig struct {
	Server string                     `toml:"server,omitempty"`
	Host   map[string]hostEntryConfig `toml:"host,omitempty"`
}

// hostEntryConfig represents a single host entry in a hosts.toml file.
type hostEntryConfig struct {
	Capabilities []string `toml:"capabilities"`
	SkipVerify   bool     `toml:"skip_verify,omitempty"`
	OverridePath bool     `toml:"override_path,omitempty"`
}

func (eng *Containerd) Config() (string, error) {
	criRegistry := &containerdCRIRegistry{
		ConfigPath: "/etc/containerd/certs.d",
	}

	// Add registry credentials to CRI config for authentication.
	// Per containerd v2 docs, auth is configured under
	// [plugins."io.containerd.cri.v1.images".registry.configs."<registry>".auth]
	// The registry key must be a host (with optional port), not a URL.
	// Docker config JSON uses full URLs (e.g. "https://gcr.io") as keys,
	// so we strip the scheme if present.
	if len(eng.registryCredentials) > 0 {
		criRegistry.Configs = make(map[string]containerdRegistryConfig, len(eng.registryCredentials))
		for registry, auth := range eng.registryCredentials {
			auth := auth
			host := registry
			if u, err := url.Parse(registry); err == nil && u.Host != "" {
				host = u.Host
			}
			criRegistry.Configs[host] = containerdRegistryConfig{
				Auth: &auth,
			}
		}
	}

	criImagesPlugin := containerdCRIImagesPlugin{
		DiscardUnpackedLayers: false,
		Registry:              criRegistry,
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

	// docker.io always gets a hosts.toml with registry-1.docker.io as the default
	// upstream entry. If the user configures explicit mirrors below, those replace
	// this default so only the user-configured mirrors appear.
	configs["docker.io"] = &registryHostConfig{
		endpoints: []registryEndpoint{{url: "https://registry-1.docker.io"}},
	}

	// Process registry mirrors: parse per-endpoint kubermatic params read-only.
	for registryName, mirrorURLs := range eng.registryMirrors {
		if _, ok := configs[registryName]; !ok {
			configs[registryName] = &registryHostConfig{}
		}
		regCfg := configs[registryName]
		// User-configured mirrors replace any pre-seeded defaults.
		regCfg.endpoints = nil

		for _, mirrorURL := range mirrorURLs {
			endpoint := registryEndpoint{url: stripKubermaticParam(mirrorURL)}

			if parsedURL, err := url.Parse(mirrorURL); err == nil {
				query := parsedURL.Query()
				if query.Has("kubermatic") {
					if kubermaticParams, err := url.QueryUnescape(query.Get("kubermatic")); err == nil {
						if kubermaticValues, err := url.ParseQuery(kubermaticParams); err == nil {
							endpoint.overridePath, _ = strconv.ParseBool(kubermaticValues.Get("override_path"))
						}
					}
				}
			}

			regCfg.endpoints = append(regCfg.endpoints, endpoint)
		}
	}

	// Process insecure registries
	for _, registryName := range eng.insecureRegistries {
		if _, ok := configs[registryName]; !ok {
			configs[registryName] = &registryHostConfig{}
		}
		configs[registryName].insecure = true
	}

	return configs
}

// RegistryHostConfigs returns a map of file path to file content for containerd
// registry host configuration files. Each key is a path like
// "/etc/containerd/certs.d/<registry>/hosts.toml" and the value is the TOML content.
// This preserves all the existing logic for kubermatic params, override_path,
// insecure registries, and registry credentials.
func (eng *Containerd) RegistryHostConfigs() (map[string]string, error) {
	result := make(map[string]string)
	configs := eng.buildRegistryHostConfigs()

	// Sort registry names for deterministic output
	registryNames := make([]string, 0, len(configs))
	for name := range configs {
		registryNames = append(registryNames, name)
	}
	sort.Strings(registryNames)

	for _, registryName := range registryNames {
		regCfg := configs[registryName]

		// Skip registries that have no mirrors and are not insecure —
		// a hosts.toml with only a server URL adds no value over containerd defaults.
		if len(regCfg.endpoints) == 0 && !regCfg.insecure {
			continue
		}

		// Determine the server URL and certs.d directory name for this registry.
		// See: https://github.com/containerd/containerd/blob/546ce382/core/remotes/docker/config/hosts.go#L430-L431
		registryDir := registryHost(registryName)
		var serverURL string
		switch registryDir {
		case "docker.io":
			serverURL = "https://registry-1.docker.io"
		case "*", "_default":
			// Wildcard / default catch-all: containerd uses the _default directory
			// as a fallback for any registry without its own hosts.toml.
			// No server URL is set because there is no single upstream.
			registryDir = "_default"
		default:
			serverURL = registryName
		}

		hostsCfg := hostsTomlConfig{
			Server: serverURL,
			Host:   make(map[string]hostEntryConfig),
		}

		// Add per-endpoint mirror host entries.
		for _, endpoint := range regCfg.endpoints {
			hostsCfg.Host[endpoint.url] = hostEntryConfig{
				Capabilities: []string{"pull", "resolve"},
				OverridePath: endpoint.overridePath,
				SkipVerify:   regCfg.insecure,
			}
		}

		// If no mirrors are configured but the registry is insecure,
		// create a self-referencing host entry.
		if len(regCfg.endpoints) == 0 && regCfg.insecure {
			hostsCfg.Host[serverURL] = hostEntryConfig{
				Capabilities: []string{"pull", "resolve"},
				SkipVerify:   regCfg.insecure,
			}
		}

		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.Indent = ""

		if err := enc.Encode(hostsCfg); err != nil {
			return nil, fmt.Errorf("encoding hosts.toml for %s: %w", registryName, err)
		}

		// Remove empty parent table header that TOML encoder generates for nested maps
		output := strings.ReplaceAll(buf.String(), "[host]\n", "")

		filePath := fmt.Sprintf("/etc/containerd/certs.d/%s/hosts.toml", registryDir)
		result[filePath] = output
	}

	return result, nil
}

// stripKubermaticParam removes the kubermatic sideband query parameter from a
// mirror URL before it is written into hosts.toml. The parameter is an
// OSM-internal marker and is not understood by containerd.
func stripKubermaticParam(mirrorURL string) string {
	parsedURL, err := url.Parse(mirrorURL)
	if err != nil {
		return mirrorURL
	}
	query := parsedURL.Query()
	if !query.Has("kubermatic") {
		return mirrorURL
	}
	query.Del("kubermatic")
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

// registryHost extracts the host[:port] from a registry name,
// stripping any subpath. Containerd's certs.d directory and auth
// config keys only use the host[:port] portion.
func registryHost(registry string) string {
	if i := strings.IndexByte(registry, '/'); i >= 0 {
		return registry[:i]
	}

	return registry
}
