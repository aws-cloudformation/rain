package main

import (
	"context"
	"fmt"
	"os"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func HandleRequest(ctx context.Context,
	request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("request: %+v\n", request)
	message := fmt.Sprintf("{\"message\": \"Request Resource: %s, Path: %s, HTTPMethod: %s\"}", request.Resource, request.Path, request.HTTPMethod)
	fmt.Printf("message: %s\n", message)

	client := dynamodb.NewFromConfig(rainaws.Config())

	switch request.HTTPMethod {
	case "GET":
		input := &dynamodb.ScanInput{
			TableName: aws.String(os.Getenv("TABLE_NAME")),
		}
		res, err := client.Scan(context.Background(), input)
		if err != nil {
			fmt.Printf("Scan failed: %v\n", err)
			return fail500(fmt.Sprintf("%v", err)), nil
		}
		fmt.Printf("Scan result: %+v", res)
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "{\"message\": \"Success\"}",
	}

	return response, nil
}

func fail500(msg string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "{\"message\": \"" + msg + "\"}",
	}
}

func main() {
	lambda.Start(HandleRequest)
}
