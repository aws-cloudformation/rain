package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
)

// Check everything that could go wrong with an AWS::S3::Bucket resource.
// Returns numFailed, numChecked
func checkBucketPolicy(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

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
	// TODO: switch to yaml nodes so we retain the line number
	for elementName, element := range input.resource.(map[string]interface{}) {
		config.Debugf("BucketPolicy element %v %v", elementName, element)

		if elementName == "Properties" {
			for propName, prop := range element.(map[string]interface{}) {

				if propName == "PolicyDocument" {
					res, err := iam.CheckPolicyDocument(prop)

					if err != nil {
						forecast.Add(false, fmt.Sprintf("Unable to check policy document: %v", err))
					}

					if !res {
						forecast.Add(false, "Invalid principal in policy document")
					} else {
						forecast.Add(true, "Principal is valid")
					}
				}
			}
		}
	}

	return forecast
}
