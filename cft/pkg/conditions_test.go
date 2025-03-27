package pkg

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

func TestProcessConditions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name: "simple equals condition",
			template: `
Parameters:
  Environment:
    Type: String
    Default: prod
Conditions:
  IsProd:
    Fn::Equals:
      - !Ref Environment
      - prod
  NotProd:
    Fn::Not:
      - Fn::Equals:
        - !Ref Environment
        - prod
Resources:
  ProdBucket:
    Type: AWS::S3::Bucket
    Condition: IsProd
  DevBucket:
    Type: AWS::S3::Bucket
    Condition: NotProd 
`,
			want: `
Parameters:
  Environment:
    Type: String
    Default: prod
Resources:
  ProdBucket:
    Type: AWS::S3::Bucket
`,
		},
		{
			name: "and condition with modules",
			template: `
Parameters:
  Environment:
    Type: String
    Default: prod
  Region:
    Type: String
    Default: us-east-1
Conditions:
  IsProdAndEast:
    Fn::And:
      - !Equals [!Ref Environment, prod]
      - !Equals [!Ref Region, us-east-1]
Resources:
  ProdEastBucket:
    Type: AWS::S3::Bucket
    Condition: IsProdAndEast
Modules:
  ProdModule:
    Source: ./prod-module.yaml
    Condition: IsProdAndEast
`,
			want: `
Parameters:
  Environment:
    Type: String
    Default: prod
  Region:
    Type: String
    Default: us-east-1
Resources:
  ProdEastBucket:
    Type: AWS::S3::Bucket
Modules:
  ProdModule:
    Source: ./prod-module.yaml
`,
		},
		{
			name: "or condition with fn::if",
			template: `
Parameters:
  Environment:
    Type: String
    Default: dev
  Region:
    Type: String
    Default: us-east-1
Conditions:
  IsProdOrEast:
    Fn::Or:
      - !Equals [!Ref Environment, prod]
      - !Equals [!Ref Region, us-east-1]
Resources:
  ConditionalBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !If [IsProdOrEast, prod-bucket, dev-bucket]
      Tags:
        - Key: Environment
          Value: !If [IsProdOrEast, Production, Development]
`,
			want: `
Parameters:
  Environment:
    Type: String
    Default: dev
  Region:
    Type: String
    Default: us-east-1
Resources:
  ConditionalBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: prod-bucket
      Tags:
        - Key: Environment
          Value: Production
`,
		},
		{
			name: "nested conditions",
			template: `
Parameters:
  Environment:
    Type: String
    Default: prod
  Region:
    Type: String
    Default: us-east-1
  Feature:
    Type: String
    Default: enabled
Conditions:
  IsProd:
    Fn::Equals: [!Ref Environment, prod]
  IsEast:
    Fn::Equals: [!Ref Region, us-east-1]
  IsFeatureEnabled:
    Fn::Equals: [!Ref Feature, enabled]
  ShouldDeploy:
    Fn::And:
      - !Condition IsProd
      - Fn::Or:
        - !Condition IsEast
        - !Condition IsFeatureEnabled
Resources:
  ConditionalBucket:
    Type: AWS::S3::Bucket
    Condition: ShouldDeploy
`,
			want: `
Parameters:
  Environment:
    Type: String
    Default: prod
  Region:
    Type: String
    Default: us-east-1
  Feature:
    Type: String
    Default: enabled
Resources:
  ConditionalBucket:
    Type: AWS::S3::Bucket
`,
		},
		{
			name: "condition with no value",
			template: `
Conditions:
  AlwaysFalse:
    Fn::Equals: [true, false]
Resources:
  NeverCreated:
    Type: AWS::S3::Bucket
    Condition: AlwaysFalse
  AlwaysCreated:
    Type: AWS::S3::Bucket
`,
			want: `
Resources:
  AlwaysCreated:
    Type: AWS::S3::Bucket
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the template
			var templateNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.template), &templateNode)
			if err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			err = parse.NormalizeNode(&templateNode)
			if err != nil {
				t.Fatalf("failed to normalize template: %v", err)
			}

			// Create a Module instance
			m := &Module{
				Node: templateNode.Content[0],
			}

			m.Config = &cft.ModuleConfig{
				Name: "Test",
			}
			m.Parent = &cft.Template{Node: &templateNode}

			// Get the sections
			m.InitNodes()

			// Process conditions
			err = m.ProcessConditions()
			if err != nil {
				t.Fatalf("ProcessConditions() error = %v", err)
			}

			// Parse the expected result
			var wantNode yaml.Node
			err = yaml.Unmarshal([]byte(tt.want), &wantNode)
			if err != nil {
				t.Fatalf("failed to parse want template: %v", err)
			}

			err = parse.NormalizeNode(&wantNode)
			if err != nil {
				t.Fatalf("failed to normalize want template: %v", err)
			}

			// Compare the results
			got := node.YamlStr(m.Node)
			want := node.YamlStr(wantNode.Content[0])

			if got != want {
				t.Errorf("ProcessConditions() got = \n%v\n, want \n%v", got, want)
			}
		})
	}
}

func TestProcessConditionsErrors(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  string
	}{
		{
			name: "invalid condition reference",
			template: `
Conditions:
  Test:
    Condition: NonExistentCondition
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Condition: Test
`,
			wantErr: "referenced condition 'NonExistentCondition' not found",
		},
		{
			name: "invalid equals condition",
			template: `
Conditions:
  Test:
    Fn::Equals:
      - only one value
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Condition: Test
`,
			wantErr: "Fn::Equals requires exactly two values",
		},
		{
			name: "invalid and condition",
			template: `
Conditions:
  Test:
    Fn::And: true
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Condition: Test
`,
			wantErr: "Fn::And requires a list of conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the template
			var templateNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.template), &templateNode)
			if err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			err = parse.NormalizeNode(&templateNode)
			if err != nil {
				t.Fatalf("failed to normalize template: %v", err)
			}

			// Create a Module instance
			m := &Module{
				Node:   templateNode.Content[0],
				Config: &cft.ModuleConfig{Name: "Test"},
			}

			// Get the sections
			_, resources, _ := s11n.GetMapValue(m.Node, string(cft.Resources))
			m.ResourcesNode = resources

			_, conditions, _ := s11n.GetMapValue(m.Node, string(cft.Conditions))
			m.ConditionsNode = conditions

			// Process conditions
			err = m.ProcessConditions()
			if err == nil {
				t.Fatal("ProcessConditions() expected error")
			}

			if err.Error() != tt.wantErr {
				t.Errorf("ProcessConditions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
