/*
Copyright 2022 The Operating System Manager contributors.

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

package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
)

func init() {
	klog.InitFlags(nil)
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatalf("failed to add clusterv1alpha1 to scheme: %v", err)
	}
}

const (
	AWSManifest   = "./testdata/machinedeployment-aws.yaml"
	AzureManifest = "./testdata/machinedeployment-azure.yaml"
	GCEManifest   = "./testdata/machinedeployment-gce.yaml"
)

var testRunIdentifier = flag.String("identifier", "local", "The unique identifier for this test run")

// TestOSMProvisioningAWS - a test suite that exercises AWS provider
// by requesting nodes with different combination of container runtime type, container runtime version and the OS flavour.
func TestOSMProvisioningAWS(t *testing.T) {
	t.Parallel()

	// test data
	awsKeyID := os.Getenv("AWS_E2E_TESTS_KEY_ID")
	awsSecret := os.Getenv("AWS_E2E_TESTS_SECRET")
	if len(awsKeyID) == 0 || len(awsSecret) == 0 {
		t.Fatal("unable to run the test suite, AWS_E2E_TESTS_KEY_ID or AWS_E2E_TESTS_SECRET environment variables cannot be empty")
	}
	selector := Not(OsSelector("sles"))
	// act
	params := []string{fmt.Sprintf("<< AWS_ACCESS_KEY_ID >>=%s", awsKeyID),
		fmt.Sprintf("<< AWS_SECRET_ACCESS_KEY >>=%s", awsSecret),
	}

	runScenarios(t, selector, params, AWSManifest, fmt.Sprintf("aws-%s", *testRunIdentifier))
}
