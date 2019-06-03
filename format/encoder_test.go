package format

import (
	"testing"

	"github.com/aws-cloudformation/rain/parse"
)

var inputTemplate = `
Outputs:
  Cake:
    Value: Lie
Resources:
  Bucket:
    Properties:
      BucketName: !Sub Chum${Suffix}
    Type: AWS::S3::Bucket
Parameters:
  Suffix:
    Default: ""
    Type: String
`

var source map[string]interface{}

func init() {
	var err error

	source, err = parse.ReadString(inputTemplate)
	if err != nil {
		panic(err)
	}
}

func BenchmarkJson(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Json(source)
	}
}

func BenchmarkYaml(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Yaml(source)
	}
}

func BenchmarkVerifyJson(b *testing.B) {
	output := Json(source)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		parse.VerifyOutput(source, output)
	}
}

func BenchmarkVerifyYaml(b *testing.B) {
	output := Yaml(source)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		parse.VerifyOutput(source, output)
	}
}
