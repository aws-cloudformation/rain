package pkg

import (
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Process the Packages section with aliases to modules
func processPackages(t *cft.Template, n *yaml.Node) error {
	packages := s11n.GetMap(n, "Packages")
	if packages == nil {
		return nil
	}
	t.Packages = make(map[string]*cft.PackageAlias)
	for k, v := range packages {
		p := &cft.PackageAlias{}
		p.Alias = k
		p.Location = s11n.GetValue(v, "Location")
		p.Hash = s11n.GetValue(v, "Hash")
		t.Packages[k] = p
	}

	// Visit all resources to look for Type nodes that use $alias.module shorthand
	resources, err := t.GetSection(cft.Resources)
	if err == nil {
		for i := 0; i < len(resources.Content); i += 2 {
			resource := resources.Content[i+1]
			_, typ, _ := s11n.GetMapValue(resource, "Type")
			if typ == nil {
				continue
			}
			if strings.HasPrefix(typ.Value, "$") {
				// This is a package alias, fix it so it gets processed later
				//typ.Value = "!Rain::Module " + strings.Trim(typ.Value, "$")
				newTypeNode := yaml.Node{Kind: yaml.MappingNode}
				newTypeNode.Content = []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "Rain::Module"},
					{Kind: yaml.ScalarNode, Value: strings.Trim(typ.Value, "$")},
				}
				*typ = newTypeNode
			}
		}
	}
	return nil
}
