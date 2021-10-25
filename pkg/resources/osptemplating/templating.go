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

package osptemplating

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

func templateOSP(files []osmv1alpha1.File, templatefiles ...string) ([]osmv1alpha1.File, error) {
	var pfiles []osmv1alpha1.File

	for _, file := range files {
		content := file.Content.Inline.Data
		tmpl, err := template.New(file.Path).Funcs(template.FuncMap(sprig.FuncMap())).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OSP file [%s] template: %v", file.Path, err)
		}
		for _, tf := range templatefiles {
			tmpl = template.Must(tmpl.Funcs(template.FuncMap(sprig.FuncMap())).Parse(tf))
		}

		buff := bytes.Buffer{}
		if err := tmpl.Execute(&buff, nil); err != nil {
			return nil, err
		}

		file.Content.Inline.Data = buff.String()
		pfile := file.DeepCopy()
		pfile.Content.Inline.Data = buff.String()
		pfiles = append(pfiles, *pfile)
	}

	return pfiles, nil
}
