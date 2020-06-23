package main

//go:generate go run generate.go

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/spf13/cobra/doc"
)

var tmpl *template.Template

func init() {
	var err error

	tmpl = template.New("README.tmpl")

	tmpl = tmpl.Funcs(template.FuncMap{
		"pad": func(s string, n int) string {
			return strings.Repeat(" ", n-len(s))
		},
	})
	if err != nil {
		panic(err)
	}

	tmpl, err = tmpl.ParseFiles("README.tmpl")
	if err != nil {
		panic(err)
	}
}

func emptyStr(s string) string {
	return ""
}

func identity(s string) string {
	if s == "rain.md" {
		return "index.md"
	}

	return s
}

func main() {
	err := doc.GenMarkdownTreeCustom(cmd.Rain, "./", emptyStr, identity)
	if err != nil {
		panic(err)
	}

	err = os.Rename("rain.md", "index.md")
	if err != nil {
		panic(err)
	}

	// Generate usage
	usage := bytes.Buffer{}
	cmd.Rain.SetOutput(&usage)
	cmd.Rain.Usage()

	// Generate README
	buf := bytes.Buffer{}
	err = tmpl.Execute(&buf, map[string]string{
		"Usage": string(usage.Bytes()),
	})
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile("../README.md", buf.Bytes(), 0644)

	cmd.Rain.GenBashCompletionFile("bash_completion.sh")
	cmd.Rain.GenZshCompletionFile("zsh_completion.sh")
}
