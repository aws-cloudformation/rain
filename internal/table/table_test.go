package table

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/google/go-cmp/cmp"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
)

func TestFormatter(t *testing.T) {
	t.Parallel()

	var formatter Formatter

	fn := func(a string, b ...interface{}) string { return "" }
	f := Formatter(fn)

	assert.IsType(t, formatter, f)
}

func TestTable_New(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}
	New("foo", "bar").WithWriter(&buf).Print()
	out := buf.String()

	assert.Contains(t, out, "foo")
	assert.Contains(t, out, "bar")

	buf.Reset()
	New().WithWriter(&buf).Print()
	out = buf.String()

	assert.Empty(t, strings.TrimSpace(out))
}

func TestTable_WithHeaderFormatter(t *testing.T) {
	t.Parallel()

	uppercase := func(f string, v ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(f, v...))
	}
	buf := bytes.Buffer{}

	tbl := New("foo", "bar").WithWriter(&buf).WithHeaderFormatter(uppercase)
	tbl.Print()
	out := buf.String()

	assert.Contains(t, out, "FOO")
	assert.Contains(t, out, "BAR")

	buf.Reset()
	tbl.WithHeaderFormatter(nil).Print()
	out = buf.String()

	assert.Contains(t, out, "foo")
	assert.Contains(t, out, "bar")
}

func TestTable_WithFirstColumnFormatter(t *testing.T) {
	t.Parallel()

	uppercase := func(f string, v ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(f, v...))
	}

	buf := bytes.Buffer{}

	tbl := New("foo", "bar").WithWriter(&buf).WithFirstColumnFormatter(uppercase).AddRow("fizz", "buzz")
	tbl.Print()
	out := buf.String()

	config.Debug = true
	config.Debugf(out)

	assert.Contains(t, out, "foo")
	assert.Contains(t, out, "bar")
	assert.Contains(t, out, "FIZZ")
	assert.Contains(t, out, "buzz")

	buf.Reset()
	tbl.WithFirstColumnFormatter(nil).Print()
	out = buf.String()

	assert.Contains(t, out, "fizz")
	assert.Contains(t, out, "buzz")
}

func TestTable_WithPadding(t *testing.T) {
	t.Parallel()

	// zero value
	buf := bytes.Buffer{}
	tbl := New("foo", "bar").WithWriter(&buf).WithPadding(0)
	tbl.Print()
	out := buf.String()
	assert.Contains(t, out, "foobar")

	// positive value
	buf.Reset()
	tbl.WithPadding(4).Print()
	out = buf.String()
	assert.Contains(t, out, "foo    bar    ")

	// negative value
	buf.Reset()
	tbl.WithPadding(-1).Print()
	out = buf.String()
	assert.Contains(t, out, "foobar")
}

func TestTable_WithWriter(t *testing.T) {
	t.Parallel()

	// not that we haven't been using it in all these tests but:
	buf := bytes.Buffer{}
	New("foo", "bar").WithWriter(&buf).Print()
	assert.NotEmpty(t, buf.String())

	stdout := os.Stdout
	temp, _ := os.CreateTemp("", "")
	os.Stdout = temp
	defer func() {
		os.Stdout = stdout
		temp.Close()
	}()

	New("foo", "bar").WithWriter(nil).Print()
	temp.Seek(0, 0)

	out, _ := io.ReadAll(temp)
	assert.NotEmpty(t, out)
}

func TestTable_AddRow(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}
	tbl := New("foo", "bar").WithWriter(&buf).AddRow("fizz", "buzz")
	tbl.Print()
	out := buf.String()
	assert.Contains(t, out, "fizz")
	assert.Contains(t, out, "buzz")
	lines := strings.Count(out, "\n")

	// empty should add empty line
	buf.Reset()
	tbl.AddRow().Print()
	assert.Equal(t, lines+1, strings.Count(buf.String(), "\n"))

	// less than one will fill left-to-right
	buf.Reset()
	tbl.AddRow("cat").Print()
	assert.Contains(t, buf.String(), "\ncat")

	// more than initial length are truncated
	buf.Reset()
	tbl.AddRow("bippity", "boppity", "boo").Print()
	assert.NotContains(t, buf.String(), "boo")

	// check the full table
	buf.Reset()
	tbl.Print()
	expected := `foo      bar      
fizz     buzz     
                  
cat               
bippity  boppity  
`
	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Fatalf("table mismatch (-expected +got):\n%s\nout=%#v", diff, buf.String())
	}
}

func TestTable_AddRow_WithNewLines(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}
	tbl := New("foo", "bar").WithWriter(&buf).AddRow("fizz", "buzz")

	// Add some rows
	tbl.AddRow()
	tbl.AddRow("cat")

	// add an entry that contains new lines
	tbl.AddRow("bippity", "boppity\nboop")

	// Add a couple more rows
	tbl.AddRow("a", "b")
	tbl.AddRow("c", "d")

	// and another entry with more new lines
	tbl.AddRow("1\n2", "x\ny\nz")

	// check the full table
	buf.Reset()
	tbl.Print()
	expected := `foo      bar      
fizz     buzz     
                  
cat               
bippity  boppity  
         boop     
a        b        
c        d        
1        x        
2        y        
         z        
`
	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Fatalf("table mismatch (-expected +got):\n%s\nout=%#v", diff, buf.String())
	}
}

func TestTable_SetRows(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}
	tbl := New("foo", "bar").WithWriter(&buf).SetRows([][]string{
		{"fizz", "buzz"},
		{"lorem", "ipsum"},
	})
	tbl.Print()
	out := buf.String()
	assert.Contains(t, out, "fizz")
	assert.Contains(t, out, "buzz")
	assert.Contains(t, out, "lorem")
	assert.Contains(t, out, "ipsum")
	assert.Equal(t, 3, strings.Count(out, "\n"))

	// empty should remove all rows
	buf.Reset()
	tbl.SetRows([][]string{}).Print()
	assert.Equal(t, 1, strings.Count(buf.String(), "\n"))

	// less than one will fill left-to-right
	buf.Reset()
	tbl.SetRows([][]string{{"cat"}}).Print()
	assert.Contains(t, buf.String(), "\ncat")

	// more than initial length are truncated
	buf.Reset()
	tbl.SetRows([][]string{
		{"lorem", "ipsum"},
		{"bippity", "boppity", "boo"},
	}).Print()
	assert.NotContains(t, buf.String(), "boo")
}

func TestTable_WithWidthFunc(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}

	New("", "").
		WithWriter(&buf).
		WithPadding(1).
		WithWidthFunc(runewidth.StringWidth).
		AddRow("请求", "alpha").
		AddRow("abc", "beta").
		Print()

	actual := buf.String()
	assert.Contains(t, actual, "请求 alpha")
	assert.Contains(t, actual, "abc  beta")
}
