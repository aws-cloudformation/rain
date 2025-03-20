package pkg

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"gopkg.in/yaml.v3"
)

func TestConstants(t *testing.T) {

	p, err := parse.File("./tmpl/constant2-template.yaml")
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := Template(p, ".", nil)
	if err != nil {
		t.Fatal(err)
	}

	et, err := parse.File("./tmpl/constant2-expect.yaml")
	if err != nil {
		t.Fatal(err)
	}

	d := diff.New(tmpl, et)
	if d.Mode() != "=" {
		t.Errorf("Output does not match expected: %v", d.Format(true))
	}

}

func TestReplaceConstants(t *testing.T) {
	n := &yaml.Node{Kind: yaml.ScalarNode, Value: "${Rain::Test}"}
	constants := make(map[string]*yaml.Node)
	constants["Test"] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Foo"}
	err := replaceConstants(n, constants)
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != "Foo" {
		t.Fatalf("Expected Foo, got %s", n.Value)
	}
}

func TestIsSubNeeded(t *testing.T) {
	cases := make(map[string]bool)
	cases["ABC"] = false
	cases["${A}bc"] = true
	cases["${Rain::Something}"] = true
	cases[""] = false
	cases["${Abc.Def"] = true
	cases["${!saml:sub}"] = false
	cases["${!Literal}-abc"] = false
	cases["$foo$bar"] = false

	for k, v := range cases {
		if parse.IsSubNeeded(k) != v {
			t.Errorf("IsSubNeeded(%s) should be %v", k, v)
		}
	}
}
