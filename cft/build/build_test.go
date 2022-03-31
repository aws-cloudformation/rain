package build_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/build"
	"github.com/aws-cloudformation/rain/cft/spec"
)

var allResourceTypes map[string]string

func init() {
	allResourceTypes = make(map[string]string)

	for resourceType := range spec.Cfn.ResourceTypes {
		allResourceTypes[resourceType] = resourceType
	}
}

func TestAllResourceTypes(t *testing.T) {
	for resourceType := range spec.Cfn.ResourceTypes {
		_, err := build.Template(map[string]string{
			"Res": resourceType,
		}, true)

		if err != nil {
			t.Error(fmt.Errorf("%s: %w", resourceType, err))
		}
	}
}

func BenchmarkAllResourceTypesIndividually(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for resourceType := range allResourceTypes {
			build.Template(map[string]string{
				"Res": resourceType,
			}, true)
		}
	}
}

func BenchmarkAllResourceTypesInOne(b *testing.B) {
	for n := 0; n < b.N; n++ {
		build.Template(allResourceTypes, true)
	}
}
