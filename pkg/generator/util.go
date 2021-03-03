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

import "strings"

// GetServiceName get the name of the service file if it was a service one.
func GetServiceName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		fileName := parts[len(parts)-1]
		if fileParts := strings.SplitAfter(fileName, "."); len(fileParts) > 0 &&
			fileParts[len(fileParts)-1] == "service" {
			return fileName
		}
	}

	return ""
}
