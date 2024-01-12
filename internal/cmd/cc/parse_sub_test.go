package cc

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/config"
)

func TestParseSub(t *testing.T) {
	config.Debug = true

	cases := make(map[string][]word, 0)

	// Just a string
	cases["ABC"] = []word{word{t: STR, w: "ABC"}}

	// Basic use case with a Ref
	cases["ABC-${XYZ}-123"] = []word{
		word{t: STR, w: "ABC-"},
		word{t: REF, w: "XYZ"},
		word{t: STR, w: "-123"},
	}

	// Literal
	cases["ABC-${!Literal}-123"] = []word{
		word{t: STR, w: "ABC-${Literal}-123"},
	}

	// Variable by itself
	cases["${ABC}"] = []word{word{t: REF, w: "ABC"}}

	// GetAtt
	cases["${ABC.XYZ}"] = []word{word{t: GETATT, w: "ABC.XYZ"}}

	// AWS
	cases["ABC${AWS::AccountId}XYZ"] = []word{
		word{t: STR, w: "ABC"},
		word{t: AWS, w: "AccountId"},
		word{t: STR, w: "XYZ"},
	}

	// DOLLARS everywhere
	cases["BAZ${ABC$XYZ}FOO$BAR"] = []word{
		word{t: STR, w: "BAZ"},
		word{t: REF, w: "ABC$XYZ"},
		word{t: STR, w: "FOO$BAR"},
	}

	for sub, expect := range cases {
		words, err := ParseSub(sub)
		if err != nil {
			t.Fatal(err)
		}
		config.Debugf("%v", words)
		if len(expect) != len(words) {
			t.Fatalf("\"%s\": words len is %v, expected %v", sub, len(words), len(expect))
		}
		for i, w := range expect {
			if words[i].t != w.t || words[i].w != w.w {
				t.Fatalf("\"%s\": got %v, expected %v", sub, words[i], w)
			}
		}
	}

	// Invalid strings should fail
	sub := "${AAA"
	words, err := ParseSub(sub)
	if err == nil {
		t.Fatalf("\"%s\": should have failed, got %v", sub, words)
	}
}
