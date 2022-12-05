package ui

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/internal/console"
)

type statusCategory int

const (
	failed statusCategory = iota
	complete
	inProgress
	pending
	cancelled
)

var statusColour = map[statusCategory]func(...interface{}) string{
	pending:    console.Plain,
	inProgress: console.Blue,
	failed:     console.Red,
	complete:   console.Green,
	cancelled:  console.Grey,
}

type statusRep struct {
	category statusCategory
	symbol   string
}

func (s *statusRep) String() string {
	return fmt.Sprint(statusColour[s.category](s.symbol))
}

func mapStatus(status string) *statusRep {
	rep := statusRep{}

	// Colour
	switch {
	case status == "REVIEW_IN_PROGRESS":
		rep.category = pending

	case strings.HasSuffix(status, "_FAILED"), strings.HasPrefix(status, "DELETE_"), strings.Contains(status, "ROLLBACK"):
		rep.category = failed

	case strings.HasSuffix(status, "_IN_PROGRESS"):
		rep.category = inProgress

	case strings.HasSuffix(status, "_COMPLETE"):
		rep.category = complete

	// stack set statuses
	case strings.HasSuffix(status, "ACTIVE"):
		rep.category = complete

	// stack set instance statuses
	case strings.HasSuffix(status, "SUCCEEDED"):
		rep.category = complete

	case strings.HasSuffix(status, "CANCELLED"):
	case strings.HasSuffix(status, "OUTDATED"):
		rep.category = cancelled

	case strings.HasSuffix(status, "FAILED"):
	case strings.HasSuffix(status, "INOPERABLE"):
		rep.category = failed

	case strings.HasSuffix(status, "RUNNING"):
		rep.category = inProgress

	default:
		rep.category = pending
	}

	// Symbol
	switch {
	case status == "REVIEW_IN_PROGRESS":
		rep.symbol = "."

	case strings.HasSuffix(status, "_FAILED"):
		rep.symbol = "x"

	case strings.HasSuffix(status, "_IN_PROGRESS"):
		rep.symbol = "o"

	case strings.HasSuffix(status, "_COMPLETE"):
		rep.symbol = "✓"

	default:
		rep.symbol = "."
	}

	return &rep
}

// Colourise wraps a message in an appropriate colour
// based on the accompanying status string
func Colourise(msg, status string) string {
	rep := mapStatus(status)

	return statusColour[rep.category](msg)
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
