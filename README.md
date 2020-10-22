![Build badge](https://codebuild.eu-west-2.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoiWjJrNU1WSTg0OUswalBkRWFWQnVTeDk4Zm8xTGNiQ0NUNnNuYkxWWjZHcnNWMzlXOHZzMVJwTE1QTzFqcFNyTisxWmVPc0d6TFVnMnVNZjZRY2FyQmRNPSIsIml2UGFyYW1ldGVyU3BlYyI6IkE5K0tnNU5YRmQ3ZWE5VHUiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=develop)

# Rain

> Rain is what happens when you have a lot of CloudFormation

*Rain is currently in preview and shouldn't yet be considered stable enough for production use.*

Please report any bugs you find [through GitHub issues](https://github.com/aws-cloudformation/rain/issues).

You can read the full documentation at <https://aws-cloudformation.github.io/rain/>.

Here's what rain looks like:

[![Make it Rain](https://asciinema.org/a/269609.png)](https://asciinema.org/a/269609?autoplay=1)

## Installing

You can download the appropriate binary for your system from [the releases page](https://github.com/aws-cloudformation/rain/releases).

Alternatively, if you have [Go](https://golang.org) (v1.12 or higher) installed:

`GO111MODULE=on go get github.com/aws-cloudformation/rain`

You can find shell completion scripts in [docs/bash_completion.sh](./docs/bash_completion.sh) and [docs/zsh_completion.sh](./docs/zsh_completion.sh).

Rain requires the [AWS CLI](https://aws.amazon.com/cli/) to package CloudFormation templates for deployment. (See the `[aws cloudformation package](https://docs.aws.amazon.com/cli/latest/reference/cloudformation/package.html) command for details.)

## License

Rain is licensed under the Apache 2.0 License. 

## Usage

You will need to make sure you have installed and configured [the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-welcome.html) as rain uses the same credentials and configuration.

Rain is composed of a number of sub-commands. Run `rain` without any arguments to get help.

```
Usage:
  rain [command]

Stack commands:
  cat         Get the CloudFormation template from a running stack
  deploy      Deploy a CloudFormation stack from a local template
  logs        Show the event log for the named stack
  ls          List running CloudFormation stacks
  rm          Delete a running CloudFormation stack
  watch       Display an updating view of a CloudFormation stack

Template commands:
  build       Create CloudFormation templates
  check       Validate a CloudFormation template against the spec
  diff        Compare CloudFormation templates
  fmt         Format CloudFormation templates
  merge       Merge two or more CloudFormation templates
  tree        Find dependencies of Resources and Outputs in a local template

Other Commands:
  help        Help about any command
  info        Show your current configuration

Flags:
      --debug            Output debugging information
  -h, --help             help for rain
  -p, --profile string   AWS profile name; read from the AWS CLI configuration file
  -r, --region string    AWS region to use

Use "rain [command] --help" for more information about a command.
```

## Design Principles

Rain is designed with the following principles in mind:

### Do the most obvious and useful thing by default

The default behaviour for every feature of rain should:
* do what you would expect from the name
* output only the most pertinent information
* have a high signal to noise ratio

Examples:

* `ls` lists stacks in the current region, showing only the stack name and status

### Have consistent options available for non-default behaviour

While the default behaviour should stick to the obvious, every command should offer options for other use cases.

Examples:

* `-a | --all` shows information that would otherwise be filtered:
    * `ls -a` lists stacks in all regions
    * `logs -a` shows uninteresting logs

* `-l | --long` shows more detailed information:
    * `logs -l` shows all available log details

### Be human-friendly

Output should:
* be easy to read, making good use of white-space while taking up the minimum of space needed to convey the important information
* use colour to highlight things the user needs to be aware of
* use colour to differentiate different kinds of information
* show progress so that the user knows we haven't crashed

Examples:

* All commands colourise the stack/resource status
* `logs` colours the message field so that it stands out
* `deploy` shows a "spinner" while working

### Be machine-friendly

Output should:
* be consumable by other processes; use YAML where possible
* strip special formatting when it is part of a pipe

Examples:

* All commands have YAML-compatible output
* All commands strip formatting if stdout is not connected to a terminal

## Other CloudFormation tools

In alphabetical order:

* [cfn-flip](https://github.com/awslabs/aws-cfn-template-flip)

    cfn-flip converts AWS CloudFormation templates between JSON and YAML formats, making use of the YAML format's short function syntax where possible.

* [cfn-format](https://github.com/awslabs/aws-cloudformation-template-formatter)

    cfn-format reads in an existing AWS CloudFormation template and outputs a cleanly-formatted, easy-to-read copy of the same template adhering to standards as used in AWS documentation. cfn-format can output either YAML or JSON as desired.

* [cfn-lint](https://github.com/aws-cloudformation/cfn-python-lint)

    Validate CloudFormation yaml/json templates against the CloudFormation spec and additional checks. Includes checking valid values for resource properties and best practices.

* [cfn-nag](https://github.com/stelligent/cfn_nag)

    The cfn-nag tool looks for patterns in CloudFormation templates that may indicate insecure infrastructure.

* [cfn-skeleton](https://github.com/awslabs/aws-cloudformation-template-builder)

    cfn-skeleton that consumes the published CloudFormation specification and generates skeleton CloudFormation templates with mandatory and optional parameters of chosen resource types pre-filled with placeholder values.

* [sceptre](https://sceptre.cloudreach.com/)

    Sceptre is a tool to drive CloudFormation. Sceptre manages the creation, update and deletion of stacks while providing meta commands which allow users to retrieve information about their stacks.

* [takomo](https://takomo.io)

    Takomo makes it easy to organize, parameterize and deploy CloudFormation stacks across multiple regions and accounts. You can also manage your AWS organization, including its member accounts, organizational units and policies.

* [taskcat](https://github.com/aws-quickstart/taskcat)

    taskcat is a tool that tests AWS CloudFormation templates. It deploys your AWS CloudFormation template in multiple AWS Regions and generates a report with a pass/fail grade for each region. You can specify the regions and number of Availability Zones you want to include in the test, and pass in parameter values from your AWS CloudFormation template. taskcat is implemented as a Python class that you import, instantiate, and run.

Are we missing an excellent tool? Let us know via a GitHub issue.
