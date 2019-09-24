package main

//go:generate go run generate.go

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/spf13/cobra/doc"
)

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
	err := doc.GenMarkdownTreeCustom(cmd.Root, "./", emptyStr, identity)
	if err != nil {
		panic(err)
	}

	err = os.Rename("rain.md", "index.md")
	if err != nil {
		panic(err)
	}
}
