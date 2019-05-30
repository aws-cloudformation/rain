# Rain

Rain is a development workflow tool for working with AWS CloudFormation.

> Rain is also what happens when you have a lot of CloudFormation

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
