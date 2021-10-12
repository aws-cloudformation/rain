// Package spec contains generated models for CloudFormation and IAM
package spec

//go:generate go run internal/main.go internal/format.go

import "strings"

// ResolveResource returns a list of possible Resource names for
// the provided suffix
func ResolveResource(suffix string) []string {
	suffix = strings.ToLower(suffix)

	options := make([]string, 0)

	for typeName := range Cfn {
		if strings.HasSuffix(strings.ToLower(typeName), suffix) {
			options = append(options, typeName)
		}
	}

	return options
}
