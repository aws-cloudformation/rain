/*
Package format provides functions for formatting CloudFormation templates
using an opinionated, idiomatic format as used in AWS documentation.

For each function, CloudFormation templates should be represented using
a map[string]interface{} as output by other libraries that parse JSON/YAML
such as github.com/awslabs/goformation and encoding/json.

Comments can be passed along with the template data in the following format:

	map[interface{}]interface{}{
		"": "This is a top-level comment",
		"Resources": map[interface{}]interface{}{
			"": "This is a comment on the whole `Resources` property",
			"MyBucket": map[interface{}]interface{}{
				"Properties": map[interface{}]interface{}{
					"BucketName": "This is a comment on BucketName",
				},
			},
		},
	}

Empty string keys are taken to represent a comment on the overall node
that the comment is attached to. Numeric keys can be used to reference
elements of arrays in the source data.
*/
package format

type Style int

const (
	YAML Style = iota
	JSON
)

type Options struct {
	Style   Style
	Compact bool
}

type Formatter struct {
	Options Options
}

func New(options Options) Formatter {
	return Formatter{
		Options: options,
	}
}

func (f *Formatter) Format(data interface{}) string {
	return f.FormatWithComments(data, nil)
}

func (f *Formatter) FormatWithComments(data interface{}, comments map[interface{}]interface{}) string {
	return newEncoder(*f, value{data, comments}).format()
}

// FIXME: This needs to be refactored
// SortKeys sorts the given keys
// based on their location within a CloudFormation template
// as given by the path parameter
/*
func SortKeys(keys []string, path []interface{}) {
	data := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		data[key] = nil
	}

	newKeys := sortKeys(data, path)

	for i, _ := range keys {
		keys[i] = newKeys[i]
	}
}
*/
