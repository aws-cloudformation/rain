package format_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

const input = `
Outputs:
  Bucket1:
    Value: !GetAtt Bucket1.Arn # Short GetAtt
  Bucket2: # Bucket comment
    Value:
      Fn::GetAtt: # GetAtt comment
        - Bucket2
        - Arn # Arn comment

Description: |
  An example template for testing rain fmt command.

# Multiline comment
# starting at indent 0
Resources:
  Bucket2:
    Properties:
      BucketName: !Ref Name # Ref: comment
    Type: "AWS::S3::Bucket"
  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub ${Bucket2}-newer
  Func1:
    Type: AWS::Lambda::Function
    Properties:
      Role: !Sub "arn:aws:iam::${AWS::AccountID}:role/lambda-basic"
      Runtime: python3.7
      Handler: index.handler
      Code:
        ZipFile: |
          import boto3

          def handler: 
            """Example."""

            print('hello')
  Instance1:
    Type: AWS::EC2::Instance
    Properties:
      UserData: !Base64
        Fn::Sub:
          - |
            #!/bin/bash -xe
            apt-get update

            apt-get upgrade -y
Rules:
  Rule1:
    RuleCondition: !Equals
      - !Ref Environment
      - test
    Assertions:
      - Assert:
          Fn::Contains:
            - - a1.medium
            - !Ref InstanceType
Parameters:
  Name:
    Type: String
`

const expectedYaml = `Description: |
  An example template for testing rain fmt command.

Parameters:
  Name:
    Type: String

Rules:
  Rule1:
    RuleCondition: !Equals
      - !Ref Environment
      - test
    Assertions:
      - Assert:
          Fn::Contains:
            - - a1.medium
            - !Ref InstanceType

# Multiline comment
# starting at indent 0
Resources:
  Bucket2:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref Name # Ref: comment

  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub ${Bucket2}-newer

  Func1:
    Type: AWS::Lambda::Function
    Properties:
      Role: !Sub arn:aws:iam::${AWS::AccountID}:role/lambda-basic
      Runtime: python3.7
      Handler: index.handler
      Code:
        ZipFile: "import boto3\n\ndef handler: \n  \"\"\"Example.\"\"\"\n\n  print('hello')\n"

  Instance1:
    Type: AWS::EC2::Instance
    Properties:
      UserData: !Base64
        Fn::Sub:
          - |
            #!/bin/bash -xe
            apt-get update

            apt-get upgrade -y

Outputs:
  Bucket1:
    Value: !GetAtt Bucket1.Arn # Short GetAtt

  Bucket2: # Bucket comment
    Value: !GetAtt Bucket2.Arn # GetAtt comment Arn comment
`

const expectedYamlUnsorted = `Outputs:
  Bucket1:
    Value: !GetAtt Bucket1.Arn # Short GetAtt

  Bucket2: # Bucket comment
    Value: !GetAtt Bucket2.Arn # GetAtt comment Arn comment

Description: |
  An example template for testing rain fmt command.

# Multiline comment
# starting at indent 0
Resources:
  Bucket2:
    Properties:
      BucketName: !Ref Name # Ref: comment
    Type: AWS::S3::Bucket

  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub ${Bucket2}-newer

  Func1:
    Type: AWS::Lambda::Function
    Properties:
      Role: !Sub arn:aws:iam::${AWS::AccountID}:role/lambda-basic
      Runtime: python3.7
      Handler: index.handler
      Code:
        ZipFile: "import boto3\n\ndef handler: \n  \"\"\"Example.\"\"\"\n\n  print('hello')\n"

  Instance1:
    Type: AWS::EC2::Instance
    Properties:
      UserData: !Base64
        Fn::Sub:
          - |
            #!/bin/bash -xe
            apt-get update

            apt-get upgrade -y

Rules:
  Rule1:
    RuleCondition: !Equals
      - !Ref Environment
      - test
    Assertions:
      - Assert:
          Fn::Contains:
            - - a1.medium
            - !Ref InstanceType

Parameters:
  Name:
    Type: String
`

const expectedJson = `{
    "Description": "An example template for testing rain fmt command.\n",
    "Parameters": {
        "Name": {
            "Type": "String"
        }
    },
    "Rules": {
        "Rule1": {
            "RuleCondition": {
                "Fn::Equals": [
                    {
                        "Ref": "Environment"
                    },
                    "test"
                ]
            },
            "Assertions": [
                {
                    "Assert": {
                        "Fn::Contains": [
                            [
                                "a1.medium"
                            ],
                            {
                                "Ref": "InstanceType"
                            }
                        ]
                    }
                }
            ]
        }
    },
    "Resources": {
        "Bucket2": {
            "Type": "AWS::S3::Bucket",
            "Properties": {
                "BucketName": {
                    "Ref": "Name"
                }
            }
        },
        "Bucket1": {
            "Type": "AWS::S3::Bucket",
            "Properties": {
                "BucketName": {
                    "Fn::Sub": "${Bucket2}-newer"
                }
            }
        },
        "Func1": {
            "Type": "AWS::Lambda::Function",
            "Properties": {
                "Role": {
                    "Fn::Sub": "arn:aws:iam::${AWS::AccountID}:role/lambda-basic"
                },
                "Runtime": "python3.7",
                "Handler": "index.handler",
                "Code": {
                    "ZipFile": "import boto3\n\ndef handler: \n  \"\"\"Example.\"\"\"\n\n  print('hello')\n"
                }
            }
        },
        "Instance1": {
            "Type": "AWS::EC2::Instance",
            "Properties": {
                "UserData": {
                    "Fn::Base64": {
                        "Fn::Sub": [
                            "#!/bin/bash -xe\napt-get update\n\napt-get upgrade -y\n"
                        ]
                    }
                }
            }
        }
    },
    "Outputs": {
        "Bucket1": {
            "Value": {
                "Fn::GetAtt": [
                    "Bucket1",
                    "Arn"
                ]
            }
        },
        "Bucket2": {
            "Value": {
                "Fn::GetAtt": [
                    "Bucket2",
                    "Arn"
                ]
            }
        }
    }
}
`

const expectedUnsortedJson = `{
    "Outputs": {
        "Bucket1": {
            "Value": {
                "Fn::GetAtt": [
                    "Bucket1",
                    "Arn"
                ]
            }
        },
        "Bucket2": {
            "Value": {
                "Fn::GetAtt": [
                    "Bucket2",
                    "Arn"
                ]
            }
        }
    },
    "Description": "An example template for testing rain fmt command.\n",
    "Resources": {
        "Bucket2": {
            "Properties": {
                "BucketName": {
                    "Ref": "Name"
                }
            },
            "Type": "AWS::S3::Bucket"
        },
        "Bucket1": {
            "Type": "AWS::S3::Bucket",
            "Properties": {
                "BucketName": {
                    "Fn::Sub": "${Bucket2}-newer"
                }
            }
        },
        "Func1": {
            "Type": "AWS::Lambda::Function",
            "Properties": {
                "Role": {
                    "Fn::Sub": "arn:aws:iam::${AWS::AccountID}:role/lambda-basic"
                },
                "Runtime": "python3.7",
                "Handler": "index.handler",
                "Code": {
                    "ZipFile": "import boto3\n\ndef handler: \n  \"\"\"Example.\"\"\"\n\n  print('hello')\n"
                }
            }
        },
        "Instance1": {
            "Type": "AWS::EC2::Instance",
            "Properties": {
                "UserData": {
                    "Fn::Base64": {
                        "Fn::Sub": [
                            "#!/bin/bash -xe\napt-get update\n\napt-get upgrade -y\n"
                        ]
                    }
                }
            }
        }
    },
    "Rules": {
        "Rule1": {
            "RuleCondition": {
                "Fn::Equals": [
                    {
                        "Ref": "Environment"
                    },
                    "test"
                ]
            },
            "Assertions": [
                {
                    "Assert": {
                        "Fn::Contains": [
                            [
                                "a1.medium"
                            ],
                            {
                                "Ref": "InstanceType"
                            }
                        ]
                    }
                }
            ]
        }
    },
    "Parameters": {
        "Name": {
            "Type": "String"
        }
    }
}
`

const correctMultilineBlockHeaders = `
|+
|2+
|+2
|-
|2-
|-2
>+
>2+
>+2
>-
>2-
>-2
|10+
|+10
>10+
>+10
`

const wrongMultilineBlockHeaders = `
|2+2
>2+2
|+2+
>+2+
|++
>++
>2+ a
`

func checkMatch(t *testing.T, expected string, opt format.Options) {
	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, opt)

	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf("%s", d)
	}
}

func checkMultilineBlockHeaders(t *testing.T, s string, expected bool) {
	parts := strings.Split(s, "\n")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if format.CheckMultilineBegin(part) != expected {
			t.Errorf("%s", part)
		}
	}
}

func TestFormatYaml(t *testing.T) {
	checkMatch(t, expectedYaml, format.Options{})
}

func TestFormatYamlUnsorted(t *testing.T) {
	checkMatch(t, expectedYamlUnsorted, format.Options{
		Unsorted: true,
	})
}

func TestFormatJson(t *testing.T) {
	checkMatch(t, expectedJson, format.Options{
		JSON: true,
	})
}

func TestFormatUnsortedJson(t *testing.T) {
	checkMatch(t, expectedUnsortedJson, format.Options{
		JSON:     true,
		Unsorted: true,
	})
}

func TestFormatMultiLineBlock(t *testing.T) {
	checkMultilineBlockHeaders(t, correctMultilineBlockHeaders, true)
}

func TestFormatMultiLineBlockWrong(t *testing.T) {
	checkMultilineBlockHeaders(t, wrongMultilineBlockHeaders, false)
}

func TestFindInMap(t *testing.T) {
	yaml := `
AWSTemplateFormatVersion: '2010-09-09'

Description: Reproduce "semantic difference after formatting" error

Parameters:
  EnvironmentParam:
    Default: dev
    Type: String
    AllowedValues:
      - dev
      - prod

Mappings:
  EnvironmentMap:
    MappedParam:
      dev: my-dev-topic
      prod: my-prod-topic

Resources:
  Topic:
    Type: AWS::SNS::Topic
    Properties: 
      TopicName: !FindInMap [EnvironmentMap, MappedParam, !Ref EnvironmentParam]
`

	expect := `AWSTemplateFormatVersion: "2010-09-09"

Description: Reproduce "semantic difference after formatting" error

Parameters:
  EnvironmentParam:
    Default: dev
    Type: String
    AllowedValues:
      - dev
      - prod

Mappings:
  EnvironmentMap:
    MappedParam:
      dev: my-dev-topic
      prod: my-prod-topic

Resources:
  Topic:
    Type: AWS::SNS::Topic
    Properties:
      TopicName: !FindInMap
        - EnvironmentMap
        - MappedParam
        - !Ref EnvironmentParam
`

	template, err := parse.String(yaml)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, format.Options{
		Unsorted: true,
	})

	if d := cmp.Diff(expect, actual); d != "" {
		t.Fatal(d)
	}
}

func TestZipLines(t *testing.T) {
	yaml := `
  AWSTemplateFormatVersion: "2010-09-09"

  Description: Example AWS CloudFormation template snippet.
  
  Resources:
    Test:
      Type: AWS::Lambda::Function
      Properties:
        Role: arn:aws:iam::755952356119:role/lambda-basic
        Runtime: python3.7
        Handler: index.handler
        Code:
          ZipFile: |
            import boto3
  
            def handler: 
  
              """Example."""
  
              print('hello')
   
  `

	expect := `AWSTemplateFormatVersion: "2010-09-09"

Description: Example AWS CloudFormation template snippet.

Resources:
  Test:
    Type: AWS::Lambda::Function
    Properties:
      Role: arn:aws:iam::755952356119:role/lambda-basic
      Runtime: python3.7
      Handler: index.handler
      Code:
        ZipFile: "import boto3\n\ndef handler: \n\n  \"\"\"Example.\"\"\"\n\n  print('hello')\n"
`

	template, err := parse.String(yaml)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, format.Options{
		Unsorted: true,
	})

	if d := cmp.Diff(expect, actual); d != "" {
		t.Fatalf("%s", d)
	}
}

func TestMultiWithGT(t *testing.T) {
	yaml := `
AWSTemplateFormatVersion: "2010-09-09"

Description: Example AWS CloudFormation template snippet.

Resources:
  Test:
    Type: AWS::Lambda::Function
    Properties:
      Code:
        ZipFile: |
          """Example."""

          import boto3

          breaks = """
          >
          """

          TEST = 1
  `

	expect := `AWSTemplateFormatVersion: "2010-09-09"

Description: Example AWS CloudFormation template snippet.

Resources:
  Test:
    Type: AWS::Lambda::Function
    Properties:
      Code:
        ZipFile: |
          """Example."""

          import boto3

          breaks = """
          >
          """

          TEST = 1
`

	template, err := parse.String(yaml)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, format.Options{
		Unsorted: true,
	})

	if d := cmp.Diff(expect, actual); d != "" {
		t.Fatalf("%s", d)
	}
}

func TestToJson(t *testing.T) {
	s := "Test<String>"
	j, err := format.ToJson(s, "    ")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("\"%s\"", s) != string(j) {
		t.Fatalf("j is \"%s\", expected \"%s\"", j, s)
	}

	m := make(map[string]string, 0)
	m["Type"] = "AWS::SSM::Parameter::Value<String>"
	j, err = format.ToJson(m, "")
	if err != nil {
		t.Fatal(err)
	}
	expected := "{\"Type\":\"AWS::SSM::Parameter::Value<String>\"}"
	if expected != string(j) {
		t.Fatalf("j is \"%s\", expected \"%s\"", j, expected)
	}
}

func TestUnicodeJson(t *testing.T) {
	input := `
Parameters:
  pBotToken:
    Description: Bot Token
    Type: AWS::SSM::Parameter::Value<String>
    Default: /token/bot
    NoEcho: "true"
`

	// Parse the template
	source, err := parse.String(string(input))
	if err != nil {
		t.Fatal(err)
	}

	output := format.String(source, format.Options{
		JSON:     true,
		Unsorted: true,
	})

	// Verify the output is valid
	if err = parse.Verify(source, output); err != nil {
		t.Fatal(err)
	}

	expected := `
{
    "Parameters": {
        "pBotToken": {
            "Description": "Bot Token",
            "Type": "AWS::SSM::Parameter::Value<String>",
            "Default": "/token/bot",
            "NoEcho": "true"
        }
    }
}
`

	if strings.TrimSpace(output) != strings.TrimSpace(expected) {
		t.Fatalf("Got:\n%s\n\nExpected:\n%s", output, expected)
	}

}

func TestFnGetAZs(t *testing.T) {
	input := `
Resources:
  PrivateSubnet1Subnet:
    Type: AWS::EC2::Subnet
    Properties:
      AvailabilityZone: !Select
        - 0
        - !GetAZs ""
`

	// Parse the template
	source, err := parse.String(string(input))
	if err != nil {
		t.Fatal(err)
	}

	output := format.String(source, format.Options{
		JSON:     false,
		Unsorted: true,
	})

	// Verify the output is valid
	if err = parse.Verify(source, output); err != nil {
		t.Fatal(err)
	}

	expected := `
Resources:
  PrivateSubnet1Subnet:
    Type: AWS::EC2::Subnet
    Properties:
      AvailabilityZone: !Select
        - 0
        - !GetAZs 
`

	if strings.TrimSpace(output) != strings.TrimSpace(expected) {
		t.Fatalf("Got:\n%s\n\nExpected:\n%s", output, expected)
	}

}

func TestPkl(t *testing.T) {
	source := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
Outputs:
  BucketName:
    Value: !Ref Bucket
`
	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	output, err := format.CftToPkl(template, false, "@cfn")
	if err != nil {
		t.Fatal(err)
	}

	expected := `amends "@cfn/template.pkl"
import "@cfn/cloudformation.pkl" as cfn
import "@cfn/aws/s3/bucket.pkl"

Resources {
    ["Bucket"] = new bucket.Bucket {
    }

}

Outputs {
    ["BucketName"] = new cfn.Output {
        Value = cfn.Ref("Bucket")
    }
}
`
	if output != expected {
		t.Fatalf("Got:\n[%s]\nExpected:\n[%s]\n", output, expected)
	}
}

func TestAmbiguousScalars_StrictBooleans_On(t *testing.T) {
	// Save the original style
	originalStyle := format.NodeStyle
	// Restore after test completes
	defer func() {
		format.NodeStyle = originalStyle
	}()

	// Override for this test only
	format.NodeStyle = "strict-booleans"

	input := `
Param1:
  Default: "ON"
Param2:
  Default: "OFF"
Param3:
  Default: "Yes"
Param4:
  Default: "No"
Param5:
  Default: "True"
Param6:
  Default: "False"
Param7:
  Default: Maybe
Param8:
  Default: ON
Param9:
  Default: OFF
Param10:
  Default: Yes
Param11:
  Default: No
Param12:
  Default: True
Param13:
  Default: False
`
	// The ambiguous values (Param1 through Param6) should be rendered with quotes,
	// while a non-ambiguous value (Param7) may be unquoted.
	expected := `
Param1:
  Default: "ON"

Param2:
  Default: "OFF"

Param3:
  Default: "Yes"

Param4:
  Default: "No"

Param5:
  Default: "True"

Param6:
  Default: "False"

Param7:
  Default: Maybe

Param8:
  Default: "ON"

Param9:
  Default: "OFF"

Param10:
  Default: "Yes"

Param11:
  Default: "No"

Param12:
  Default: True

Param13:
  Default: False
`
	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, format.Options{Unsorted: true})
	if d := cmp.Diff(strings.TrimSpace(expected), strings.TrimSpace(actual)); d != "" {
		t.Fatalf("Diff: %s", d)
	}
}

func TestAmbiguousScalars_StrictBooleans_Off(t *testing.T) {
	input := `
Param1:
  Default: "ON"
Param2:
  Default: "OFF"
Param3:
  Default: "Yes"
Param4:
  Default: "No"
Param5:
  Default: "True"
Param6:
  Default: "False"
Param7:
  Default: Maybe
Param8:
  Default: ON
Param9:
  Default: OFF
Param10:
  Default: Yes
Param11:
  Default: No
Param12:
  Default: True
Param13:
  Default: False
`
	// The ambiguous values (Param1 through Param6) should be rendered with quotes,
	// while a non-ambiguous value (Param7) may be unquoted.
	expected := `
Param1:
  Default: ON

Param2:
  Default: OFF

Param3:
  Default: Yes

Param4:
  Default: No

Param5:
  Default: "True"

Param6:
  Default: "False"

Param7:
  Default: Maybe

Param8:
  Default: ON

Param9:
  Default: OFF

Param10:
  Default: Yes

Param11:
  Default: No

Param12:
  Default: True

Param13:
  Default: False
`
	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual := format.String(template, format.Options{Unsorted: true})
	if d := cmp.Diff(strings.TrimSpace(expected), strings.TrimSpace(actual)); d != "" {
		t.Fatalf("Diff: %s", d)
	}
}
