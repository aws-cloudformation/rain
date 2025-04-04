package pkg

import (
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

const (
	Hash     string = "Hash"
	Location string = "Location"
)

// Process the Packages section with aliases to modules
func processPackages(t *cft.Template, n *yaml.Node) error {

	// Process top-level Packages section (for Modules section)
	packages := s11n.GetMap(n, string(cft.Packages))
	if packages != nil {
		if t.Packages == nil {
			t.Packages = make(map[string]*cft.PackageAlias)
		}

		for k, v := range packages {
			p := &cft.PackageAlias{}
			p.Alias = k
			p.Location = s11n.GetValue(v, Source) // Use Source instead of Location for top-level
			p.Hash = s11n.GetValue(v, Hash)
			t.Packages[k] = p
		}

		// Process modules section to replace $alias/path with actual paths
		_, moduleSection, _ := s11n.GetMapValue(n, string(cft.Modules))
		if moduleSection != nil && moduleSection.Kind == yaml.MappingNode {
			for i := 0; i < len(moduleSection.Content); i += 2 {
				module := moduleSection.Content[i+1]
				_, sourceNode, _ := s11n.GetMapValue(module, Source)

				if sourceNode != nil && sourceNode.Kind == yaml.ScalarNode {
					source := sourceNode.Value
					if strings.HasPrefix(source, "$") {
						parts := strings.SplitN(source, "/", 2)
						if len(parts) == 2 {
							alias := parts[0][1:] // Remove the $ prefix
							path := parts[1]

							if packageAlias, ok := t.Packages[alias]; ok {
								// Replace the alias with the actual location
								sourceNode.Value = packageAlias.Location + "/" + path
							}
						}
					}
				}
			}
		}

		node.RemoveFromMap(n, string(cft.Packages))
	}

	// Process Rain.Packages section (for backward compatibility)
	rainSection, err := t.GetSection(cft.Rain)
	if err == nil {
		rainPackages := s11n.GetMap(rainSection, string(cft.Packages))
		if rainPackages != nil {
			if t.Packages == nil {
				t.Packages = make(map[string]*cft.PackageAlias)
			}

			for k, v := range rainPackages {
				p := &cft.PackageAlias{}
				p.Alias = k
				p.Location = s11n.GetValue(v, Location)
				p.Hash = s11n.GetValue(v, Hash)
				t.Packages[k] = p
			}
		}
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
