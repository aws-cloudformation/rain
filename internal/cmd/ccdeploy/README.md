# ccdeploy

The `rain ccdeploy` command provisions resources using the AWS Cloud Control API. It does not use CloudFormation, so there will be no managed stack to interact with after running this command. All API calls are made directly from the client, and the state for the resources is stored by Rain in the same S3 bucket that is used for assets.

This is a highly experimental feature that exists mainly as a prototype for what a client-side provisioning engine might look like. *Do not* use this for production workloads.

Only resources that have been fully migrated to the CloudFormation registry can be provisioned with this command.

The following is an excerpt from a blog post: [The history and future roadmap of the AWS CloudFormation Registry](https://aws.amazon.com/blogs/devops/cloudformation-coverage/)

If you want to see if a given CloudFormation resource is on the new registry model or not, check if the provisioning type is either Fully Mutable or Immutable by invoking the DescribeType API and inspecting the ProvisioningType response element.

Here is a sample CLI command that gets a description for the AWS::Lambda::Function resource, which is on the new registry model.

```sh
$ aws cloudformation describe-type --type RESOURCE \
    --type-name AWS::Lambda::Function | grep ProvisioningType

   "ProvisioningType": "FULLY_MUTABLE",
```

The difference between FULLY\_MUTABLE and IMMUTABLE is the presence of the Update handler. FULLY\_MUTABLE types includes an update handler to process updates to the type during stack update operations. Whereas, IMMUTABLE types do not include an update handler, so the type can’t be updated and must instead be replaced during stack update operations. Legacy resource types will be NON\_PROVISIONABLE.

## State management

State cannot be managed based on the template alone, due to the fact that primary identifiers are not always required (and often disouraged). For example, the following template deploys an S3 bucket:

```yaml
Resources
  MyBucket:
    Type: AWS::S3::Bucket
```

The physical name of the bucket was not specified, and since two different templates could both have an S3 bucket with the logical ID "MyBucket", there is no way to tell from looking at the template alone what bucket it corresponds to in your account.

It is obviously very important to store the state in a way that it cannot be lost or associated with the wrong AWS environment. It is also important to make sure two processes don't try to deploy the same template at the same time.

The state file for `rain ccdeploy` is stored as a YAML file that has the same format as the source CloudFormation template. Extra sections are added to the file to associate state with the deployed resources.

```yaml
State:
    FirstDeployment:
        FilePath:
        Identity:
    LastDeployment:
        ...

Resources:
    MyResource:
        Type: ...
        State:
            ...
```

Each deployment is stored in its own folder. The folder has the state file and also a lock file, if a deployment is currently in progress. If a user attempts a deploymemnt while there is a lock file, the command gives an error message with instructions on how to remediate the issue. Oten times, this will result from a deployment that failed halfway through.

```
rain-artifacts-0123456789012-us-east-1/
  deployments/
    name1/
        lock.txt
        state.yaml
    name2/
        lock.txt
        state.yaml
```


Drift detection can be run on the state file to inspect the actual resource properties and compare them to the stored state.




