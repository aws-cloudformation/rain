package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context,
	request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("request: %+v\n", request)
	message := fmt.Sprintf("{\"message\": \"Request Resource: %s, Path: %s, HTTPMethod: %s\"}", request.Resource, request.Path, request.HTTPMethod)
	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       message,
	}

	return response, nil
}

func main() {
	lambda.Start(HandleRequest)
}
