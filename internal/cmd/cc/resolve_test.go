package cc

import (
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"
)

// Test to make sure we can resolve Refs to Parameters
func TestResolveRefParam(t *testing.T) {
	source := `
Parameters:
    A:
        Type: String
    Missing:
        Type: String
        Default: a
Resources:
    B:
        Type: AWS::S3::Bucket
        Properties:
            BucketName: 
                Ref: A
    C:
        Type: AWS::S3::Bucket
        Properties:
            BucketName: !Ref Missing
`
	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	testParams = append(testParams, "A=aaa")
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	resourceNode, err := template.GetResource("B")
	if err != nil {
		t.Fatal(err)
	}

	resource := NewResource("B", "AWS::S3::Bucket", Waiting, resourceNode)

	resolved, err := Resolve(resource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved node B: %v", node.ToSJson(resolved))

	// Make sure the value is what we expect
	_, props := s11n.GetMapValue(resolved, "Properties")
	if props == nil {
		t.Fatalf("B Properties is missing")
	}
	_, bucketName := s11n.GetMapValue(props, "BucketName")
	if bucketName == nil {
		t.Fatalf("B Properties BucketName is missing")
	}
	if bucketName.Value != "aaa" {
		t.Fatalf("Expected BucketName for B to be aaa, got %v", bucketName.Value)
	}

	// Check a missing parameter to make sure the default is applied
	resourceNode, err = template.GetResource("C")
	if err != nil {
		t.Fatal(err)
	}

	resource = NewResource("C", "AWS::S3::Bucket", Waiting, resourceNode)

	resolved, err = Resolve(resource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved node C: %v", node.ToSJson(resolved))

	_, props = s11n.GetMapValue(resolved, "Properties")
	if props == nil {
		t.Fatalf("C Properties is missing")
	}
	_, bucketName = s11n.GetMapValue(props, "BucketName")
	if bucketName == nil {
		t.Fatalf("C Properties BucketName is missing")
	}
	if bucketName.Value != "a" {
		t.Fatalf("Expected BucketName for C to be a, got %v", bucketName.Value)
	}
}

// Test to make sure we can resolve Refs to Resources
func TestResolveRefResource(t *testing.T) {
	source := `
Resources:
    B:
        Type: AWS::S3::Bucket
        Properties:
            BucketName: mybucket
    C:
        Type: AWS::S3::Bucket
        Properties:
            LoggingConfiguration:
                DestinationBucketName: !Ref B
`
	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	resourceNode, err := template.GetResource("C")
	if err != nil {
		t.Fatal(err)
	}

	resource := NewResource("C", "AWS::S3::Bucket", Waiting, resourceNode)

	// Put B into the resource map, as if we had deployed it
	bNode, _ := deployedTemplate.GetResource("B")
	bResource := NewResource("B", "AWS::S3::Bucket", Waiting, bNode)
	bResource.Identifier = "bname"
	bResource.Model = `
{
	"BucketName": "bname"	
}
`
	resMap["B"] = bResource

	resolved, err := Resolve(resource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved node C: %v", node.ToSJson(resolved))

	// Make sure the value is what we expect
	gotVal := resolved.Content[3].Content[1].Content[1].Value
	if gotVal != "bname" {
		t.Fatalf("Expected DestinationBucketName to be bname, got %s", gotVal)
	}
}

// Test to make sure we can resolve GetAtts
func TestResolveGetAtt(t *testing.T) {
	source := `
Resources:
    MyFunc:
        Type: AWS::Lambda::Function
        Properties:
            Role: 
                Fn::GetAtt: [ MyBucket, Arn ]
    MyFunc2:
        Type: AWS::Lambda::Function
        Properties:
            Role: !GetAtt MyBucket.Arn
    MyBucket:
        Type: AWS::S3::Bucket
`

	// Note that MyFunc and MyFunc2 look the same in the parsed YAML

	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	logicalId := "MyBucket"
	bNode, err := template.GetResource(logicalId)
	if err != nil {
		t.Fatal(err)
	}

	// Put it into the resource map, as if we had deployed it
	bResource := NewResource(logicalId, "AWS::S3::Bucket", Waiting, bNode)
	bResource.Identifier = "bname"
	arn := "arn:aws:s3:::bname"
	bResource.Model = `
{
	"BucketName": "bname",
	"Arn": "ARN" 
}
`
	bResource.Model = strings.Replace(bResource.Model, "ARN", arn, -1)
	config.Debugf("bResource.Model: %s", bResource.Model)
	resMap[logicalId] = bResource

	myfuncNode, _ := template.GetResource("MyFunc")
	myfunc := NewResource("MyFunc", "AWS::Lambda::Function", Waiting, myfuncNode)

	resolved, err := Resolve(myfunc)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved MyFunc node: %v", node.ToSJson(resolved))

	// Make sure the value is what we expect
	if resolved.Content[3].Content[1].Kind != yaml.ScalarNode {
		t.Fatalf("Expected resolved Arn to be a scalar")
	}
	gotVal := resolved.Content[3].Content[1].Value
	if gotVal != arn {
		t.Fatalf("Expected %s but got %s", arn, gotVal)
	}
}

// Test to make sure we can resolve Subs
func TestResolveSub(t *testing.T) {
	source := `
Resources:
    MyFunc:
        Type: AWS::Lambda::Function
        Properties:
            Role: 
                Fn::Sub: "${MyBucket.Arn}"
    MyFunc2:
        Type: AWS::Lambda::Function
        Properties:
            Role: !Sub "${MyBucket.Arn}"
    MyBucket:
        Type: AWS::S3::Bucket
        Properties:
            BucketName: 
                Fn::Sub:
                  - "Test${A}"
                  - A: 1
`

	// Note that MyFunc and MyFunc2 look the same in the parsed YAML

	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	logicalId := "MyBucket"
	bNode, err := template.GetResource(logicalId)
	if err != nil {
		t.Fatal(err)
	}

	// Put it into the resource map, as if we had deployed it
	bResource := NewResource(logicalId, "AWS::S3::Bucket", Waiting, bNode)
	bResource.Identifier = "Test1"
	arn := "arn:aws:s3:::Test"
	bResource.Model = `
{
	"BucketName": "Test1",
	"Arn": "ARN" 
}
`
	bResource.Model = strings.Replace(bResource.Model, "ARN", arn, -1)
	config.Debugf("bResource.Model: %s", bResource.Model)
	resMap[logicalId] = bResource

	myfuncNode, _ := template.GetResource("MyFunc")
	myfunc := NewResource("MyFunc", "AWS::Lambda::Function", Waiting, myfuncNode)

	resolved, err := Resolve(myfunc)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved MyFunc node: %v", node.ToSJson(resolved))

	// Make sure the value is what we expect
	if resolved.Content[3].Content[1].Kind != yaml.ScalarNode {
		t.Fatalf("Expected resolved Arn to be a scalar")
	}
	gotVal := resolved.Content[3].Content[1].Value
	if gotVal != arn {
		t.Fatalf("Expected %s but got %s", arn, gotVal)
	}

	// Now resolve the Sub with a Sequence in MyBucket
	resolved, err = Resolve(bResource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved MyBucket node: %v", node.ToSJson(resolved))

	gotVal = resolved.Content[3].Content[1].Value
	if gotVal != "Test1" {
		t.Fatalf("Got %s for MyBucket Name, expected Test1", gotVal)
	}
}

// Test to make sure we can resolve Refs to pseudo params (AWS::X)
// Commenting this out since it should actually be an integ test.
// It has to make network calls to get aws config.
// TODO: Move to integ test
/*
func TestResolvePseudo(t *testing.T) {
	source := `
Resources:
    A:
        Type: A::B::C
        Properties:
            AccountId: !Sub "${AWS::AccountId}"
            Region: !Sub "${AWS::Region}"
            Partition: !Sub "${AWS::Partition}"
`

	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	logicalName := "A"

	resourceNode, err := template.GetResource(logicalName)
	if err != nil {
		t.Fatal(err)
	}

	resource := NewResource(logicalName, "AWS::S3::Bucket", Waiting, resourceNode)

	resolved, err := Resolve(resource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved node : %v", node.ToSJson(resolved))

	// TODO: Verify values
}
*/

// Test to make sure we can resolve Sequences
func TestResolveSequence(t *testing.T) {
	source := `
Resources:
  Log:
    Type: AWS::Logs::LogGroup
    Properties:
      LogGroupName: test
  Query1:
    Type: AWS::Logs::QueryDefinition
    Properties:
      Name: Query1
      QueryString: "fields @message"
      LogGroupNames:
      - !Sub "${Log.LogGroupName}"
      - !Ref Log 
`
	template, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	config.Debug = true
	config.Debugf("template: %v", node.ToSJson(template.Node))

	// Set globals
	deployedTemplate = template
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	testParams := make([]string, 0)
	testTags := make([]string, 0)
	dc, err := dc.GetDeployConfig(testTags, testParams, "", "",
		template, stack, false, true, false)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	logicalName := "Query1"

	resourceNode, err := template.GetResource(logicalName)
	if err != nil {
		t.Fatal(err)
	}

	resource := NewResource(logicalName, "AWS::Logs::QueryDefinition", Waiting, resourceNode)

	// Put Log into the resource map, as if we had deployed it
	rName := "Log"
	rNode, _ := deployedTemplate.GetResource(rName)
	rResource := NewResource(rName, "AWS::Logs::LogGoup", Waiting, rNode)
	rResource.Identifier = "test"
	rResource.Model = `
{
	"LogGroupName": "test"	
}
`
	resMap[rName] = rResource

	resolved, err := Resolve(resource)
	if err != nil {
		t.Fatal(err)
	}

	config.Debugf("resolved node: %v", node.ToSJson(resolved))

	// Make sure the value is what we expect
}
