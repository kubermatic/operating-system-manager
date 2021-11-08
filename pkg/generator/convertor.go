/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package generator

import (
	"encoding/json"
	"fmt"

	ignitionconfig "github.com/coreos/ignition/v2/config/v3_3"
)

func toIgnition(s string) ([]byte, error) {
	// Convert to ignition
	cfg, report, err := ignitionconfig.Parse([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("failed to validate coreos cloud config: %v", err)
	}
	// Check if report has any errors
	if report.IsFatal() {
		return nil, fmt.Errorf("failed to validate coreos cloud config: %s", report.String())
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ignition config: %v", err)
	}

	return out, nil
}
