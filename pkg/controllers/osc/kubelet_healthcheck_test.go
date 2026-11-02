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

package osc

import (
	"slices"
	"strings"
	"testing"

	"github.com/coreos/go-systemd/unit"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

func TestKubeletHealthcheckUnits(t *testing.T) {
	for _, profile := range []string{"amzn2", "flatcar", "flatcar-cloud-init", "rhel", "rockylinux", "ubuntu"} {
		t.Run(profile, func(t *testing.T) {
			osp := &osmv1alpha1.OperatingSystemProfile{}
			if err := loadFile(osp, defaultOSPPathPrefix+"osp-"+profile+".yaml"); err != nil {
				t.Fatalf("failed to load operating system profile: %v", err)
			}
			files := slices.Concat(osp.Spec.BootstrapConfig.Files, osp.Spec.ProvisioningConfig.Files)

			for _, name := range []string{"kubelet-healthcheck.service", "kubelet-healthcheck.timer"} {
				t.Run(name, func(t *testing.T) {
					path := "/etc/systemd/system/" + name
					index := slices.IndexFunc(files, func(file osmv1alpha1.File) bool {
						return file.Path == path
					})
					if index == -1 {
						t.Fatalf("profile is missing %s", path)
					}
					file := files[index]
					if file.Content.Inline == nil || strings.TrimSpace(file.Content.Inline.Data) == "" {
						t.Fatalf("%s has no inline unit content", path)
					}

					// These units have no template expressions. Parse their contents to
					// catch missing continuations that turn shell statements into unit keys.
					if _, err := unit.Deserialize(strings.NewReader(file.Content.Inline.Data)); err != nil {
						t.Fatalf("invalid systemd unit %s: %v", path, err)
					}
				})
			}
		})
	}
}
