# Rain Web App Sample

This application demonstrates advanced capabilities of AWS
CloudFormation Rain, a CLI tool written in Go that uses the AWS SDK to
improve the developer experience for authoring and deploying
CloudFormation templates.

Rain allows you to create client-side modules in YAML, so that you
can re-use common patterns and speed up the development process. These
modules look just like regular CloudFormation templates and can be
included in a parent template using the `!Rain::Module`
directive.

Rain also looks for special Metadata comments on the
`AWS::S3::Bucket` resource to run external build scripts
before deployment, and then to upload the contents of a folder to the
newly created bucket after deployment is complete

For Lambda functions, Rain can run a script to build and/or package
your handler, and then it will upload a zip file to an assets bucket
before deploying the stack.

Combining these features together allows you to deploy an entire
serverless web application with a single command, in this case
`rain deploy -x webapp.yaml`.

TODO: Architecture diagram

TODO: Sequence of operations

TODO: Template snippets


