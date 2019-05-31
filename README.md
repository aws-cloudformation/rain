# Rain

Rain is a development workflow tool for working with AWS CloudFormation.

*Rain is currently in preview and shouldn't yet be considered stable enough for production use. Please report any bugs you find [through GitHub issues](https://github.com/aws-cloudformation/rain/issues)*

> Rain is also what happens when you have a lot of CloudFormation

Here's what it looks like:

![Make it Rain](./media/rain.gif)

## Installing

First, you'll need [Go](https://golang.org/) installed.

After that, to install rain, run the following:

`go install github.com/aws-cloudformation/rain`

## License

This library is licensed under the Apache 2.0 License. 

## Usage

Rain is composed of a number of sub-commands. Invoke a command like this:

```
rain [command] [arguments...]
```

The following commands are available:

```
cat         Get the CloudFormation template from a running stack
deploy      Deploy a CloudFormation stack from a local template
diff        Compare CloudFormation templates
ls          List running CloudFormation stacks
rm          Delete a running CloudFormation stack
```

You can get additional information about any command by running:

```
rain help [command]
```
