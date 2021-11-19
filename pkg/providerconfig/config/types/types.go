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

package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// GlobalObjectKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace
type GlobalObjectKeySelector struct {
	corev1.ObjectReference `json:",inline"`
	Key                    string `json:"key,omitempty"`
}

type GlobalSecretKeySelector GlobalObjectKeySelector
type GlobalConfigMapKeySelector GlobalObjectKeySelector

type ConfigVarString struct {
	Value           string                     `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

// This type only exists to have the same fields as ConfigVarString but
// not its funcs, so it can be used as target for json.Unmarshal without
// causing a recursion
type configVarStringWithoutUnmarshaller ConfigVarString

// MarshalJSON converts a configVarString to its JSON form, omitting empty strings.
// This is done to not have the json object cluttered with empty strings
// This will eventually hopefully be resolved within golang itself
// https://github.com/golang/go/issues/11939
func (configVarString ConfigVarString) MarshalJSON() ([]byte, error) {
	var secretKeyRefEmpty, configMapKeyRefEmpty bool
	if configVarString.SecretKeyRef.ObjectReference.Namespace == "" &&
		configVarString.SecretKeyRef.ObjectReference.Name == "" &&
		configVarString.SecretKeyRef.Key == "" {
		secretKeyRefEmpty = true
	}

	if configVarString.ConfigMapKeyRef.ObjectReference.Namespace == "" &&
		configVarString.ConfigMapKeyRef.ObjectReference.Name == "" &&
		configVarString.ConfigMapKeyRef.Key == "" {
		configMapKeyRefEmpty = true
	}

	if secretKeyRefEmpty && configMapKeyRefEmpty {
		return []byte(fmt.Sprintf(`"%s"`, configVarString.Value)), nil
	}

	buffer := bytes.NewBufferString("{")
	if !secretKeyRefEmpty {
		jsonVal, err := json.Marshal(configVarString.SecretKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`"secretKeyRef":%s`, string(jsonVal)))
	}

	if !configMapKeyRefEmpty {
		var leadingComma string
		if !secretKeyRefEmpty {
			leadingComma = ","
		}
		jsonVal, err := json.Marshal(configVarString.ConfigMapKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`%s"configMapKeyRef":%s`, leadingComma, jsonVal))
	}

	if configVarString.Value != "" {
		buffer.WriteString(fmt.Sprintf(`,"value":"%s"`, configVarString.Value))
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (configVarString *ConfigVarString) UnmarshalJSON(b []byte) error {
	if !bytes.HasPrefix(b, []byte("{")) {
		b = bytes.TrimPrefix(b, []byte(`"`))
		b = bytes.TrimSuffix(b, []byte(`"`))
		configVarString.Value = string(b)
		return nil
	}
	// This type must have the same fields as ConfigVarString but not
	// its UnmarshalJSON, otherwise we cause a recursion
	var cvsDummy configVarStringWithoutUnmarshaller
	err := json.Unmarshal(b, &cvsDummy)
	if err != nil {
		return err
	}
	configVarString.Value = cvsDummy.Value
	configVarString.SecretKeyRef = cvsDummy.SecretKeyRef
	configVarString.ConfigMapKeyRef = cvsDummy.ConfigMapKeyRef
	return nil
}

type ConfigVarBool struct {
	Value           bool                       `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

type configVarBoolWithoutUnmarshaller ConfigVarBool

// MarshalJSON encodes the configVarBool, omitting empty strings
// This is done to not have the json object cluttered with empty strings
// This will eventually hopefully be resolved within golang itself
// https://github.com/golang/go/issues/11939
func (configVarBool ConfigVarBool) MarshalJSON() ([]byte, error) {
	var secretKeyRefEmpty, configMapKeyRefEmpty bool
	if configVarBool.SecretKeyRef.ObjectReference.Namespace == "" &&
		configVarBool.SecretKeyRef.ObjectReference.Name == "" &&
		configVarBool.SecretKeyRef.Key == "" {
		secretKeyRefEmpty = true
	}

	if configVarBool.ConfigMapKeyRef.ObjectReference.Namespace == "" &&
		configVarBool.ConfigMapKeyRef.ObjectReference.Name == "" &&
		configVarBool.ConfigMapKeyRef.Key == "" {
		configMapKeyRefEmpty = true
	}

	if secretKeyRefEmpty && configMapKeyRefEmpty {
		return []byte(fmt.Sprintf("%v", configVarBool.Value)), nil
	}

	buffer := bytes.NewBufferString("{")
	if !secretKeyRefEmpty {
		jsonVal, err := json.Marshal(configVarBool.SecretKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`"secretKeyRef":%s`, string(jsonVal)))
	}

	if !configMapKeyRefEmpty {
		var leadingComma string
		if !secretKeyRefEmpty {
			leadingComma = ","
		}
		jsonVal, err := json.Marshal(configVarBool.ConfigMapKeyRef)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf(`%s"configMapKeyRef":%s`, leadingComma, jsonVal))
	}

	buffer.WriteString(fmt.Sprintf(`,"value":%v}`, configVarBool.Value))

	return buffer.Bytes(), nil
}

func (configVarBool *ConfigVarBool) UnmarshalJSON(b []byte) error {
	if !bytes.HasPrefix(b, []byte("{")) {
		value, err := strconv.ParseBool(string(b))
		if err != nil {
			return fmt.Errorf("Error converting string to bool: '%v'", err)
		}
		configVarBool.Value = value
		return nil
	}
	var cvbDummy configVarBoolWithoutUnmarshaller
	err := json.Unmarshal(b, &cvbDummy)
	if err != nil {
		return err
	}
	configVarBool.Value = cvbDummy.Value
	configVarBool.SecretKeyRef = cvbDummy.SecretKeyRef
	configVarBool.ConfigMapKeyRef = cvbDummy.ConfigMapKeyRef
	return nil
}
