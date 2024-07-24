package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// checkLambdaFunction checks for potential stack failures related to functions
func checkLambdaFunction(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("No Properties found for %s", input.logicalId)
		return forecast
	}
	_, roleProp, _ := s11n.GetMapValue(props, "Role")

	// If the role is specified, and it's a scalar, check if it exists
	if roleProp != nil && roleProp.Kind == yaml.ScalarNode {
		spin(input.typeName, input.logicalId, "Checking if lambda role exists")
		roleArn := roleProp.Value
		LineNumber = roleProp.Line
		if !iam.RoleExists(roleArn) {
			forecast.Add(F0016, false, "Role does not exist")
		} else {
			forecast.Add(F0016, true, "Role exists")
		}
		spinner.Pop()

		// Check to make sure the iam role can be assumed by the lambda function
		spin(input.typeName, input.logicalId, "Checking if lambda role can be assumed")
		canAssume, err := iam.CanAssumeRole(roleArn, "lambda.amazonaws.com")
		if err != nil {
			config.Debugf("Error checking role: %s", err)
		} else {
			if !canAssume {
				forecast.Add(F0017, false, "Role can not be assumed")
			} else {
				forecast.Add(F0017, true, "Role can be assumed")
			}
		}
		spinner.Pop()
	}

	// If the lambda function has an s3 bucket and key, make sure they exist
	_, codeProp, _ := s11n.GetMapValue(props, "Code")
	if codeProp != nil {
		_, s3Bucket, _ := s11n.GetMapValue(codeProp, "S3Bucket")
		_, s3Key, _ := s11n.GetMapValue(codeProp, "S3Key")
		if s3Bucket != nil && s3Key != nil {
			spin(input.typeName, input.logicalId,
				fmt.Sprintf("Checking to see if S3 object %s/%s exists",
					s3Bucket.Value, s3Key.Value))

			// See if the bucket exists
			exists, err := s3.BucketExists(s3Bucket.Value)
			if err != nil {
				config.Debugf("Unable to check if S3 bucket exists: %v", err)
			}
			if !exists {
				forecast.Add(F0019, false, "S3 bucket does not exist")
			} else {
				forecast.Add(F0019, true, "S3 bucket exists")

				// If the bucket exists, check to see if the object exists
				obj, err := s3.GetObject(s3Bucket.Value, s3Key.Value)
				if err != nil || obj == nil {
					forecast.Add(F0020, false, "S3 object does not exist")
				} else {
					forecast.Add(F0020, true, "S3 object exists")
				}
			}

			spinner.Pop()
		} else {
			config.Debugf("%s does not have S3Bucket and S3Key", input.logicalId)
		}
	} else {
		config.Debugf("Unexpected missing Code property from %s", input.logicalId)
	}

	return forecast
}
