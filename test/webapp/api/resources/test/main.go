package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/apigateway"
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

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "{\"message\": \"Success\"}",
		Headers:    make(map[string]string),
	}

	apigateway.AddCORSHeaders(response)

	switch request.HTTPMethod {
	case "GET":
		input := &dynamodb.ScanInput{
			TableName: aws.String(os.Getenv("TABLE_NAME")),
		}
		res, err := client.Scan(context.Background(), input)
		if err != nil {
			fmt.Printf("Scan failed: %v\n", err)
			return apigateway.Fail(500, fmt.Sprintf("%v", err)), nil
		}
		fmt.Printf("Scan result: %+v", res)
		jsonData, err := json.Marshal(res.Items)
		if err != nil {
			fmt.Printf("Marshal failed: %v\n", err)
			return apigateway.Fail(500, fmt.Sprintf("%v", err)), nil
		}
		response.Body = string(jsonData)
	case "OPTIONS":
		response.StatusCode = 204
		response.Body = "{}"
		return response, nil
	default:
		return apigateway.Fail(400, fmt.Sprintf("Unexpected HttpMethod: %s", request.HTTPMethod)), nil
	}

	return response, nil
}

func main() {
	lambda.Start(HandleRequest)
}
