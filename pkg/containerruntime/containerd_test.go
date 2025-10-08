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
	"testing"

	testUtil "k8c.io/operating-system-manager/pkg/test/util"
)

var update = flag.Bool("update", false, "update testdata files")

func TestContainerd_Config(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		eng     Engine
	}{
		{
			name: "simple",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {"https://simple.tld"},
				},
			},
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
			name: "second endpoint",
			eng: &Containerd{
				registryMirrors: map[string][]string{
					"reg.tld": {
						"https://host1.reg.tld",
						"https://host2.reg.tld?kubermatic=override_path%3Dtrue",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buff, gotErr := tt.eng.Config()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Config() failed: %v", gotErr)
				}
				return
			}

			testUtil.CompareOutput(t, testUtil.FSGoldenName(t), buff, *update)
		})
	}
}
