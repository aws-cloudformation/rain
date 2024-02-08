package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// Check everything that could go wrong with an AWS::S3::Bucket resource.
// Returns numFailed, numChecked
func checkS3BucketPolicy(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	spin(input.typeName, input.logicalId, "bucket policy")

	if input.stackExists {
		_, err := cfn.GetStackResource(input.stackName, input.logicalId)

		// Do we need this?

		if err != nil {
			// Likely the resource has been added after the stack was created
			config.Debugf("Unable to get stack resource %v: %v", input.logicalId, err)
		}
	}

	// Go back to the template to get the referenced bucket

	// Check the policy for invalid principals
	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props != nil {
		_, policyDocument, _ := s11n.GetMapValue(props, "PolicyDocument")
		if policyDocument != nil {
			res, err := iam.CheckPolicyDocument(policyDocument)

			if err != nil {
				forecast.Add(false, fmt.Sprintf("Unable to check policy document: %v", err))
			}

			if !res {
				LineNumber = policyDocument.Line
				forecast.Add(false, "Invalid principal in policy document")
			} else {
				forecast.Add(true, "Principal is valid")
			}
		}
	}

	spinner.Pop()

	return forecast
}
