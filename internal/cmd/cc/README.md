# Cloud Control API Deployments 

The `rain cc deploy` command provisions resources using the AWS Cloud Control
API (CCAPI). It does not submit templates to CloudFormation, so there will be
no managed stack to interact with after running this command. API calls are
made directly from the client to CCAPI, and the state for the resources is
stored by Rain in the same S3 bucket that is used for assets.

This is a highly experimental feature that exists mainly as a prototype for
what a client-side provisioning engine might look like. *Do not* use this for
production workloads. (Seriously)

Only resources that have been fully migrated to the CloudFormation registry can
be provisioned with this command. It is important to note that resource
provisioning is still done by the same back-end resource providers that
CloudFormation uses. Those are available on GitHub under the
`aws-cloudformation` organization, for example
[RDS](https://github.com/aws-cloudformation/aws-cloudformation-resource-providers-rds).
The `cc deploy` command makes client-side calls to CCAPI endpoints like `CreateResource`, 
but then CCAPI itself is the one invoking resource providers, which make SDK
calls into specific services.

If you want to learn a bit more about the CloudFormation registry and the
history of Cloud Control API, check out this blog post: [The history and future
roadmap of the AWS CloudFormation
Registry](https://aws.amazon.com/blogs/devops/cloudformation-coverage/)

If you want to see if a CloudFormation resource is on the new registry
model or not, check if the provisioning type is either Fully Mutable or
Immutable by invoking the DescribeType API and inspecting the ProvisioningType
response element.

Here is a CLI command that gets a description for the
AWS::Lambda::Function resource, which is on the new registry model.

```
sh $ aws cloudformation describe-type --type RESOURCE \ --type-name AWS::Lambda::Function | grep ProvisioningType

   "ProvisioningType": "FULLY_MUTABLE", 
```

The difference between FULLY\_MUTABLE and IMMUTABLE is the presence of the
Update handler. FULLY\_MUTABLE types include an update handler to process
updates to the type during stack update operations. IMMUTABLE types do
not include an update handler, so the type can’t be updated and must instead be
replaced during stack update operations. Legacy resource types will be
NON\_PROVISIONABLE.

## Why would I want to use this?

Again, for production workloads, you shouldn't. But the one big benefit is that you 
have access to the resource state, which is described in detail below. You could, 
in theory, modify the state to deal with unexpected deployment failures, or to 
remediate complex drift situations. Template deployment might be slightly faster, since you 
won't wait for the CloudFormation backend to push your stack through the workflow, but since 
CloudFormation uses the same resource providers, the difference will not be huge.

Another good reason is that you are curious about how CCAPI works, and you are
interested in learning about all the really hard things a template provisioning
engine has to do. Let us know if you want to dive in and contribute. 
The best way to learn is by doing!

## State management

State cannot be managed based on the template alone, due to the fact that
primary identifiers are not always required (and often disouraged). For
example, the following template deploys an S3 bucket:

```yaml 
Resources: 
    MyBucket: 
        Type: AWS::S3::Bucket 
```

The physical name of the bucket was not specified, and since two different
templates could both have an S3 bucket with the logical ID "MyBucket", there is
no way to tell from looking at the template alone what bucket it corresponds to
in your account.

It is obviously very important to store the state in a way that it cannot be
lost or associated with the wrong AWS environment. It is also important to make
sure two processes don't try to deploy the same template at the same time.

The state file for `rain cc deploy` is stored as a YAML file that has the same
format as the source CloudFormation template. Extra sections are added to the
file to associate state with the deployed resources.

```yaml 
State: 
  LastWriteTime: ...
  ResourceModels:
    MyResource:
      Identifier:
      Model:
        ...
```

Each deployment has its own state file in the rain artifacts bucket in the
region where you are deploying. If a user tries a deployment while there is a
locked state file, the command gives an error message with instructions on how
to remediate the issue. Often times, this will result from a deployment that
failed halfway through.

```
rain-artifacts-0123456789012-us-east-1/ 
    deployments/ 
        name1.yaml
        name2.yaml
```

Drift detection can be run on the state file to inspect the actual resource
properties and compare them to the stored state. When you deploy a change to 
a template with this command, drift from the stored state will be pointed out 
and you will be able to resolve it before continuing with the update.

## Usage

To use this command, supply the same arguments that you would supply to the `deploy` command:

```sh
$ rain cc deploy -x my-template.yaml my-deployment-name
```

(The `-x` argument stands for `--experimental`. This is a nag to make sure you understand this feature is still in active development!)

To remove resources deployed with `cc deploy`, use the `cc rm` command:

```sh
$ rain cc rm -x my-deployment-name
```

To view the state file for a deployment:

```sh
rain cc state -x my-deployment-name
```

To remediate drift on a deployment (also runs when you `deploy`)

```sh
rain cc drift -x my-deployment name
```

## Unsupported features

Since this is a prototype, some features are not yet supported:

- Not all instrinsic functions have been implemented
- Tags are ignored
- Any resource not yet migrated to the new registry model
- Retention policies
- Probably more stuff that is totally necessary for production use





