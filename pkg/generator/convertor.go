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

package generator

import (
	"encoding/json"
	"fmt"

	config "github.com/kinvolk/container-linux-config-transpiler/config"
)

func toIgnition(s string) ([]byte, error) {
	// Convert string to container linux config
	cfg, ast, report := config.Parse([]byte(s))

	// Check if report has any errors
	if report.IsFatal() {
		return nil, fmt.Errorf("failed to validate coreos cloud config: %s", report.String())
	}

	// Convert container linux config to ignition config
	ignCfg, report := config.Convert(cfg, "", ast)
	if len(report.Entries) > 0 {
		return nil, fmt.Errorf("failed to convert container linux config to ignition: %s", report.String())
	}

	out, err := json.Marshal(ignCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ignition config: %w", err)
	}
	return out, nil
}
