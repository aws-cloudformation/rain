## rain stackset deploy

Deploy a CloudFormation stack set from a local template

### Synopsis

Creates or updates a CloudFormation stack set <stackset> from the template file <template>.
If you don't specify a stack set name, rain will use the template filename minus its extension.
If you do not specify a template file, rain will asume that you want to add a new instance to an existing template,
If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.

The config flags can be used to set accounts, regions to operate and tags with parameters to use.
Configuration file with extended options can be provided along with '--config' flag in YAML or JSON format (see [example file](https://github.com/aws-cloudformation/rain/blob/main/test/samples/test-config.yaml) for details).

YAML:
```
Parameters:
	Name: Value
Tags:
	Name: Value
StackSet:
	description: "test description"
	...
StackSetInstances:
	accounts:
		- "123456789123"
	regions:
		- us-east-1
		- us-east-2
	...
```

Account(s) and region(s) provided as flags OVERRIDE values from configuration files. Tags and parameters from the configuration file are MERGED with CLI flag values.


```
rain stackset deploy <template> [stackset] [flags]
```

### Options

```
      --accounts strings         accounts for which to create stack set instances
      --admin                    Use delegated admin permissions
  -c, --config string            YAML or JSON file to set additional configuration parameters
  -d, --detach                   once deployment has started, don't wait around for it to finish
  -h, --help                     help for deploy
  -i, --ignore-stack-instances   ignores adding or removing stack instances while updating, useful if you are managing the stack instances separately
      --params strings           set parameter values; use the format key1=value1,key2=value2
  -p, --profile string           AWS profile name; read from the AWS CLI configuration file
  -r, --region string            AWS region to use
      --regions strings          regions where you want to create stack set instances
      --s3-bucket string         Name of the S3 bucket that is used to upload assets
      --s3-owner string          The account where S3 assets are stored
      --s3-prefix string         Prefix to add to objects uploaded to S3 bucket
      --tags strings             add tags to the stack; use the format key1=value1,key2=value2
  -y, --yes                      update the stackset without confirmation
```

### Options inherited from parent commands

```
      --debug       Output debugging information
      --no-colour   Disable colour output
```

### SEE ALSO

* [rain stackset](rain_stackset.md)	 - This command manipulates stack sets.

###### Auto generated by spf13/cobra on 4-Jul-2025
