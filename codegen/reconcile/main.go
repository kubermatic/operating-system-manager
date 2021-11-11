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

// The following directive is necessary to make the package coherent:

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func main() {
	data := struct {
		Resources []reconcileFunctionData
	}{
		Resources: []reconcileFunctionData{
			{
				ResourceName:       "Secret",
				ImportAlias:        "corev1",
				ResourceImportPath: "k8s.io/api/core/v1",
			},
			{
				ResourceName:       "OperatingSystemConfig",
				ImportAlias:        "osmv1alpha1",
				ResourceImportPath: "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1",
			},
			{
				ResourceName:       "ClusterRoleBinding",
				ImportAlias:        "rbacv1",
				ResourceImportPath: "k8s.io/api/rbac/v1",
			},
			{
				ResourceName:       "ClusterRole",
				ImportAlias:        "rbacv1",
				ResourceImportPath: "k8s.io/api/rbac/v1",
			},
		},
	}

	buf := &bytes.Buffer{}
	if err := reconcileAllTemplate.Execute(buf, data); err != nil {
		log.Fatal(err)
	}

	fmtB, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("zz_generated_reconcile.go", fmtB, 0644); err != nil {
		log.Fatal(err)
	}
}

func lowercaseFirst(str string) string {
	return strings.ToLower(string(str[0])) + str[1:]
}

var (
	reconcileAllTplFuncs = map[string]interface{}{
		"namedReconcileFunc": namedReconcileFunc,
	}
	reconcileAllTemplate = template.Must(template.New("").Funcs(reconcileAllTplFuncs).Funcs(sprig.TxtFuncMap()).Parse(`// This file is generated. DO NOT EDIT.
package reconciling

import (
	"fmt"
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
{{ range .Resources }}
{{- if .ResourceImportPath }}
	{{ .ImportAlias }} "{{ .ResourceImportPath }}"
{{- end }}
{{- end }}
)

{{ range .Resources }}
{{ namedReconcileFunc .ResourceName .ImportAlias .DefaultingFunc .RequiresRecreate .ResourceNamePlural .APIVersionPrefix}}
{{- end }}

`))
)

type reconcileFunctionData struct {
	ResourceName       string
	ResourceNamePlural string
	ResourceImportPath string
	ImportAlias        string
	// Optional: A defaulting func for the given object type
	// Must be defined inside the resources package
	DefaultingFunc string
	// Whether the resource must be recreated instead of updated. Required
	// e.G. for PDBs
	RequiresRecreate bool
	// Optional: adds an api version prefix to the generated functions to avoid duplication when different resources
	// have the same ResourceName
	APIVersionPrefix string
}

func namedReconcileFunc(resourceName, importAlias, defaultingFunc string, requiresRecreate bool, plural, apiVersionPrefix string) (string, error) {
	if len(plural) == 0 {
		plural = fmt.Sprintf("%ss", resourceName)
	}

	b := &bytes.Buffer{}
	err := namedReconcileFunctionTemplate.Execute(b, struct {
		ResourceName       string
		ResourceNamePlural string
		ImportAlias        string
		DefaultingFunc     string
		RequiresRecreate   bool
		APIVersionPrefix   string
	}{
		ResourceName:       resourceName,
		ResourceNamePlural: plural,
		ImportAlias:        importAlias,
		DefaultingFunc:     defaultingFunc,
		RequiresRecreate:   requiresRecreate,
		APIVersionPrefix:   apiVersionPrefix,
	})

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

var (
	reconcileFunctionTplFuncs = map[string]interface{}{
		"lowercaseFirst": lowercaseFirst,
	}
)

var namedReconcileFunctionTemplate = template.Must(template.New("").Funcs(reconcileFunctionTplFuncs).Parse(`// {{ .APIVersionPrefix }}{{ .ResourceName }}Creator defines an interface to create/update {{ .ResourceName }}s
type {{ .APIVersionPrefix }}{{ .ResourceName }}Creator = func(existing *{{ .ImportAlias }}.{{ .ResourceName }}) (*{{ .ImportAlias }}.{{ .ResourceName }}, error)

// Named{{ .APIVersionPrefix }}{{ .ResourceName }}CreatorGetter returns the name of the resource and the corresponding creator function
type Named{{ .APIVersionPrefix }}{{ .ResourceName }}CreatorGetter = func() (name string, create {{ .APIVersionPrefix }}{{ .ResourceName }}Creator)

// {{ .APIVersionPrefix }}{{ .ResourceName }}ObjectWrapper adds a wrapper so the {{ .APIVersionPrefix }}{{ .ResourceName }}Creator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func {{ .APIVersionPrefix }}{{ .ResourceName }}ObjectWrapper(create {{ .APIVersionPrefix }}{{ .ResourceName }}Creator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*{{ .ImportAlias }}.{{ .ResourceName }}))
		}
		return create(&{{ .ImportAlias }}.{{ .ResourceName }}{})
	}
}

// Reconcile{{ .APIVersionPrefix }}{{ .ResourceNamePlural }} will create and update the {{ .APIVersionPrefix }}{{ .ResourceNamePlural }} coming from the passed {{ .APIVersionPrefix }}{{ .ResourceName }}Creator slice
func Reconcile{{ .APIVersionPrefix }}{{ .ResourceNamePlural }}(ctx context.Context, namedGetters []Named{{ .APIVersionPrefix }}{{ .ResourceName }}CreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
{{- if .DefaultingFunc }}
		create = {{ .DefaultingFunc }}(create)
{{- end }}
		createObject := {{ .APIVersionPrefix }}{{ .ResourceName }}ObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &{{ .ImportAlias }}.{{ .ResourceName }}{}, {{ .RequiresRecreate}}); err != nil {
			return fmt.Errorf("failed to ensure {{ .ResourceName }} %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

`))
