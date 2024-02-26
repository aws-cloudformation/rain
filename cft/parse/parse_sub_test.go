package parse_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
)

func TestParseSub(t *testing.T) {
	config.Debug = true

	cases := make(map[string][]parse.SubWord, 0)

	// Just a string
	cases["ABC"] = []parse.SubWord{parse.SubWord{T: parse.STR, W: "ABC"}}

	// Basic use case with a Ref
	cases["ABC-${XYZ}-123"] = []parse.SubWord{
		parse.SubWord{T: parse.STR, W: "ABC-"},
		parse.SubWord{T: parse.REF, W: "XYZ"},
		parse.SubWord{T: parse.STR, W: "-123"},
	}

	// Literal
	cases["ABC-${!Literal}-123"] = []parse.SubWord{
		parse.SubWord{T: parse.STR, W: "ABC-${Literal}-123"},
	}

	// Variable by itself
	cases["${ABC}"] = []parse.SubWord{parse.SubWord{T: parse.REF, W: "ABC"}}

	// GetAtt
	cases["${ABC.XYZ}"] = []parse.SubWord{parse.SubWord{T: parse.GETATT, W: "ABC.XYZ"}}

	// AWS
	cases["ABC${AWS::AccountId}XYZ"] = []parse.SubWord{
		parse.SubWord{T: parse.STR, W: "ABC"},
		parse.SubWord{T: parse.AWS, W: "AccountId"},
		parse.SubWord{T: parse.STR, W: "XYZ"},
	}

	// DOLLARS everywhere
	cases["BAZ${ABC$XYZ}FOO$BAR"] = []parse.SubWord{
		parse.SubWord{T: parse.STR, W: "BAZ"},
		parse.SubWord{T: parse.REF, W: "ABC$XYZ"},
		parse.SubWord{T: parse.STR, W: "FOO$BAR"},
	}

	for sub, expect := range cases {
		words, err := parse.ParseSub(sub)
		if err != nil {
			t.Fatal(err)
		}
		config.Debugf("%v", words)
		if len(expect) != len(words) {
			t.Fatalf("\"%s\": words len is %v, expected %v", sub, len(words), len(expect))
		}
		for i, w := range expect {
			if words[i].T != w.T || words[i].W != w.W {
				t.Fatalf("\"%s\": got %v, expected %v", sub, words[i], w)
			}
		}
	}

	// Invalid strings should fail
	sub := "${AAA"
	words, err := parse.ParseSub(sub)
	if err == nil {
		t.Fatalf("\"%s\": should have failed, got %v", sub, words)
	}
}
