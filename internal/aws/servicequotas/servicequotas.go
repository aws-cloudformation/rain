package servicequotas

import (
	"context"

	aws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

func getClient() *servicequotas.Client {
	return servicequotas.NewFromConfig(aws.Config())
}

// Get the value for a service quota
func GetQuota(serviceCode string, quotaCode string) (float64, error) {

	res, err := getClient().GetServiceQuota(context.Background(),
		&servicequotas.GetServiceQuotaInput{
			QuotaCode:   &quotaCode,
			ServiceCode: &serviceCode,
		})
	if err != nil {
		return -1, nil
	}
	return *res.Quota.Value, nil
}
