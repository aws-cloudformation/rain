package ccdeploy

import (
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
)

type token rune

const (
	DATA    token = ' ' // Any other rune
	DOLLAR        = '$'
	OPEN          = '{'
	CLOSE         = '}'
	LITERAL       = '!'
)

type wordtype int

const (
	STR    wordtype = iota // A literal string fragment
	REF                    // ${ParamOrResourceName}
	AWS                    // ${AWS::X}
	GETATT                 // ${X.Y}
)

type word struct {
	t wordtype
	w string // Does not include the ${} if it's not a STR
}

type state int

const (
	READSTR state = iota
	MAYBE
	READVAR
)

// ParseSub returns a slice of words
//
// "ABC-${XYZ}-123"
//
// returns a slice containing:
//
//	word { t: STR, w: "ABC-" }
//	word { t: REF, w: "XYZ" }
//	word { t: STR, w: "-123" }
//
// Invalid syntax like "${AAA" returns an error
func ParseSub(sub string) ([]word, error) {
	words := make([]word, 0)
	state := READSTR
	var last rune
	var buf string
	var wt wordtype
	for _, r := range sub {
		config.Debugf("%#U", r)
		switch r {
		case DOLLAR:
			if state != READVAR {
				state = MAYBE
			} else {
				buf += string(r)
			}
		case OPEN:
			if state == MAYBE {
				state = READVAR
				// We're about to start reading a variable.
				// Append the last word in the buffer if it's not empty
				if len(buf) > 0 {
					wt = STR
					words = append(words, word{t: wt, w: buf})
					buf = ""
				}
			} else {
				buf += string(r)
			}
		case CLOSE:
			if state == READVAR {
				// Figure out what type it is
				if strings.HasPrefix(buf, "AWS::") {
					wt = AWS
				} else if strings.Contains(buf, ".") {
					wt = GETATT
				} else {
					wt = REF
				}
				words = append(words, word{t: wt, w: buf})
				buf = ""
				state = READSTR
			} else {
				buf += string(r)
			}
		case LITERAL:
			// ${!LITERAL} becomes ${LITERAL}
			if state == READVAR {
				buf = "${"
				state = READSTR
			} else {
				buf += string(r)
			}
		default:
			if state == MAYBE {
				buf += string(last) // Put the $ back on the buffer
				state = READSTR
			}
			buf += string(r)
		}
		last = r
	}
	if len(buf) > 0 {
		wt = STR
		words = append(words, word{t: wt, w: buf})
		buf = ""
	}

	return words, nil
}
