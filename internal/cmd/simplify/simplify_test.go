package simplify_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/simplify"
)

func Example_simplify_foreach() {
	os.Args = []string{
		os.Args[0],
		"--foreach",
		"../../../test/templates/simplifyforeach.template",
	}

	simplify.Cmd.Execute()
	// Output:
	// AWSTemplateFormatVersion: "2010-09-09"
	//
	// Transform: AWS::LanguageExtensions
	//
	// Resources:
	//   Fn::ForEach::Loop0:
	//     - Variable0
	//     - - Table8
	//       - Table1
	//       - Table3
	//       - Table5
	//       - Table6
	//       - Table7
	//       - Table9
	//       - Table10
	//       - Table2
	//       - Table4
	//     - Resource${Variable0}:
	//         Properties:
	//           AttributeDefinitions:
	//             - AttributeName: id
	//               AttributeType: S
	//           KeySchema:
	//             - AttributeName: id
	//               KeyType: HASH
	//           ProvisionedThroughput:
	//             ReadCapacityUnits: "5"
	//             WriteCapacityUnits: "5"
	//           TableName: !Ref TableName
	//         Type: 'AWS::DynamoDB::Variable0'
}
