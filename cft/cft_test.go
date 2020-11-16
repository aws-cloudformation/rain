package cft

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestResolve(t *testing.T) {
	orig := map[string]interface{}{
		"TestRef": map[string]interface{}{
			"Ref": "Param1",
		},
		"TestSub1": map[string]interface{}{
			"Fn::Sub": "A wild ${Param1} appears",
		},
		"TestSub2": map[string]interface{}{
			"Fn::Sub": []interface{}{
				"A ${adj} ${Param1} ${verb}",
				map[string]interface{}{
					"adj":  "tame",
					"verb": "disappears",
				},
			},
		},
	}

	expected := map[string]interface{}{
		"TestRef":  "Charizard",
		"TestSub1": "A wild Charizard appears",
		"TestSub2": "A tame Charizard disappears",
	}

	var origNode yaml.Node
	origNode.Encode(orig)

	resolve(&origNode, map[string]string{"Param1": "Charizard"})

	var actual map[string]interface{}
	origNode.Decode(&actual)

	if d := cmp.Diff(expected, actual); d != "" {
		t.Error(d)
	}
}
