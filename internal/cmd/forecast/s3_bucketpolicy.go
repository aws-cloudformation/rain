package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

// Check everything that could go wrong with an AWS::S3::Bucket resource.
// Returns numFailed, numChecked
func CheckS3BucketPolicy(input fc.PredictionInput) fc.Forecast {

	forecast := fc.MakeForecast(&input)

	spin(input.TypeName, input.LogicalId, "bucket policy")

	if input.StackExists {
		_, err := cfn.GetStackResource(input.StackName, input.LogicalId)

		// Do we need this?

		if err != nil {
			// Likely the resource has been added after the stack was created
			config.Debugf("Unable to get stack resource %v: %v", input.LogicalId, err)
		}
	}

	// Go back to the template to get the referenced bucket

	// Check the policy for invalid principals
	_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
	if props != nil {
		_, policyDocument, _ := s11n.GetMapValue(props, "PolicyDocument")
		if policyDocument != nil {
			res, err := iam.CheckPolicyDocument(policyDocument)

			code := F0002

			if err != nil {
				forecast.Add(code,
					false, fmt.Sprintf("Unable to check policy document: %v", err),
					getLineNum(input.LogicalId, input.Resource))
			}

			if !res {
				forecast.Add(code, false, "Invalid principal in policy document",
					getLineNum(input.LogicalId, input.Resource))
			} else {
				forecast.Add(code, true, "Principal is valid",
					getLineNum(input.LogicalId, input.Resource))
			}
		}
	}

	spinner.Pop()

	return forecast
}
