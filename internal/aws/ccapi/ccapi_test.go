package ccapi

import (
	"encoding/json"
	"testing"
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
