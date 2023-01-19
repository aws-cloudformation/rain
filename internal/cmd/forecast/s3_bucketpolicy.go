package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
)

// Check everything that could go wrong with an AWS::S3::Bucket resource.
// Returns numFailed, numChecked
func checkBucketPolicy(input PredictionInput) (int, int) {

	numFailed := 0
	numChecked := 0

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
	for elementName, element := range input.resource.(map[string]interface{}) {
		config.Debugf("BucketPolicy element %v %v", elementName, element)

		if elementName == "Properties" {
			for propName, prop := range element.(map[string]interface{}) {

				if propName == "PolicyDocument" {
					res, err := iam.CheckPolicyDocument(prop)

					if err != nil {
						fmt.Printf("Unable to check policy document: %v", err)
						return 1, 1
					}

					numChecked += 1
					if !res {
						numFailed += 1
					}
				}
			}
		}
	}

	return numFailed, numChecked
}
