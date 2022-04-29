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

package reconciling

import (
	"encoding/json"

	"github.com/go-test/deep"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	// Kubernetes Objects can be deeper than the default 10 levels.
	deep.MaxDepth = 20
	deep.LogErrors = true
}

// DeepEqual compares both objects for equality
func DeepEqual(a, b metav1.Object) (bool, error) {
	if equality.Semantic.DeepEqual(a, b) {
		return true, nil
	}

	// For some reason unstructured objects returned from the api have types for their fields
	// that are not map[string]interface{} and don't even exist in our codebase like
	// `openshift.infrastructureStatus`, so we have to compare the wire format here.
	// We only do this for unstrucutred as this comparison is pretty expensive.
	if _, isUnstructured := a.(*unstructured.Unstructured); isUnstructured {
		if equal, err := jsonEqual(a, b); err != nil || !equal {
			return false, err
		}

		return true, nil
	}

	// For informational purpose we use deep.equal as it tells us what the difference is.
	// We need to calculate the difference in both ways as deep.equal only does a one-way comparison
	diff := deep.Equal(a, b)
	if diff == nil {
		diff = deep.Equal(b, a)
	}

	return false, nil
}

func jsonEqual(a, b interface{}) (bool, error) {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false, err
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false, err
	}
	return string(aJSON) == string(bJSON), nil
}
