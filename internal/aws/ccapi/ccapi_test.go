package ccapi

import (
	"encoding/json"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

func TestPatch(t *testing.T) {
	p := patch{Op: "replace", Path: "/A", Value: "1"}
	patches := make([]patch, 0)
	patches = append(patches, p)
	m, err := json.Marshal(patches)
	if err != nil {
		t.Fatal(err)
	}
	pstr := string(m)
	expected := "[{\"op\":\"replace\",\"path\":\"/A\",\"value\":\"1\"}]"
	if pstr != expected {
		t.Fatalf("expected:\n%v\ngot:\n%v\npatches:%v", expected, pstr, patches)
	}
}

func TestPatchPath(t *testing.T) {

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
	_, resources := s11n.GetMapValue(rootMap, "Resources")
	_, myResource := s11n.GetMapValue(resources, "MyResource")
	_, props := s11n.GetMapValue(myResource, "Properties")

	// Create the patch string
	patchDocument, err := createPatch(props)
	if err != nil {
		t.Fatal(err)
	}

	// Construct what we expect
	expect := make([]patch, 0)
	expect = append(expect, patch{Op: "replace", Path: "/A", Value: 5})
	expect = append(expect, patch{Op: "replace", Path: "/B/C", Value: true})
	expect = append(expect, patch{Op: "replace", Path: "/B/D", Value: "Hello"})
	expect = append(expect, patch{Op: "replace", Path: "/B/E/F", Value: false})

	m, err := json.Marshal(expect)
	if err != nil {
		t.Fatal(err)
	}
	pstr := string(m)

	// Make sure they match
	if pstr != patchDocument {
		t.Fatalf("Got:\n%v\nexpected:\n%v", patchDocument, pstr)
	}

	//config.Debug = true
	// config.Debugf(format.PrettyPrint(expect))

	/*
		[
			{
				"op": "replace",
				"path": "/A",
				"value": 5
			},
			{
				"op": "replace",
				"path": "/B/C",
				"value": true
			},
			{
				"op": "replace",
				"path": "/B/D",
				"value": "Hello"
			},
			{
				"op": "replace",
				"path": "/B/E/F",
				"value": false
			}
		]
	*/

}
