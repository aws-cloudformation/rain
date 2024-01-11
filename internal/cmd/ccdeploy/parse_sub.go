package ccdeploy

import (
	"errors"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
)

const (
	DATA   rune = ' ' // Any other rune
	DOLLAR rune = '$'
	OPEN   rune = '{'
	CLOSE  rune = '}'
	BANG   rune = '!'
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
	READLIT
)

// ParseSub returns a slice of words, based on a string
// argument to the Fn::Sub intrinsic function.
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
	for i, r := range sub {
		config.Debugf("%#U", r)
		switch r {
		case DOLLAR:
			if state != READVAR {
				state = MAYBE
			} else {
				// This is a literal $ inside a variable: "${AB$C}"
				// TODO: Should that be an error? Is it valid?
				buf += string(r)
			}
		case OPEN:
			if state == MAYBE {
				// Peek to see if we're about to start a LITERAL !
				if len(sub) > i+1 && []rune(sub)[i+1] == BANG {
					// Treat this as part of the string, not a var
					buf += "${"
					state = READLIT
				} else {
					state = READVAR
					// We're about to start reading a variable.
					// Append the last word in the buffer if it's not empty
					if len(buf) > 0 {
						wt = STR
						words = append(words, word{t: wt, w: buf})
						buf = ""
					}
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
				buf = strings.Replace(buf, "AWS::", "", 1)
				words = append(words, word{t: wt, w: buf})
				buf = ""
				state = READSTR
			} else {
				buf += string(r)
			}
		case BANG:
			// ${!LITERAL} becomes ${LITERAL}
			if state == READLIT {
				// Don't write the ! to the string
				state = READSTR
			} else {
				// This is a ! somewhere not related to a LITERAL
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

	// Handle malformed strings, like "ABC${XYZ"
	if state != READSTR {
		// Ended the string in the middle of a variable?
		return nil, errors.New("invalid string, unclosed variable")
	}

	return words, nil
}
