## Rain

Rain is what happens when you have a lot of CloudFormation. Rain brings together a set of useful functionality from other CloudFormation tools as well as providing some helpful short commands that make for an intuitive CloudFormation development workflow.

## License

This library is licensed under the Apache 2.0 License. 

## Usage

```
Usage: rain [COMMAND] [OPTIONS...]

  The Rain CLI is a tool to save you some typing when working with CloudFormation
  
  rain is extensible and searches for commands in the following order:
  1. commands built in to the CloudFormation CLI itself
  2. binaries in your path that begin with 'cfn-'
  3. if the command you supply doesn't match 1 or 2, cfn runs 'aws cloudformation <command>'

Built-in commands:

  ls  - List running CloudFormation stacks
```
