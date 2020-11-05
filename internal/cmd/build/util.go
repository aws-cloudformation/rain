package build

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cft/spec"
)

func resolveType(suffix string) string {
	options := spec.Cfn.ResolveResource(suffix)

	if len(options) == 0 {
		fmt.Fprintf(os.Stderr, "No resource type found matching '%s'\n", suffix)
		os.Exit(1)
	} else if len(options) != 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous resource type '%s'; could be any of:\n", suffix)
		sort.Strings(options)
		for _, option := range options {
			fmt.Fprintf(os.Stderr, "  %s\n", option)
		}
		os.Exit(1)
	}

	return options[0]
}

func makeName(resourceType string) string {
	parts := strings.Split(resourceType, "::")
	return "My" + parts[len(parts)-1]
}

func resolveResources(resourceTypes []string) map[string]string {
	resources := make(map[string]string)

	for _, r := range resourceTypes {
		r = resolveType(r)
		name := makeName(r)

		if _, ok := resources[name]; ok {
			resources[name+"1"] = resources[name]
			delete(resources, name)

			name = name + "2"
		} else if _, ok := resources[name+"1"]; ok {
			for i := 2; true; i++ {
				if _, ok := resources[name+fmt.Sprint(i)]; !ok {
					name = name + fmt.Sprint(i)
					break
				}
			}
		}

		resources[name] = r
	}

	return resources
}
