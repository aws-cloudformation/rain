package ccapi

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

func TestPatch(t *testing.T) {

	// Create a PatchDocument based on the resource in this template
	s := `
Resources:
  MyResource:
    Type: X::Y::Z
    Properties:
      A: 5
      B:
        C: true
        D: Hello
        E:
          F: false
`
	template, err := parse.String(s)
	if err != nil {
		t.Fatal(err)
	}
	// Dive down to the yaml resource properties
	rootMap := template.Node.Content[0]
	_, resources, _ := s11n.GetMapValue(rootMap, "Resources")
	_, myResource, _ := s11n.GetMapValue(resources, "MyResource")
	_, props, _ := s11n.GetMapValue(myResource, "Properties")

	// Create the patch string
	patchDocument, err := CreatePatch(props, "{}")
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
    {"op":"add","path":"/A","value":5},
    {"op":"add","path":"/B","value":{"C":true,"D":"Hello","E":{"F":false}}}
]`

	// Make sure they match
	if expected != patchDocument {
		t.Fatalf("Got:\n%v\nexpected:\n%v", patchDocument, expected)
	}

	patchDocument, err = CreatePatch(props, `
{
	"A": 5,
	"B": {
		"C": false,
		"D": "World"
	}
}
`)
	if err != nil {
		t.Fatal(err)
	}

	expected = `[
    {"op":"replace","path":"/B/C","value":true},
    {"op":"replace","path":"/B/D","value":"Hello"},
    {"op":"add","path":"/B/E","value":{"F":false}}
]`

	// Make sure they match
	if expected != patchDocument {
		t.Fatalf("Got:\n%v\nexpected:\n%v", patchDocument, expected)
	}

}
