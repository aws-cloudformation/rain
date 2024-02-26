//go:build func_test

package pkg_test

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/google/go-cmp/cmp"
)

const fileName = "test.txt"

const hash = "7e81f4270269cd5111c4926e19de731fb38c6dbf07059d14f4591ce5d8ddd770"
const bucket = "rain-artifacts-1234567890-us-east-1"
const region = "us-east-1"
const packagedTemplateHash = "28f611b4c6d562fa459e7131b167960cd1b5dc5a0238da157ee1196d4679a3cc"

var zipHash = "potato"

func init() {
	// Generate the zip so we can compare the hash
	// we can't do this ahead of time because git will change the file's modification time
	tmpFile, err := os.CreateTemp(os.TempDir(), "*.zip")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())

	w := zip.NewWriter(tmpFile)

	info, err := os.Stat(fileName)
	if err != nil {
		panic(err)
	}

	fh, err := zip.FileInfoHeader(info)
	if err != nil {
		panic(err)
	}
	fh.Name = fileName
	fh.Method = zip.Deflate

	out, err := w.CreateHeader(fh)
	if err != nil {
		panic(err)
	}

	in, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(out, in)
	if err != nil {
		panic(err)
	}

	w.Close()
	tmpFile.Close()

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		panic(err)
	}

	zipHash = fmt.Sprintf("%x", (sha256.Sum256(content)))
}

func compare(t *testing.T, in cft.Template, path string, expected interface{}) {
	out, err := pkg.Template(in, "./", nil)
	if err != nil {
		t.Error(err)
	}

	n := s11n.MatchOne(out.Node, path)

	var actual interface{}
	err = n.Decode(&actual)
	if err != nil {
		t.Error(err)
	}

	if d := cmp.Diff(expected, actual); d != "" {
		t.Error(d)
	}
}

func TestEmbed(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Embed": fileName,
		},
	})

	compare(t, in, "Test", "This: is a test")
}

func TestInclude(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Include": fileName,
		},
	})
	compare(t, in, "Test", map[string]interface{}{"This": "is a test"})
}

func TestEnv(t *testing.T) {
	os.Setenv("RAIN_TEST_ENV_EXISTS", "foo")
	in1, _ := parse.Map(map[string]interface{}{
		"Success": map[string]interface{}{
			"Rain::Env": "RAIN_TEST_ENV_EXISTS",
		},
	})
	in2, _ := parse.Map(map[string]interface{}{
		"Failure": map[string]interface{}{
			"Rain::Env": "RAIN_TEST_ENV_DOESNT_EXISTS",
		},
	})
	compare(t, in1, "Success", "foo")
	_, err2 := pkg.Template(in2, "./", nil)
	if err2 == nil {
		t.Errorf("Expected error since %q environment variable doesn't exist", "RAIN_TEST_ENV_DOESNT_EXISTS")
	}
}

func TestS3Http(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3Http": fileName,
		},
	})

	compare(t, in, "Test", fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, hash))
}

func TestS3(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": fileName,
		},
	})

	compare(t, in, "Test", fmt.Sprintf("s3://%s/%s", bucket, hash))
}

func TestS3Defaults(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": map[string]interface{}{
				"Path": fileName,
			},
		},
	})

	compare(t, in, "Test", fmt.Sprintf("s3://%s/%s", bucket, hash))
}

func TestS3Object(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": map[string]interface{}{
				"Path":           fileName,
				"BucketProperty": "RainS3Bucket",
				"KeyProperty":    "RainS3Key",
			},
		},
	})

	compare(t, in, "Test", map[string]interface{}{
		"RainS3Bucket": bucket,
		"RainS3Key":    hash,
	})
}

func TestS3ObjectHttp(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": map[string]interface{}{
				"Path":   fileName,
				"Format": "Http",
			},
		},
	})

	compare(t, in, "Test", fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, hash))
}

func TestS3ObjectURIZip(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": map[string]interface{}{
				"Path":   fileName,
				"Format": "URI",
				"Zip":    true,
			},
		},
	})

	compare(t, in, "Test", fmt.Sprintf("s3://%s/%s", bucket, zipHash))
}

func TestRecursion(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Include": "recurse.yaml",
		},
	})

	compare(t, in, "Test", map[string]interface{}{"Description": map[string]interface{}{"This": "is a test"}})
}

func TestWrappedTypes(t *testing.T) {
	s3URI := fmt.Sprintf("s3://%s/%s", bucket, hash)
	s3ZipURI := fmt.Sprintf("s3://%s/%s", bucket, zipHash)
	httpURI := fmt.Sprintf("https://%s.s3.us-east-1.amazonaws.com/%s", bucket, hash)

	for _, testCase := range []struct {
		typeName string
		propName string
		expected interface{}
	}{
		{"AWS::Serverless::Function", "CodeUri", s3ZipURI},
		{"AWS::Serverless::Api", "DefinitionUri", s3URI},
		{"AWS::AppSync::GraphQLSchema", "DefinitionS3Location", s3URI},
		{"AWS::AppSync::Resolver", "RequestMappingTemplateS3Location", s3URI},
		{"AWS::AppSync::Resolver", "ResponseMappingTemplateS3Location", s3URI},
		{"AWS::AppSync::FunctionConfiguration", "RequestMappingTemplateS3Location", s3URI},
		{"AWS::AppSync::FunctionConfiguration", "ResponseMappingTemplateS3Location", s3URI},
		{"AWS::ServerlessRepo::Application", "ReadmeUrl", s3URI},
		{"AWS::ServerlessRepo::Application", "LicenseUrl", s3URI},
		{"AWS::Glue::Job", "Command/ScriptLocation", s3URI},
		{"AWS::Serverless::LayerVersion", "ContentUri", s3ZipURI},
		{"AWS::Serverless::Application", "Location", httpURI},
		{"AWS::Lambda::Function", "Code", map[string]interface{}{"S3Bucket": bucket, "S3Key": zipHash}},
		{"AWS::ElasticBeanstalk::ApplicationVersion", "SourceBundle", map[string]interface{}{"S3Bucket": bucket, "S3Key": hash}},
		{"AWS::Lambda::LayerVersion", "Content", map[string]interface{}{"S3Bucket": bucket, "S3Key": zipHash}},
		{"AWS::ApiGateway::RestApi", "BodyS3Location", map[string]interface{}{"Bucket": bucket, "Key": hash}},
		{"AWS::StepFunctions::StateMachine", "DefinitionS3Location", map[string]interface{}{"Bucket": bucket, "Key": hash}},
		{"AWS::CloudFormation::Stack", "TemplateURL", httpURI},
	} {
		props := make(map[string]interface{})

		parts := strings.Split(testCase.propName, "/")

		props[parts[len(parts)-1]] = fileName

		for i := len(parts) - 2; i >= 0; i-- {
			part := parts[i]
			props = map[string]interface{}{
				part: props,
			}
		}

		in, _ := parse.Map(map[string]interface{}{
			"Resources": map[string]interface{}{
				"MyResource": map[string]interface{}{
					"Type":       testCase.typeName,
					"Properties": props,
				},
			},
		})

		compare(t, in, fmt.Sprintf("Resources/MyResource/Properties/%s", testCase.propName), testCase.expected)
	}
}

func TestTemplates(t *testing.T) {
	httpURI := fmt.Sprintf("https://%s.s3.us-east-1.amazonaws.com/%s", bucket, packagedTemplateHash)

	for _, testCase := range []struct {
		typeName string
		propName string
		expected interface{}
	}{
		{"AWS::Serverless::Application", "Location", httpURI},
		{"AWS::CloudFormation::Stack", "TemplateURL", httpURI},
	} {
		props := make(map[string]interface{})

		parts := strings.Split(testCase.propName, "/")

		props[parts[len(parts)-1]] = "recurse.yaml"

		for i := len(parts) - 2; i >= 0; i-- {
			part := parts[i]
			props = map[string]interface{}{
				part: props,
			}
		}

		in, _ := parse.Map(map[string]interface{}{
			"Resources": map[string]interface{}{
				"MyResource": map[string]interface{}{
					"Type":       testCase.typeName,
					"Properties": props,
				},
			},
		})

		compare(t, in, fmt.Sprintf("Resources/MyResource/Properties/%s", testCase.propName), testCase.expected)
	}
}
