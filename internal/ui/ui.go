package ui

import (
	"errors"
	"fmt"
	"strings"

	smithy "github.com/aws/smithy-go"
)

// Errorf wraps an error, extracting the AWS API error if it exists
func Errorf(err error, message string, parts ...interface{}) error {
	message = fmt.Sprintf(message, parts...)

	// Pull out API errors
	var apiErr = &smithy.GenericAPIError{}
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: %s", message, apiErr.Message)
	}

	return fmt.Errorf("%s: %w", message, err)
}

// Indent adds prefix to every line of in
func Indent(prefix string, in string) string {
	return prefix + strings.Join(strings.Split(strings.TrimSpace(in), "\n"), "\n"+prefix)
}
