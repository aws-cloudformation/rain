package graph_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/cft/parse"
)

const templateString = `
Parameters:
  Name:
    Type: String
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref Name
      Tags:
        - Key: Account
        - Value: !Ref AWS::AccountId
Outputs:
  BucketName:
    Value: !Ref Bucket
  BucketArn:
    Value: !GetAtt Bucket.Arn
`

var template cft.Template

var g graph.Graph

func TestMain(m *testing.M) {
	var err error
	template, err = parse.String(templateString)
	if err != nil {
		panic(err)
	}

	g = graph.New(template)

	os.Exit(m.Run())
}

func Example_nodes() {
	for _, node := range g.Nodes() {
		fmt.Println(node)
	}
	// Output:
	// Parameters/AWS::AccountId
	// Parameters/Name
	// Resources/Bucket
	// Outputs/BucketArn
	// Outputs/BucketName
}

func Example_get() {
	fmt.Println(g.Get(graph.Node{"Parameters", "Name"}))
	fmt.Println(g.Get(graph.Node{"Resources", "Bucket"}))
	fmt.Println(g.Get(graph.Node{"Outputs", "BucketName"}))
	// Output:
	// []
	// [Parameters/AWS::AccountId Parameters/Name]
	// [Resources/Bucket]
}

func Example_getReverse() {
	fmt.Println(g.GetReverse(graph.Node{"Parameters", "Name"}))
	fmt.Println(g.GetReverse(graph.Node{"Resources", "Bucket"}))
	fmt.Println(g.GetReverse(graph.Node{"Outputs", "BucketName"}))
	// Output:
	// [Resources/Bucket]
	// [Outputs/BucketArn Outputs/BucketName]
	// []
}
