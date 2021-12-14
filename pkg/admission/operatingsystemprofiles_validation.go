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

func (ad *admissionData) validateOperatingSystemProfile(osp osmv1alpha1.OperatingSystemProfile) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func (ad *admissionData) validateOperatingSystemProfileUpdate(ospOld osmv1alpha1.OperatingSystemProfile, ospNew osmv1alpha1.OperatingSystemProfile) field.ErrorList {
	allErrs := field.ErrorList{}

	if equal := apiequality.Semantic.DeepEqual(ospOld.Spec, ospNew.Spec); equal {
		// There is no change in spec so no validation is required
		return allErrs
	}

	// OSP is immutable by nature and to make modifications a version bump is mandatory
	if ospOld.Spec.Version == ospNew.Spec.Version {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), ospNew.Spec.Version, "OperatingSystemProfile is immutable. For updates .spec.version needs to be updated"))
		return allErrs
	}
	return allErrs
}
