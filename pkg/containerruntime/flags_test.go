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
