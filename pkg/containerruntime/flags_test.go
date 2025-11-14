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
	"reflect"
	"testing"

	"github.com/go-test/deep"
)

func TestRegistryMirrorsFlags_Set(t *testing.T) {
	tests := []struct {
		name    string
		want    RegistryMirrorsFlags
		input   string
		wantErr bool
	}{
		{
			name:  "simple",
			input: "reg.tld=https://registry.reg.tld",
			want: RegistryMirrorsFlags{
				"reg.tld": []string{"https://registry.reg.tld"},
			},
		},
		{
			name:  "with params",
			input: "reg.tld=https://registry.reg.tld?param1=value1",
			want: RegistryMirrorsFlags{
				"reg.tld": []string{"https://registry.reg.tld?param1=value1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rmf := RegistryMirrorsFlags{}

			if err := rmf.Set(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("RegistryMirrorsFlags.Set() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(deep.Equal(rmf, tt.want)) != 0 {
				t.Errorf("%v not equal to %v", rmf, tt.want)
			}
		})
	}
}

// TestRegistryMirrorsFlagsWithFlagParse verifies the behavior of RegistryMirrorsFlags
// when used with Go's flag.FlagSet.Parse(), specifically testing the scenario where
// empty strings appear between repeated flags.
//
// KKP had a bug (https://github.com/kubermatic/kubermatic/pull/15154) where it was
// incorrectly constructing args for the OSM deployment by inserting empty strings
// after each containerd registry mirror flag, like:
//
//	["-node-containerd-registry-mirrors=docker.io=mirror1", "",
//	 "-node-containerd-registry-mirrors=docker.io=mirror2", ""]
//
// This causes only the first node containerd registry mirror to be parsed, causes
// silent missing of subsequent mirrors.
func TestRegistryMirrorsFlagsWithFlagParse(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedResult RegistryMirrorsFlags
		expectedArgs   []string
		expectError    bool
	}{
		{
			name: "multiple flags without empty strings (expected format)",
			args: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1.docker.io", "mirror2.docker.io"},
				"gcr.io":    []string{"mirror.gcr.io"},
			},
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			// so, in this case, we replicate the https://github.com/kubermatic/kubermatic/pull/15154
			// bug where KKP was passing empty strings between flags.
			// it causes OSM to only see the first registry mirror, ignoring the rest, with no error.
			name: "multiple flags with empty strings between them (KKP bug)",
			args: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
				"",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
				"",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1.docker.io"},
			},
			expectedArgs: []string{
				"",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
				"",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
				"",
			},
			expectError: false,
		},
		{
			name: "single flag works correctly",
			args: []string{
				"-node-containerd-registry-mirrors=quay.io=mirror.quay.io",
			},
			expectedResult: RegistryMirrorsFlags{
				"quay.io": []string{"mirror.quay.io"},
			},
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			name: "empty string at start stops all flag parsing",
			args: []string{
				"",
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
			expectedResult: RegistryMirrorsFlags{},
			expectedArgs: []string{
				"",
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
			},
			expectError: false,
		},
		{
			name: "double dash stops flag parsing (standard behavior)",
			args: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"--",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1.docker.io"},
			},
			expectedArgs: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
			},
			expectError: false,
		},
		{
			name: "non-flag argument stops parsing",
			args: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"non-flag-arg",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1.docker.io"},
			},
			expectedArgs: []string{
				"non-flag-arg",
				"-node-containerd-registry-mirrors=docker.io=mirror2.docker.io",
			},
			expectError: false,
		},
		{
			name: "invalid flag format causes error",
			args: []string{
				"-node-containerd-registry-mirrors=invalid-no-equals",
			},
			expectedResult: nil,
			expectedArgs:   nil,
			expectError:    true,
		},
		{
			name: "mixed with other flags before empty string",
			args: []string{
				"-some-other-flag=value",
				"-node-containerd-registry-mirrors=docker.io=mirror1.docker.io",
				"-node-containerd-registry-mirrors=gcr.io=mirror.gcr.io",
				"-another-flag=value2",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1.docker.io"},
				"gcr.io":    []string{"mirror.gcr.io"},
			},
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			// The entire comma-separated string is treated as a single mirror URL.
			// the comma separated format is decalted through legacy -node-registry-mirrors flag, not
			// by -node-containerd-registry-mirrors.
			name: "comma-separated values are NOT parsed (treated as single value)",
			args: []string{
				"-node-containerd-registry-mirrors=docker.io=mirror1,docker.io=mirror2,quay.io=mirror3",
			},
			expectedResult: RegistryMirrorsFlags{
				"docker.io": []string{"mirror1,docker.io=mirror2,quay.io=mirror3"},
			},
			expectedArgs: []string{},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			rmf := make(RegistryMirrorsFlags)
			fs.Var(&rmf, "node-containerd-registry-mirrors", "Registry mirrors for containerd")

			fs.String("some-other-flag", "", "dummy flag")
			fs.String("another-flag", "", "dummy flag")

			err := fs.Parse(tt.args)

			if tt.expectError && err == nil {
				t.Errorf("expected parsing error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected parsing error: %v", err)
			}

			if !tt.expectError {
				if tt.expectedResult != nil {
					if !reflect.DeepEqual(rmf, tt.expectedResult) {
						t.Errorf(
							"registry mirrors mismatch\nGot:  %+v\nWant: %+v",
							rmf,
							tt.expectedResult,
						)
					}
				}

				remainingArgs := fs.Args()
				if !reflect.DeepEqual(remainingArgs, tt.expectedArgs) {
					t.Errorf("remaining args mismatch\nGot:  %q\nWant: %q",
						remainingArgs, tt.expectedArgs)
				}
			}
		})
	}
}
