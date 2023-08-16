[![Unit tests](https://github.com/aws-cloudformation/rain/actions/workflows/test.yml/badge.svg)](https://github.com/aws-cloudformation/rain/actions/workflows/test.yml)
[![Mentioned in Awesome CloudFormation](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/aws-cloudformation/awesome-cloudformation)

# Rain

* Documentation: <https://aws-cloudformation.github.io/rain/>

> Rain is what happens when you have a lot of CloudFormation

Rain is also a command line tool for working with [AWS CloudFormation](https://aws.amazon.com/cloudformation/) templates and stacks.

[![Make it Rain](./docs/rain.svg)](https://asciinema.org/a/vtbAXkriC0zg0T2UzP0t63G4S?autoplay=1)

## Discord

Join us on Discord to discuss rain and all things CloudFormation! Connect and interact with CloudFormation developers and
experts, find channels to discuss rain, the CloudFormation registry, StackSets,
cfn-lint, Guard and more:

[![Join our Discord](https://discordapp.com/api/guilds/981586120448020580/widget.png?style=banner3)](https://discord.gg/9zpd7TTRwq)

## Key features

* **Interactive deployments**: With `rain deploy`, rain packages your CloudFormation templates using [`aws cloudformation package`](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/cloudformation/package.html), prompts you for any parameters that have not yet been defined, shows you a summary of the changes that will be made, and then displays real-time updates as your stack is being deployed. Once finished, you get a summary of the outcome along with any error messages collected along the way - including errors messages for stacks that have been rolled back and no longer exist.

* **Consistent formatting of CloudFormation templates**: Using `rain fmt`, you can format your CloudFormation templates to a consistent standard or reformat a template from JSON to YAML (or YAML to JSON if you prefer). Rain preserves your comments when using YAML and switches use of [intrinsic functions](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html) to use the short syntax where possible.

* **Combined logs for nested stacks with sensible filtering**: When you run `rain log`, you will see a combined stream of logs from the stack you specified along with any nested stack associated with it. Rain also filters out uninteresting log messages by default so you just see the errors that require attention.

* **Build new CloudFormation templates**: `rain build` generates new CloudFormation templates containing skeleton resources that you specify. This saves you having to look up which properties are available and which are required vs. optional.

* **Manipulate CloudFormation stack sets**: `rain stackset deploy` creates a new stackset, updates an existing one or adds a stack instance(s) to an existing stack set. You can list stack sets using `rain stackset ls`, review stack set details with `rain stackset ls <stack set name>` and delete stack set and\or its instances with `rain stackset rm <stack set name>`

* **Predict deployment failures** (EXPERIMENTAL): `rain forecast` analyzes a template and the target deployment account to predict things that might go wrong when you attempt to create, update, or delete a stack. This command speeds up development by giving you advanced notice for issues like missing permissions, resources that already exist, and a variety of other common resource-specific deployment blockers.

* **Modules** (EXPERIMENTAL): `rain pkg` supports client-side module development with the `!Rain::Module` directive. Rain modules are partial templates that are inserted into the parent template, with some extra functionality added to enable extending existing resource types.

_Note that in order to use experimental commands, you have to add `--experimental` or `-x` as an argument._

## Getting started

If you have [homebrew](https://brew.sh/) installed, `brew install rain`

Or you can download the appropriate binary for your system from [the releases page](https://github.com/aws-cloudformation/rain/releases).

Or if you're a [Gopher](https://blog.golang.org/gopher), you can `GO111MODULE=on go install github.com/aws-cloudformation/rain/cmd/rain`

```
Usage:
  rain [command]

Stack commands:
  cat         Get the CloudFormation template from a running stack
  deploy      Deploy a CloudFormation stack from a local template
  logs        Show the event log for the named stack
  ls          List running CloudFormation stacks
  rm          Delete a running CloudFormation stack
  stackset    This command manipulates stack sets.
  watch       Display an updating view of a CloudFormation stack

Template commands:
  bootstrap   Creates the artifacts bucket
  build       Create CloudFormation templates
  diff        Compare CloudFormation templates
  fmt         Format CloudFormation templates
  forecast    Predict deployment failures
  merge       Merge two or more CloudFormation templates
  pkg         Package local artifacts into a template
  tree        Find dependencies of Resources and Outputs in a local template

Other Commands:
  console     Login to the AWS console
  help        Help about any command
  info        Show your current configuration
```

You can find shell completion scripts in [docs/bash_completion.sh](./docs/bash_completion.sh) and [docs/zsh_completion.sh](./docs/zsh_completion.sh).

## Contributing

Rain is written in [Go](https://golang.org/) and uses the [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2).

To contribute a change to Rain, [fork this repository](https://github.com/aws-cloudformation/rain/fork), make your changes, and submit a Pull Request.

### Go Generate

The `README.md`, documentation in `docs/`, the auto completion scripts and a copy of the cloudformation specification in `cft/spec/cfn.go` are generated through `go generate`.

## License

Rain is licensed under the Apache 2.0 License. 

## Example Usage

### Packaging

The `rain pkg` command can be used as a replacement for the `aws cloudformation
package` CLI command.  When packaging a template, `rain` looks for specific
directives to appear in resources.

#### Embed

The `!Rain::Embed` directive simply inserts the contents of a file into the template as a string.

The template:

```yaml
Resources:
  Test:
    Type: AWS::CloudFormation:WaitHandle
    Metadata:
      Comment: !Rain::Embed embed.txt
```
The contents of `embed.txt`, which is in the same directory as the template:

```txt
This is a test
```

The resulting packaged template:

```yaml
Resources:
  Test:
    Type: AWS::CloudFormation:WaitHandle
    Metadata:
      Comment: This is a test
```

#### Include

The `!Rain::Include` directive parses a YAML or JSON file and inserts the object into the template.

The template:

```yaml
Resources:
  Test:
    !Rain::Include include-file.yaml
```

The file to be included:

```yaml
Type: AWS::S3::Bucket
Properties:
  BucketName: test
```

The resulting packaged template:

```yaml
Resources:
  Test:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test
```

#### Env

The `!Rain::Env` directive reads environment variables and inserts them into the template as strings.

The template:

```yaml
Resources:
  Test:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Rain::Env BUCKET_NAME
```

The resulting packaged template, if you have exported an environment variable named `BUCKET_NAME` with value `abc`:

```yaml
Resources:
  Test:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: abc
```

#### S3Http

The `!Rain::S3Http` directive uploads a file or directory to S3 and inserts the
HTTPS URL into the template as a string.

The template:

```yaml
Resources:
  Test:
    Type: A::B::C
    Properties:
      TheS3URL: !Rain::S3Http s3http.txt
```

If you have a file called `s3http.txt` in the same directory as the template,
rain will use your current default profile to upload the file to the artifact
bucket that rain creates as a part of bootstrapping. If the path provided is a 
directory and not a file, the directory will be zipped first.

```yaml
Resources:
  Test:
    Type: A::B::C
    Properties:
      TheS3URL: https://rain-artifacts-012345678912-us-east-1.s3.us-east-1.amazonaws.com/a84b588aa54068ed4b027b6e06e5e0bb283f83cf0d5a6720002d36af2225dfc3
```

#### S3 

The `!Rain::S3` directive is basically the same as `S3Http`, but it inserts the S3 URI instead of an HTTPS URL.

The template:

```yaml
Resources:
  Test:
    Type: A::B::C
    Properties:
      TheS3URI: !Rain::S3 s3.txt
```

If you have a file called `s3.txt` in the same directory as the template,
rain will use your current default profile to upload the file to the artifact
bucket that rain creates as a part of bootstrapping. If the path provided is a 
directory and not a file, the directory will be zipped first.

```yaml
Resources:
  Test:
    Type: A::B::C
    Properties:
      TheS3URI: s3://rain-artifacts-755952356119-us-east-1/a84b588aa54068ed4b027b6e06e5e0bb283f83cf0d5a6720002d36af2225dfc3 
```

If instead of providing a path to a file, you supply an object with properties, you can exercise more control over how the object is uploaded to S3. The following example is a common pattern for uploading Lambda function code.

```yaml
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      Code: !Rain::S3 
        Path: lambda-src 
        Zip: true
        BucketProperty: S3Bucket
        KeyProperty: S3Key
```

The packaged template:

```yaml
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      Code:
        S3Bucket: rain-artifacts-012345678912-us-east-1
        S3Key: 1b4844dacc843f09941c11c94f80981d3be8ae7578952c71e875ef7add37b1a7
```

#### Module

The `!Rain::Module` directive is an experimental feature that allows you to
create local modules of reuseable code that can be inserted into templates. A
rain module is similar in some ways to a CDK construct, in that a module can
extend an existing resource, allowing the user of the module to override
properties. For example, your module could extend an S3 bucket to provide a
default implementation that passes static security scans. Users of the module
would inherit these best practices by default, but they would still have the
ability to configure any of the original properties on `AWS::S3::Bucket`, in
addition to the properties defined as module parameters.

In order to use this feature, you have to acknowledge that it's experimental by
adding a flag on the command line:

`rain pkg -x my-template.yaml`

Keep in mind that with new versions of rain, this functionality could change,
so use caution if you decide to use this feature for production applications.
The `rain pkg` command does not actually deploy any resources if the template
does not upload any objects to S3, so you always have a chance to review the
packaged template. It's recommended to run linters and scanners on the packaged
template, rather than a pre-processed template that makes use of these advanced
directives.

A sample module:

```yaml
Description: |
  This module extends AWS::S3::Bucket

Parameters:
  LogBucketName:
    Type: String

Resources:
  ModuleExtension:
    Metadata:
      Extends: AWS::S3::Bucket
    Properties:
      LoggingConfiguration:
        DestinationBucketName: !Ref LogBucket
      BucketEncryption:
        ServerSideEncryptionConfiguration:
          - ServerSideEncryptionByDefault:
             SSEAlgorithm: AES256
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true
      Tags:
        - Key: test-tag
          Value: test-value1
  
  LogBucket:
    Type: AWS::S3::Bucket
    DeletionPolicy: Retain
    Properties:
      BucketName: !Ref LogBucketName
      BucketEncryption:
        ServerSideEncryptionConfiguration:
          - ServerSideEncryptionByDefault:
              SSEAlgorithm: AES256
      VersioningConfiguration:
        Status: Enabled
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true
```

A module must include a resource called `ModuleExtension`, and it must indicate 
which resource it is extending with a Metadata entry called `Extends`.

Note that we defined a single parameter to the module called `LogBucketName`.
In the module, we create an additional bucket to hold logs, and we apply the
name to that bucket. In the template that uses the module, we specify that name
as a property. This shows how we have extended the basic behavior of a bucket
to add something new. 

A template that uses the module:

```yaml
Resources:
  ModuleExample:
    Type: !Rain::Module "./bucket-module.yaml"
    UpdateReplacePolicy: Delete
    Properties:
      LogBucketName: test-module-log-bucket
      VersioningConfiguration:
        Status: Enabled
      Tags:
        - Key: test-tag
          Value: test-value2
```

Note that in addition to supplying the expected `LogBucketName` property, we have also 
decided to override a few of the properties on the underlying `AWS::S3::Bucket` resource, 
which shows the flexibility of the inheritance model.

The resulting template after running `rain pkg`:

```yaml
Resources:
  ModuleExample:
    Type: AWS::S3::Bucket
    Properties:
      LoggingConfiguration:
        DestinationBucketName: !Ref ModuleExampleLogBucket
      BucketEncryption:
        ServerSideEncryptionConfiguration:
          - ServerSideEncryptionByDefault:
              SSEAlgorithm: AES256
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true
      Tags:
        - Key: test-tag
          Value: test-value2
      VersioningConfiguration:
        Status: Enabled

  ModuleExampleLogBucket:
    DeletionPolicy: Retain
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-module-log-bucket
      BucketEncryption:
        ServerSideEncryptionConfiguration:
          - ServerSideEncryptionByDefault:
              SSEAlgorithm: AES256
      VersioningConfiguration:
        Status: Enabled
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true
```

## Other CloudFormation tools

* [cfn-lint](https://github.com/aws-cloudformation/cfn-python-lint)

    Validate CloudFormation yaml/json templates against the CloudFormation spec and additional checks. Includes checking valid values for resource properties and best practices.

* [cfn-guard](https://docs.aws.amazon.com/cfn-guard/latest/ug/what-is-guard.html)

    Guard is a policy evaluation tool that allows you to build your own rules with a custom DSL. You can also pull rules from the 
    [guard registry](https://github.com/aws-cloudformation/aws-guard-rules-registry) to scan your templates for resources that don't comply with common best practices.

* [taskcat](https://github.com/aws-quickstart/taskcat)

    taskcat is a tool that tests AWS CloudFormation templates. It deploys your AWS CloudFormation template in multiple AWS Regions and generates a report with a pass/fail grade for each region. You can specify the regions and number of Availability Zones you want to include in the test, and pass in parameter values from your AWS CloudFormation template. taskcat is implemented as a Python class that you import, instantiate, and run.

Are we missing an excellent tool? Let us know via a GitHub issue.

