package ui

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/internal/console"
)

type statusCategory int

const (
	Failed statusCategory = iota
	Complete
	InProgress
	Pending
	Cancelled
)

var StatusColour = map[statusCategory]func(...interface{}) string{
	Pending:    console.Plain,
	InProgress: console.Blue,
	Failed:     console.Red,
	Complete:   console.Green,
	Cancelled:  console.Grey,
}

type StatusRep struct {
	Category statusCategory
	Symbol   string
}

func (s *StatusRep) String() string {
	return fmt.Sprint(StatusColour[s.Category](s.Symbol))
}

func MapStatus(status string) *StatusRep {
	rep := StatusRep{}

	// Colour
	switch {
	case status == "REVIEW_IN_PROGRESS":
		rep.Category = Pending

	case strings.HasSuffix(status, "_FAILED"), strings.HasPrefix(status, "DELETE_"), strings.Contains(status, "ROLLBACK"):
		rep.Category = Failed

	case strings.HasSuffix(status, "_IN_PROGRESS"):
		rep.Category = InProgress

	case strings.HasSuffix(status, "_COMPLETE"):
		rep.Category = Complete

	// stack set statuses
	case strings.HasSuffix(status, "ACTIVE"):
		rep.Category = Complete

	// stack set instance statuses
	case strings.HasSuffix(status, "SUCCEEDED"):
		rep.Category = Complete

	case strings.HasSuffix(status, "CANCELLED"):
	case strings.HasSuffix(status, "OUTDATED"):
		rep.Category = Cancelled

	case strings.HasSuffix(status, "FAILED"):
	case strings.HasSuffix(status, "INOPERABLE"):
		rep.Category = Failed

	case strings.HasSuffix(status, "RUNNING"):
		rep.Category = InProgress

	default:
		rep.Category = Pending
	}

	// Symbol
	switch {
	case status == "REVIEW_IN_PROGRESS":
		rep.Symbol = "."

	case strings.HasSuffix(status, "_FAILED"):
		rep.Symbol = "x"

	case strings.HasSuffix(status, "_IN_PROGRESS"):
		rep.Symbol = "o"

	case strings.HasSuffix(status, "_COMPLETE"):
		rep.Symbol = "✓"

	default:
		rep.Symbol = "."
	}

	return &rep
}

// Colourise wraps a message in an appropriate colour
// based on the accompanying status string
func Colourise(msg, status string) string {
	rep := MapStatus(status)

	return StatusColour[rep.Category](msg)
}

// ColouriseStatus wraps a status code in an appropriate colour
func ColouriseStatus(status string) string {
	return Colourise(status, status)
}

// ColouriseDiff wraps a diff object in nice colours
func ColouriseDiff(d diff.Diff, longFormat bool) string {
	output := strings.Builder{}

	parts := strings.Split(d.Format(longFormat), "\n")

	for i, line := range parts {
		switch {
		case strings.HasPrefix(line, diff.Added.String()):
			output.WriteString(console.Green(line))
		case strings.HasPrefix(line, diff.Removed.String()):
			output.WriteString(console.Red(line))
		case strings.HasPrefix(line, diff.Changed.String()):
			output.WriteString(console.Blue(line))
		case strings.HasPrefix(line, diff.Involved.String()):
			output.WriteString(console.Grey(line))
		default:
			output.WriteString(console.Plain(line))
		}

		if i < len(parts)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}
