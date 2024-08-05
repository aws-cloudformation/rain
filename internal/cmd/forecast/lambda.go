package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"gopkg.in/yaml.v3"
)

func checkLambdaRole(input *PredictionInput, forecast *Forecast) {

	roleProp := input.GetPropertyNode("Role")

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
}

func checkLambdaS3Bucket(input *PredictionInput, forecast *Forecast) {
	// If the lambda function has an s3 bucket and key, make sure they exist
	codeProp := input.GetPropertyNode("Code")
	if codeProp != nil {
		s3Bucket := GetNode(codeProp, "S3Bucket")
		s3Key := GetNode(codeProp, "S3Key")
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
				obj, err := s3.HeadObject(s3Bucket.Value, s3Key.Value)

				if err != nil || obj == nil {
					forecast.Add(F0020, false, "S3 object does not exist")
				} else {
					forecast.Add(F0020, true, "S3 object exists")

					config.Debugf("S3 Object %s/%s SizeBytes: %v",
						s3Bucket.Value, s3Key.Value, obj.SizeBytes)

					// Make sure it's less than 50Mb and greater than 0
					// We are not downloading it and unzipping to check total size,
					// since that would take too long for large files.
					var max int64 = 50 * 1024 * 1024
					if obj.SizeBytes > 0 && obj.SizeBytes <= max {

						if obj.SizeBytes < 256 {
							// This is suspiciously small. Download it and decompress
							// to see if it's a zero byte file. A simple "Hello" python
							// handler will zip down to 207b but an empty file has a
							// similar zip size, so we can't know from the zip size itself.
							unzippedSize, err := s3.GetUnzippedObjectSize(s3Bucket.Value, s3Key.Value)
							if err != nil {
								config.Debugf("Unable to unzip object: %v", err)
							} else if unzippedSize == 0 {
								forecast.Add(F0021, false, "S3 object has a zero byte unzipped size")
							} else {
								forecast.Add(F0021, true, "S3 object has a non-zero unzipped size")
							}
						} else {
							forecast.Add(F0021, true, "S3 object has a non-zero length less than 50Mb")
						}
					} else {
						if obj.SizeBytes == 0 {
							forecast.Add(F0021, false, "S3 object has zero bytes")
						} else {
							forecast.Add(F0021, false, "S3 object is greater than 50Mb")
						}
					}
				}
			}

			spinner.Pop()
		} else {
			config.Debugf("%s does not have S3Bucket and S3Key", input.logicalId)
		}
	} else {
		config.Debugf("Unexpected missing Code property from %s", input.logicalId)
	}
}

// checkLambdaFunction checks for potential stack failures related to functions
func CheckLambdaFunction(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	checkLambdaRole(&input, &forecast)

	checkLambdaS3Bucket(&input, &forecast)

	return forecast
}
