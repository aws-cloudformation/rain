package parse_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

func TestPreserveComments(t *testing.T) {
	input := `
foo:
  Ref: foo-ref # foo-ref comment
bar: !Ref bar-ref # bar-ref comment
baz:
  Fn::GetAtt: baz.att # baz-getatt comment
quux: !GetAtt quux.att # quux-getatt comment
mooz:
  Fn::GetAtt: # mooz
    - mooz # getatt
    - att # comment
`

	expected := `foo: !Ref foo-ref # foo-ref comment

bar: !Ref bar-ref # bar-ref comment

baz: !GetAtt baz.att # baz-getatt comment

quux: !GetAtt quux.att # quux-getatt comment

mooz: !GetAtt mooz.att # mooz getatt comment
`

	tmpl, err := parse.String(input)
	if err != nil {
		t.Error(err)
	}

	actual := format.String(tmpl, format.Options{})

	if d := cmp.Diff(expected, actual); d != "" {
		t.Error(d)
	}
}
