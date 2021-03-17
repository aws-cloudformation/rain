package format_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

const data = "\n\nfoo\nbar\n\nbaz\n\n\nquux\n\n\n\n"

func TestYaml(t *testing.T) {
	template, err := parse.Map(map[string]interface{}{
		"test": data,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Format as YAML
	formatted := format.String(template, format.Options{})
	fmt.Printf("%q\n", formatted)

	// Parse back to a map
	var parsed map[string]interface{}
	err = yaml.Unmarshal([]byte(formatted), &parsed)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(data, parsed["test"]); d != "" {
		t.Errorf(d)
	}
}
