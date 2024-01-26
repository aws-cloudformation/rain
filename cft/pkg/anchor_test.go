package pkg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAnchors(t *testing.T) {
	source := "./tmpl/anchors.yaml"
	template, err := File(source)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := File("./tmpl/anchors-expect.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// This actually succeeds, because the diff package uses YAML decoding
	// to make the comparison, and the yaml package understands anchors
	//d := diff.New(template, expected)
	//if d.Mode() != "=" {
	//	t.Errorf("template does not match expected: %v", d.Format(true))
	//}

	// This correctly shows differences in the templates
	if d := cmp.Diff(expected, template); d != "" {
		t.Error(d)
	}

}
