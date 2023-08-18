package simplify_test

import (
	"github.com/aws-cloudformation/rain/cft/simplify"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

const input = `
AWSTemplateFormatVersion: 2010-09-09
Transform: 'AWS::LanguageExtensions'
Resources:
  DynamoDBTable1:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table1
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable2:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table2
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable3:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table3
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable4:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table4
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable5:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table5
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable6:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table6
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable7:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table7
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable8:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table8
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable9:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table9
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
  DynamoDBTable10:
    Type: 'AWS::DynamoDB::Table'
    Properties:
      TableName: Table10
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: '5'
        WriteCapacityUnits: '5'
`

const expectedForEach = `AWSTemplateFormatVersion: "2010-09-09"

Transform: AWS::LanguageExtensions

Resources:
  Fn::ForEach::Loop0:
    - Variable0
    - - Table1
      - Table10
      - Table2
      - Table3
      - Table4
      - Table5
      - Table6
      - Table7
      - Table8
      - Table9
    - Resource${Variable0}:
        Properties:
          AttributeDefinitions:
            - AttributeName: id
              AttributeType: S
          KeySchema:
            - AttributeName: id
              KeyType: HASH
          ProvisionedThroughput:
            ReadCapacityUnits: "5"
            WriteCapacityUnits: "5"
          TableName: !Ref Variable0
        Type: AWS::DynamoDB::Table
`

func checkMatch(t *testing.T, expected string, opt simplify.Options) {
	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual := simplify.String(template, opt)

	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf(d)
	}
}

func TestFormatDefault(t *testing.T) {
	checkMatch(t, expectedForEach, simplify.Options{
		ForEach: true,
	})
}
