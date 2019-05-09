package cmd

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
	"gopkg.in/yaml.v2"
)

const (
	ADD = "\033[32m>>>"
	DEL = "\033[31m<<<"
	END = "\033[0m"
)

func init() {
	Commands["diff"] = Command{
		Type: TEMPLATE,
		Run:  diffCommand,
		Help: "Compare templates with other templates or stacks",
	}
}

func diffCommand(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: rain diff <left> <right>")
		os.Exit(1)
	}

	leftFn, rightFn := args[0], args[1]

	left, err := parse.ReadFile(leftFn)
	if err != nil {
		panic(err)
	}

	right, err := parse.ReadFile(rightFn)
	if err != nil {
		panic(err)
	}

	fmt.Print(compare(left, right, ""))
}

func render(value interface{}, indent string) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		panic(err)
	}

	rows := strings.Split(string(y), "\n")

	for i, row := range rows {
		rows[i] = fmt.Sprintf("%s  %s", indent, row)
	}

	return strings.Join(rows, "\n") + "\n"
}

func compare(left, right interface{}, indent string) string {
	lType := reflect.TypeOf(left)
	rType := reflect.TypeOf(right)

	if lType != rType {
		return fmt.Sprintf("%s! Differing types\n", indent)
	}

	switch l := left.(type) {
	case map[string]interface{}:
		r := right.(map[string]interface{})

		output := ""

		names := make(map[string]bool)

		for name, _ := range l {
			names[name] = true
		}

		for name, _ := range r {
			names[name] = true
		}

		for name, _ := range names {
			if _, ok := l[name]; !ok {
				output += fmt.Sprintf("%s%s %s\n", indent, DEL, name)
				output += END
			} else if _, ok := r[name]; !ok {
				output += fmt.Sprintf("%s%s %s:\n", indent, ADD, name)
				output += render(l[name], indent+"  ")
				output += END
			} else {
				diff := compare(l[name], r[name], indent+"  ")
				if diff != "" {
					output += fmt.Sprintf("%s%s:\n", indent, name)
					output += diff
				}
			}
		}

		return output
	case []interface{}:
		r := right.([]interface{})

		output := ""

		for i := 0; i < int(math.Max(float64(len(l)), float64(len(r)))); i++ {
			if i > len(l)-1 {
				output += fmt.Sprintf("%s%s %d\n", indent, DEL, i)
				output += END
			} else if i > len(r)-1 {
				output += fmt.Sprintf("%s%s %d\n:", indent, ADD, i)
				output += render(l[i], indent+"  ")
				output += END
			} else {
				diff := compare(l[i], r[i], indent+"  ")
				if diff != "" {
					output += fmt.Sprintf("%s%i:\n", indent, i)
					output += diff
				}
			}
		}

		return output
	default:
		if left != right {
			return fmt.Sprintf("%s%s %s\n", indent, DEL, right) + fmt.Sprintf("%s%s %s\n", indent, ADD, left)
		}

		return ""
	}
}
