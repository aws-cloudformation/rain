package cfn

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

// converts map of strings to a slice of types.Tag
func MakeTags(tags map[string]string) []types.Tag {
	out := make([]types.Tag, 0)

	for key, value := range tags {
		out = append(out, types.Tag{
			Key:   ptr.String(key),
			Value: ptr.String(value),
		})
	}

	return out
}

// uniqueStrings returns a unique subset of the string slice provided.
func UniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
