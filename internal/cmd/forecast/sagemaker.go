package forecast

import (
	_ "embed"
	"encoding/json"
	"math"

	"github.com/aws-cloudformation/rain/internal/aws/sagemaker"
	"github.com/aws-cloudformation/rain/internal/aws/servicequotas"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

//go:embed sagemaker-notebook-instance-codes.json
var notebookCodes string

// Created with this:
//
// aws service-quotas list-service-quotas --service-code sagemaker --no-cli-pager > ~/Desktop/sagemaker-quotas.json
//
// cat ~/Desktop/sagemaker-quotas.json| jq '[ .Quotas.[] | select(.QuotaName|test(".*notebook instance usage.*")) | { Code: .QuotaCode, Name: .QuotaName, InstanceType: .QuotaName|sub(" for notebook instance usage";"") } ]'

type NotebookCode struct {
	Code          string
	Name          string
	InstanceType  string
	InstanceCount int
}

func ParseNotebookCodes(jsonData string) ([]NotebookCode, error) {
	var retval []NotebookCode
	err := json.Unmarshal([]byte(jsonData), &retval)
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func CheckSageMakerNotebook(input fc.PredictionInput) fc.Forecast {
	// AWS::SageMaker::NotebookInstance

	forecast := makeForecast(input.TypeName, input.LogicalId)

	checkNotebookLimit(&input, &forecast)

	return forecast
}

func checkNotebookLimit(input *fc.PredictionInput, forecast *fc.Forecast) {

	// The account- service limit 'Total number of notebook instances' is 8
	// NotebookInstances, with current utilization of 16 NotebookInstances and
	// a request delta of 1 NotebookInstances. Please contact AWS support to
	// request an increase for this limit.

	spin(input.TypeName, input.LogicalId, "SageMaker notebook quota ok?")
	defer spinner.Pop()

	atLimit := false

	serviceCode := "sagemaker"

	// Total number of notebook instances
	quotaCode := "L-04CE2E67"

	// First check the overall limit
	quota, err := servicequotas.GetQuota(serviceCode, quotaCode)
	if err != nil {
		config.Debugf("Unable to get quota %s %s: %v", serviceCode, quotaCode, err)
		return
	}

	config.Debugf("Quota for %s %s: %v", serviceCode, quotaCode, quota)

	// Query the SageMaker API to check the current number of instances
	instances, err := sagemaker.GetNotebookInstances()
	if err != nil {
		config.Debugf("Unable to get notebook instances")
		return
	}

	config.Debugf("Instances: %+v", instances)

	if len(instances) >= int(math.Round(quota)) {
		atLimit = true
	} else {
		config.Debugf("Overall notebook quota ok with %v current instances", len(instances))

		// Then query the ListServiceQuotas API to get all codes for each individual
		// instance type, and check that specific quota.
		// We should probably cache this data, since it's a slow API call
		codes, err := ParseNotebookCodes(notebookCodes)
		if err != nil {
			config.Debugf("Unable to parse notebook codes: %v", err)
			return
		}

		codeMap := make(map[string]*NotebookCode, 0)
		for _, code := range codes {
			codeMap[code.InstanceType] = &code
		}

		config.Debugf("codeMap: %+v", codeMap)

		for _, inst := range instances {
			if code, ok := codeMap[inst.InstanceType]; ok {
				code.InstanceCount += 1
			}
		}

		// Get the instance type from the resource we are checking
		var resourceInstanceType string
		_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
		if props == nil {
			config.Debugf("expected %s to have Properties", input.LogicalId)
			return
		}
		_, instanceTypeNode, _ := s11n.GetMapValue(props, "InstanceType")
		if instanceTypeNode == nil {
			config.Debugf("expected %s to have InstanceType", input.LogicalId)
			return
		}
		resourceInstanceType = instanceTypeNode.Value
		config.Debugf("%s InstanceType is %s", input.LogicalId, resourceInstanceType)

		code, ok := codeMap[resourceInstanceType]
		if !ok {
			config.Debugf("InstanceType %s missing from codeMap", resourceInstanceType)
			return
		}
		config.Debugf("code: %+v", code)

		quota, err := servicequotas.GetQuota(serviceCode, code.Code)
		if err != nil {
			config.Debugf("Unable to get quota %s %s: %v", serviceCode, quotaCode, err)
			return
		}
		config.Debugf("Quota for %s (%s) is %v. Current count is %v",
			code.InstanceType, code.Code, quota, code.InstanceCount)

		if code.InstanceCount >= int(math.Round(quota)) {
			atLimit = true
		}

	}

	forecastCode := F0018
	if !atLimit {
		forecast.Add(forecastCode, true, "Quota limit has not been reached")
	} else {
		forecast.Add(forecastCode, false, "Over quota limit")
	}

}
