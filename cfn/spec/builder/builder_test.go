package builder

import (
	"testing"
)

var builder = NewCfnBuilder(true, true)
var allResourceTypes map[string]string

func init() {
	allResourceTypes = make(map[string]string)

	for resourceType, _ := range builder.Spec.ResourceTypes {
		allResourceTypes[resourceType] = resourceType
	}
}

func TestAllResourceTypes(t *testing.T) {
	for resourceType, _ := range builder.Spec.ResourceTypes {
		builder.Template(map[string]string{
			"Res": resourceType,
		})
	}
}

func BenchmarkAllResourceTypesIndividually(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for resourceType, _ := range allResourceTypes {
			builder.Template(map[string]string{
				"Res": resourceType,
			})
		}
	}
}

func BenchmarkAllResourceTypesInOne(b *testing.B) {
	for n := 0; n < b.N; n++ {
		builder.Template(allResourceTypes)
	}
}
