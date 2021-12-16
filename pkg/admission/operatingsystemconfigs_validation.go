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

package admission

import (
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (ad *admissionData) validateOperatingSystemConfigUpdate(oscOld osmv1alpha1.OperatingSystemConfig, oscNew osmv1alpha1.OperatingSystemConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	// Updates for OperatingSystemConfig are not allowed
	if equal := apiequality.Semantic.DeepEqual(oscOld.Spec, oscNew.Spec); !equal {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), oscNew.Name, "OperatingSystemConfig is immutable and updates are not alloed"))
	}
	return allErrs
}
