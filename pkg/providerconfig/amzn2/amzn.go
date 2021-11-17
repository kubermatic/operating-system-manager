/*
Copyright 2021 The Operating System Manager contributors.

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

package amzn2

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

// Config contains specific configuration for Amazon Linux 2.
type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

func DefaultConfig(operatingSystemSpec runtime.RawExtension) runtime.RawExtension {
	if operatingSystemSpec.Raw == nil {
		operatingSystemSpec.Raw, _ = json.Marshal(Config{})
	}

	return operatingSystemSpec
}

// LoadConfig retrieves the Amazon Linux 2 configuration from raw data.
func LoadConfig(r runtime.RawExtension) (*Config, error) {
	r = DefaultConfig(r)
	cfg := Config{}

	if err := json.Unmarshal(r.Raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
