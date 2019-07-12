// Package table defines the Table type which can be used for displaying
// tabular data with bold headings and properly spaced columns.
package table

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/console/text"
)

// A Table holds column and row information and can be printed using String()
type Table struct {
	headings   []text.Text
	values     [][]text.Text
	maxLengths []int
}

// New returns a new Table with the supplied column headings
func New(headings ...string) Table {
	maxLengths := make([]int, len(headings))

	for i, h := range headings {
		maxLengths[i] = len(h)
	}

	ch := make([]text.Text, len(headings))
	for i, h := range headings {
		ch[i] = text.Bold(h)
	}

	return Table{
		headings:   ch,
		values:     make([][]text.Text, 0),
		maxLengths: maxLengths,
	}
}

// Append adds a new row to the table
func (t *Table) Append(values ...interface{}) {
	s := make([]text.Text, len(values))

	for i, v := range values {
		if t, ok := v.(text.Text); ok {
			s[i] = t
		} else {
			s[i] = text.Plain(fmt.Sprint(v))
		}

		if s[i].Len() > t.maxLengths[i] {
			t.maxLengths[i] = s[i].Len()
		}
	}

	t.values = append(t.values, s)
}

// Sort sorts the contents of the table alphabetically
func (t *Table) Sort() {
	valueMap := make(map[string][]text.Text)
	valueList := make([]string, len(t.values))

	for i, v := range t.values {
		vs := fmt.Sprint(v)

		valueMap[vs] = v
		valueList[i] = vs
	}

	sort.Strings(valueList)

	for i, vs := range valueList {
		t.values[i] = valueMap[vs]
	}
}

func (t *Table) rowString(values []text.Text) string {
	output := strings.Builder{}

	for i, v := range values {
		output.WriteString("| ")
		output.WriteString(v.String())
		output.WriteString(strings.Repeat(" ", t.maxLengths[i]-v.Len()))
		output.WriteString(" ")
	}
	output.WriteString("|\n")

	return output.String()
}

// String converts the table into a string
func (t *Table) String() string {
	output := strings.Builder{}

	// Top line
	for _, l := range t.maxLengths {
		output.WriteString("+" + strings.Repeat("-", l+2))
	}
	output.WriteString("+\n")

	// Headings
	output.WriteString(t.rowString(t.headings))

	// Heading underline
	for _, l := range t.maxLengths {
		output.WriteString("|" + strings.Repeat("-", l+2))
	}
	output.WriteString("|\n")

	// Rows
	for _, v := range t.values {
		output.WriteString(t.rowString(v))
	}

	// Bottom line
	for _, l := range t.maxLengths {
		output.WriteString("+" + strings.Repeat("-", l+2))
	}
	output.WriteString("+\n")

	return output.String()
}

// YAML returns a string presentation of the table as a YAML squence of mappings
func (t *Table) YAML() string {
	out := make([]interface{}, len(t.values))

	for i, v := range t.values {
		m := make(map[string]interface{})

		for j, h := range t.headings {
			m[h.Plain()] = v[j].Plain()
		}

		out[i] = m
	}

	return format.Anything(out, format.Options{})
}
