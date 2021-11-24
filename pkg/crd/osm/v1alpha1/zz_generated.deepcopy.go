//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudInitModule) DeepCopyInto(out *CloudInitModule) {
	*out = *in
	if in.BootCMD != nil {
		in, out := &in.BootCMD, &out.BootCMD
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RHSubscription != nil {
		in, out := &in.RHSubscription, &out.RHSubscription
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.RunCMD != nil {
		in, out := &in.RunCMD, &out.RunCMD
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudInitModule.
func (in *CloudInitModule) DeepCopy() *CloudInitModule {
	if in == nil {
		return nil
	}
	out := new(CloudInitModule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudProviderSpec) DeepCopyInto(out *CloudProviderSpec) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudProviderSpec.
func (in *CloudProviderSpec) DeepCopy() *CloudProviderSpec {
	if in == nil {
		return nil
	}
	out := new(CloudProviderSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerRuntimeSpec) DeepCopyInto(out *ContainerRuntimeSpec) {
	*out = *in
	if in.Files != nil {
		in, out := &in.Files, &out.Files
		*out = make([]File, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerRuntimeSpec.
func (in *ContainerRuntimeSpec) DeepCopy() *ContainerRuntimeSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerRuntimeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DropIn) DeepCopyInto(out *DropIn) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DropIn.
func (in *DropIn) DeepCopy() *DropIn {
	if in == nil {
		return nil
	}
	out := new(DropIn)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *File) DeepCopyInto(out *File) {
	*out = *in
	if in.Permissions != nil {
		in, out := &in.Permissions, &out.Permissions
		*out = new(int32)
		**out = **in
	}
	in.Content.DeepCopyInto(&out.Content)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new File.
func (in *File) DeepCopy() *File {
	if in == nil {
		return nil
	}
	out := new(File)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FileContent) DeepCopyInto(out *FileContent) {
	*out = *in
	if in.Inline != nil {
		in, out := &in.Inline, &out.Inline
		*out = new(FileContentInline)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FileContent.
func (in *FileContent) DeepCopy() *FileContent {
	if in == nil {
		return nil
	}
	out := new(FileContent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FileContentInline) DeepCopyInto(out *FileContentInline) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FileContentInline.
func (in *FileContentInline) DeepCopy() *FileContentInline {
	if in == nil {
		return nil
	}
	out := new(FileContentInline)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemConfig) DeepCopyInto(out *OperatingSystemConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemConfig.
func (in *OperatingSystemConfig) DeepCopy() *OperatingSystemConfig {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatingSystemConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemConfigList) DeepCopyInto(out *OperatingSystemConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OperatingSystemConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemConfigList.
func (in *OperatingSystemConfigList) DeepCopy() *OperatingSystemConfigList {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatingSystemConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemConfigSpec) DeepCopyInto(out *OperatingSystemConfigSpec) {
	*out = *in
	in.CloudProvider.DeepCopyInto(&out.CloudProvider)
	if in.Units != nil {
		in, out := &in.Units, &out.Units
		*out = make([]Unit, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Files != nil {
		in, out := &in.Files, &out.Files
		*out = make([]File, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.UserSSHKeys != nil {
		in, out := &in.UserSSHKeys, &out.UserSSHKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.CloudInitModules != nil {
		in, out := &in.CloudInitModules, &out.CloudInitModules
		*out = new(CloudInitModule)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemConfigSpec.
func (in *OperatingSystemConfigSpec) DeepCopy() *OperatingSystemConfigSpec {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemProfile) DeepCopyInto(out *OperatingSystemProfile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemProfile.
func (in *OperatingSystemProfile) DeepCopy() *OperatingSystemProfile {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemProfile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatingSystemProfile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemProfileList) DeepCopyInto(out *OperatingSystemProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OperatingSystemProfile, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemProfileList.
func (in *OperatingSystemProfileList) DeepCopy() *OperatingSystemProfileList {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemProfileList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatingSystemProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatingSystemProfileSpec) DeepCopyInto(out *OperatingSystemProfileSpec) {
	*out = *in
	if in.SupportedCloudProviders != nil {
		in, out := &in.SupportedCloudProviders, &out.SupportedCloudProviders
		*out = make([]CloudProviderSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SupportedContainerRuntimes != nil {
		in, out := &in.SupportedContainerRuntimes, &out.SupportedContainerRuntimes
		*out = make([]ContainerRuntimeSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Units != nil {
		in, out := &in.Units, &out.Units
		*out = make([]Unit, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Files != nil {
		in, out := &in.Files, &out.Files
		*out = make([]File, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CloudInitModules != nil {
		in, out := &in.CloudInitModules, &out.CloudInitModules
		*out = new(CloudInitModule)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatingSystemProfileSpec.
func (in *OperatingSystemProfileSpec) DeepCopy() *OperatingSystemProfileSpec {
	if in == nil {
		return nil
	}
	out := new(OperatingSystemProfileSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Unit) DeepCopyInto(out *Unit) {
	*out = *in
	if in.Enable != nil {
		in, out := &in.Enable, &out.Enable
		*out = new(bool)
		**out = **in
	}
	if in.Mask != nil {
		in, out := &in.Mask, &out.Mask
		*out = new(bool)
		**out = **in
	}
	if in.Content != nil {
		in, out := &in.Content, &out.Content
		*out = new(string)
		**out = **in
	}
	if in.DropIns != nil {
		in, out := &in.DropIns, &out.DropIns
		*out = make([]DropIn, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Unit.
func (in *Unit) DeepCopy() *Unit {
	if in == nil {
		return nil
	}
	out := new(Unit)
	in.DeepCopyInto(out)
	return out
}
