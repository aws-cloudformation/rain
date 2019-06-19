# Rain

Rain is a development workflow tool for working with AWS CloudFormation.

*Rain is currently in preview and shouldn't yet be considered stable enough for production use. Please report any bugs you find [through GitHub issues](https://github.com/aws-cloudformation/rain/issues).*

> Rain is also what happens when you have a lot of CloudFormation

Here's what it looks like:

![Make it Rain](./media/rain.gif)

## Installing

You can download the appropriate binary for your system from [the releases page](https://github.com/aws-cloudformation/rain/releases).

Alternatively, if you have [Go](https://golang.org) (v1.12 or higher) installed:

`GO111MODULE=on go get github.com/aws-cloudformation/rain`

## License

Rain is licensed under the Apache 2.0 License. 

## Usage

You will need to make sure you have installed and configured [the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-welcome.html) as rain uses the same credentials and configuration.

Rain is composed of a number of sub-commands. Invoke a command like this:

```
rain [command] [arguments...]
```

The following commands are available:

```
cat         Get the CloudFormation template from a running stack
check       Show your current configuration
deploy      Deploy a CloudFormation stack from a local template
diff        Compare CloudFormation templates
logs        Show the event log for the named stack
ls          List running CloudFormation stacks
rm          Delete a running CloudFormation stack
```

You can get additional information about any command by running:

```
rain help [command]
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

* All commands (except ls for now) output in YAML-compatible output
* All commands strip formatting if stdout is not connected to a terminal
