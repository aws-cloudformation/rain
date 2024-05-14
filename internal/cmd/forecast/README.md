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

- The resource already exists (for stack creation with hard coded resource
  names)
- IAM permissions to interact with the resource. Keep in mind that this is a
  slow operation and can be suppressed with the `--skip-iam` argument. It is
  also not guaranteed to be 100% accurate, due to the difficulty with
  predicting the exact ARNs for all possible resources that are involved with
  the resource provider.

## Specific checks

- For a delete operation, the S3 bucket is not empty
- An S3 bucket policy has an invalid principal
- Make sure RDS cluster configuration makes sense for the chosen engine 
- Check EC2 instances and launch configurations:
  - KeyName exists
  - Instance type exists
  - Instance type and AMI match

## Estimates

The forecast command also tries to estimate how long it thinks your stack will
take to deploy.

## Roadmap

You can view the issues list for the forecast command
[here](https://github.com/aws-cloudformation/rain/issues?q=is%3Aopen+is%3Aissue+label%3Aforecast).

Please feel free to create an issue here whenever you get a stack failure that
you think could have been prevented by one of these checks.

In the near term, we're going to add the ability to suppress specific checks by
giving each of them a code.

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
- Security group exists
- Is an EC2 instance type available in the AZ



