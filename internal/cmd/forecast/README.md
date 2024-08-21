# Rain Forecast

The experimental `rain forecast` command makes API calls into your account to
try to predict things that might fail during stack create, update, and delete
operations. This command is not meant to be a substitute for the CloudFormation
Linter (cfn-lint), which ideally is already an integral part of your
development process.

In order to use this command, supply the `-x` argument to recognize the fact
that this feature is currently experimental could change with minor version
upgrades.

```sh 
rain forecast -x --skip-iam my-template.yaml my-stack-name 
```

You can also supply a CLI profile with the `--profile` argument to assume a
different role for the checks you make against the template.

## Generic checks

This command currently makes a few generic checks for a wide range of
resources:

- FG001: The resource already exists (for stack creation with hard coded resource
  names)
- FG002: IAM permissions to interact with the resource. Keep in mind that this is a
  slow operation and is disabled by default. You can enable it with the `--include-iam` argument. It is
  also not guaranteed to be 100% accurate, due to the difficulty with
  predicting the exact ARNs for all possible resources that are involved with
  the resource provider.

## Specific checks

These can be ignored with the `--ignore` argument.

| Code  | Description                                                                    |                                      
|-------|--------------------------------------------------------------------------------|
| F0001 | For a delete operation, the S3 bucket is not empty                             |
| F0002 | S3 bucket policy has an invalid principal                                      |
| F0003 | RDS cluster configuration is correct for the chosen engine                     |
| F0004 | RDS monitoring role arn is correct                                             |
| F0005 | RDS cluster quota is not at limit                                              |
| F0006 | RDS instance configuration is correct for the chosen engine                    |
| F0007 | EC2 instance and launch template KeyName exists                                |
| F0008 | EC2 instance and launch template InstanceType exists                           |
| F0009 | EC2 instance and launch template instance type and AMI match                   |
| F0010 | Within the same template, are all security groups pointing to the same network |
| F0011 | If there is no default VPC, does each security group have a vpc configured?    |
| F0012 | Certificate not found for elastic load balancer                                |
| F0013 | SNS Topic Key is valid                                                         |
| F0014 | ELB target group Port and Protocol match                                       |
| F0015 | ELB target groups must be of type instance if they are used by an ASG          |
| F0016 | Lambda function role exists                                                    |
| F0017 | Lambda function role can be assumed                                            |
| F0018 | SageMaker Notebook quota limit has not been reached                            |
| F0019 | Lambda S3Bucket exists                                                         |
| F0020 | Lambda S3Key exists                                                            |
| F0021 | Lambda zip file has a valid size                                               |

## Estimates

The forecast command also tries to estimate how long it thinks your stack will
take to deploy.

## Plugins

You can build a plugin that runs prediction functions that you write yourself.
This can be useful if you have internal systems that you want to check to make 
sure the template is valid before deployment. You can see an example plugin in 
`cmd/sample_plugin/main.go`.

```go
package main

import (
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

type PluginImpl struct{}

// A sample prediction function.
func predictLambda(input fc.PredictionInput) fc.Forecast {
	forecast := fc.MakeForecast(&input)

	// Implement whatever checks you want to make here
	forecast.Add("CODE", false, "testing plugin", 0)

	return forecast
}

// GetForecasters must be implemented by forecast plugins
// This function returns all predictions functions implemented by the plugin
func (p *PluginImpl) GetForecasters() map[string]func(input fc.PredictionInput) fc.Forecast {
	retval := make(map[string]func(input fc.PredictionInput) fc.Forecast)

	retval["AWS::Lambda::Function"] = predictLambda

	return retval
}

// Leave main empty
func main() {
}

// This variable is required to allow rain to find the implementation
var Plugin = PluginImpl{}

```

## Roadmap

You can view the issues list for the forecast command
[here](https://github.com/aws-cloudformation/rain/issues?q=is%3Aopen+is%3Aissue+label%3Aforecast).

Please feel free to create an issue here whenever you get a stack failure that
you think could have been prevented by one of these checks.

Checks we plan to implement:

- DynamoDB global table replica region requirements
- Prefix list does not exist
- Flow Log format errors
- SES identity not verified
- SES sending pool does not exist
- EIP limit
- Function version does not exist
- Warn on resource replacements for active traffic
- API gateway account trust permission



