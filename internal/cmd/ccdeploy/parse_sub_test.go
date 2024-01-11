package ccdeploy

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/config"
)

func TestParseSub(t *testing.T) {
	config.Debug = true
	sub := "ABC-${XYZ}-123"
	words, err := ParseSub(sub)
	if err != nil {
		t.Fatal(err)
	}
	config.Debugf("%v", words)
	expect := []word{
		word{t: STR, w: "ABC-"},
		word{t: REF, w: "XYZ"},
		word{t: STR, w: "-123"},
	}
	if len(expect) != len(words) {
		t.Fatalf("words len is %v, expected %v", len(words), len(expect))
	}
	for i, w := range expect {
		if words[i].t != w.t || words[i].w != w.w {
			t.Fatalf("Got %v, expected %v", words[i], w)
		}
	}
}
