## rain cc deploy

Deploy a local template directly using the Cloud Control API (Experimental!)

### Synopsis

Creates or updates resources directly using Cloud Control API from the template file <template>.
You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!


```
rain cc deploy <template> <name>
```

### Options

```
  -c, --config string           YAML or JSON file to set tags and parameters
      --debug                   Output debugging information
  -x, --experimental            Acknowledge that this is an experimental feature
  -h, --help                    help for deploy
      --ignore-unknown-params   Ignore unknown parameters
      --params strings          set parameter values; use the format key1=value1,key2=value2
  -p, --profile string          AWS profile name; read from the AWS CLI configuration file
  -r, --region string           AWS region to use
      --s3-bucket string        Name of the S3 bucket that is used to upload assets
      --s3-prefix string        Prefix to add to objects uploaded to S3 bucket
      --tags strings            add tags to the stack; use the format key1=value1,key2=value2
  -u, --unlock string           Unlock <lockid> and continue
  -y, --yes                     don't ask questions; just deploy
```

### Options inherited from parent commands

```
      --no-colour   Disable colour output
```

### SEE ALSO

* [rain cc](rain_cc.md)	 - Interact with templates using Cloud Control API instead of CloudFormation

###### Auto generated by spf13/cobra on 4-Jul-2025
