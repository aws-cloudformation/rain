package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awslabs/aws-cloudformation-template-formatter/format"
)

type Table struct {
	headings   []Text
	values     [][]Text
	maxLengths []int
}

func NewTable(headings ...string) Table {
	maxLengths := make([]int, len(headings))

	for i, h := range headings {
		maxLengths[i] = len(h)
	}

	ch := make([]Text, len(headings))
	for i, h := range headings {
		ch[i] = Bold(h)
	}

	return Table{
		headings:   ch,
		values:     make([][]Text, 0),
		maxLengths: maxLengths,
	}
}

func (t *Table) Append(values ...interface{}) {
	s := make([]Text, len(values))

	for i, v := range values {
		if t, ok := v.(Text); ok {
			s[i] = t
		} else {
			s[i] = Plain(fmt.Sprint(v))
		}

		if s[i].Len() > t.maxLengths[i] {
			t.maxLengths[i] = s[i].Len()
		}
	}

	t.values = append(t.values, s)
}

func (t *Table) Sort() {
	valueMap := make(map[string][]Text)
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

func (t *Table) rowString(values []Text) string {
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

func (t *Table) YAML() string {
	out := make([]interface{}, len(t.values))

	for i, v := range t.values {
		m := make(map[string]interface{})

		for j, h := range t.headings {
			m[h.text] = v[j].text
		}

		out[i] = m
	}

	f := format.NewFormatter()
	f.SetCompact()

	return f.Format(out)
}
