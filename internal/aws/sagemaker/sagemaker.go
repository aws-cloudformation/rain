package sagemaker

import (
	"context"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
)

func getClient() *sagemaker.Client {
	return sagemaker.NewFromConfig(aws.Config())
}

type NotebookInstance struct {
	InstanceType string
}

func GetNotebookInstances() ([]NotebookInstance, error) {
	retval := make([]NotebookInstance, 0)

	client := getClient()
	var nextToken *string

	for {
		resp, err := client.ListNotebookInstances(context.Background(),
			&sagemaker.ListNotebookInstancesInput{
				NextToken: nextToken,
			})
		if err != nil {
			return retval, err
		}

		for _, inst := range resp.NotebookInstances {
			retval = append(retval, NotebookInstance{InstanceType: string(inst.InstanceType)})
		}

		if resp.NextToken != nil {
			nextToken = resp.NextToken
		} else {
			break
		}
	}

	return retval, nil
}
