package util

import (
	"fmt"
	"strings"

	"github.com/awslabs/aws-cloudformation-template-formatter/format"
)

type Table struct {
	headings   []string
	values     [][]string
	maxLengths []int
}

func NewTable(headings ...string) Table {
	maxLengths := make([]int, len(headings))

	for i, h := range headings {
		maxLengths[i] = len(h)
	}

	return Table{
		headings:   headings,
		values:     make([][]string, 0),
		maxLengths: maxLengths,
	}
}

func (t *Table) Append(values ...interface{}) {
	s := make([]string, len(values))

	for i, v := range values {
		s[i] = fmt.Sprint(v)

		if len(s[i]) > t.maxLengths[i] {
			t.maxLengths[i] = len(s[i])
		}
	}

	t.values = append(t.values, s)
}

func (t *Table) rowString(values []string) string {
	output := strings.Builder{}

	for i, v := range values {
		output.WriteString("| ")
		output.WriteString(v)
		output.WriteString(strings.Repeat(" ", t.maxLengths[i]-len(v)))
		output.WriteString(" ")
	}
	output.WriteString("|\n")

	return output.String()
}

func (t *Table) String() string {
	output := strings.Builder{}

	// Top line
	for _, l := range t.maxLengths {
		output.WriteString("+")
		output.WriteString(strings.Repeat("-", l+2))
	}
	output.WriteString("+\n")

	// Headings
	output.WriteString(t.rowString(t.headings))

	// Heading underline
	for _, l := range t.maxLengths {
		output.WriteString("|")
		output.WriteString(strings.Repeat("-", l+2))
	}
	output.WriteString("|\n")

	// Rows
	for _, v := range t.values {
		output.WriteString(t.rowString(v))
	}

	// Bottom line
	for _, l := range t.maxLengths {
		output.WriteString("+")
		output.WriteString(strings.Repeat("-", l+2))
	}
	output.WriteString("+\n")

	return output.String()
}

func (t *Table) YAML() string {
	out := make([]interface{}, len(t.values))

	for i, v := range t.values {
		m := make(map[string]interface{})

		for j, h := range t.headings {
			m[h] = v[j]
		}

		out[i] = m
	}

	f := format.NewFormatter()
	f.SetCompact()

	return f.Format(out)
}
