package ssm

import (
	"context"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
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

// SetParameter sets the value of a parameter and overwrites a pervious value
func SetParameter(name string, value string) error {
	client := getClient()
	resp, err := client.PutParameter(context.Background(), &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      types.ParameterTypeString,
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		config.Debugf("resp: %+v", resp)
		return err
	}

	return nil
}
