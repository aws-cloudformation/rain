package format_test

import (
	"io/ioutil"
	"testing"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

var input string

func init() {
	b, err := ioutil.ReadFile("test/input.yaml")
	if err != nil {
		panic(err)
	}

	input = string(b)
}

func checkMatch(t *testing.T, expectedFn string, opt format.Options) {
	b, err := ioutil.ReadFile("test/" + expectedFn)
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)

	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, opt)

	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf(d)
	}
}

func TestFormatDefault(t *testing.T) {
	//checkMatch(t, expectedYaml, format.Options{})
	//checkMatch(t, expectedYamlUnsorted, format.Options{
	//	Unsorted: true,
	//})
	checkMatch(t, "sorted.json", format.Options{
		JSON: true,
	})
	//checkMatch(t, expectedUnsortedJson, format.Options{
	//	JSON:     true,
	//	Unsorted: true,
	//})
}
