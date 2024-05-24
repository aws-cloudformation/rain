package ssm

import (
	"context"
	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func getClient() *ssm.Client {
	return ssm.NewFromConfig(rainaws.Config())
}

// GetParameter returns the value of the specified parameter.
func GetParameter(name string) (string, error) {
	client := getClient()
	parameter, err := client.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name: &name,
	})
	if err != nil {
		return "", err
	}

	return *parameter.Parameter.Value, nil
}
