package format_test

import (
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
      Role: !Sub "arn:aws:iam::${AWS::AccountID}:role/lambda-basic"
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
      Role: !Sub "arn:aws:iam::${AWS::AccountID}:role/lambda-basic"
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
		t.Errorf(d)
	}
}

func checkMultilineBlockHeaders(t *testing.T, s string, expected bool) {
	parts := strings.Split(s, "\n")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if format.CheckMultilineBegin(part) != expected {
			t.Errorf(part)
		}
	}
}

func TestFormatDefault(t *testing.T) {
	checkMatch(t, expectedYaml, format.Options{})
	checkMatch(t, expectedYamlUnsorted, format.Options{
		Unsorted: true,
	})
	checkMatch(t, expectedJson, format.Options{
		JSON: true,
	})
	checkMatch(t, expectedUnsortedJson, format.Options{
		JSON:     true,
		Unsorted: true,
	})
	checkMultilineBlockHeaders(t, correctMultilineBlockHeaders, true)
	checkMultilineBlockHeaders(t, wrongMultilineBlockHeaders, false)
}
