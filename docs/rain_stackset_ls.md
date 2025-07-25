## rain stackset ls

List a CloudFormation stack sets in a given region

### Synopsis

List a CloudFormation stack sets in a given region. If you specify a stack set name it will show all the stack instances and last 10 operations.

```
rain stackset ls <stack set>
```

### Options

```
      --admin            Use delegated admin permissions
  -a, --all              list stacks in all regions; if you specify a stack set name, show more details
  -h, --help             help for ls
  -p, --profile string   AWS profile name; read from the AWS CLI configuration file
  -r, --region string    AWS region to use
```

### Options inherited from parent commands

```
      --debug       Output debugging information
      --no-colour   Disable colour output
```

### SEE ALSO

* [rain stackset](rain_stackset.md)	 - This command manipulates stack sets.

###### Auto generated by spf13/cobra on 4-Jul-2025
