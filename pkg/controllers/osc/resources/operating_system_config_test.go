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

package resources

import (
	"testing"

	mcsdkcommon "k8c.io/machine-controller/sdk/apis/cluster/common"
)

func TestGetJournaldConfigs(t *testing.T) {
	annotationKey := mcsdkcommon.JournaldConfigAnnotationPrefixV1 + "/" + mcsdkcommon.JournaldSystemMaxUse

	testCases := []struct {
		name        string
		annotations map[string]string
		expected    string
		expectErr   bool
	}{
		{
			name:     "default when no annotations are set",
			expected: DefaultJournaldSystemMaxUse,
		},
		{
			name: "default when only other annotations are set",
			annotations: map[string]string{
				mcsdkcommon.KubeletConfigAnnotationPrefixV1 + "/" + mcsdkcommon.MaxPodsKubeletConfig: "110",
			},
			expected: DefaultJournaldSystemMaxUse,
		},
		{
			name:        "custom value",
			annotations: map[string]string{annotationKey: "500M"},
			expected:    "500M",
		},
		{
			name:        "value without suffix",
			annotations: map[string]string{annotationKey: "1073741824"},
			expected:    "1073741824",
		},
		{
			name:        "invalid suffix",
			annotations: map[string]string{annotationKey: "5GB"},
			expectErr:   true,
		},
		{
			name:        "empty value",
			annotations: map[string]string{annotationKey: ""},
			expectErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := getJournaldConfigs(tc.annotations)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.JournaldSystemMaxUse != tc.expected {
				t.Fatalf("expected JournaldSystemMaxUse %q, got %q", tc.expected, cfg.JournaldSystemMaxUse)
			}
		})
	}
}
