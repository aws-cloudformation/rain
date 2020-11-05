package validate

import (
	"github.com/aws-cloudformation/rain/cft"
)

// Errors is used to wrap a slice of *cft.Comment for convenience
type errors []*cft.Comment

func (e *errors) add(message string, path ...interface{}) {
	*e = append(*e, &cft.Comment{
		Path:  path,
		Value: message,
	})
}
