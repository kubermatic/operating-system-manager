package generator

import (
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	funcMap["runCMDs"] = runCMDs

	return funcMap
}

func runCMDs(fSpecs []*fileSpec) []string {
	var services []string
	for _, spec := range fSpecs {
		if service := getServiceName(spec.Path); service != "" {
			services = append(services, service)
		}
	}

	return services
}
