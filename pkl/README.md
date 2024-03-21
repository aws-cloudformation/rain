# pkl

Pkl is a new configuration language created by Apple. It is capable of serializing to other formats like YAML, so it's possible to write a CloudFormation template with pkl.

https://pkl-lang.org/index.html

The following is a basic example of a pkl CloudFormation template.

```pkl
AWSTemplateFormatVersion: String = "2010-09-09"
Description = "My template"
Parameters {
    ["Name"] {
        ["Type"] = "String"
    }
}
Resources {
    ["MyBucket"] {
        ["Type"] = "AWS::S3::Bucket"
        ["Properties"] {
            ["BucketName"] {
                ["Ref"] = "Name"
            }
        }
    }
}
```

Running `pkl eval -f yaml` on this file results in the following:

```yaml
AWSTemplateFormatVersion: 2010-09-09
Description: My template
Parameters:
  Name:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName:
        Ref: Name
```

In pkl it's possible to define type-safe configurations, which gives you syntax
validation and IDE support. Rain can generate pkl classes based on the
CloudFormation registry, and the repository hosts pkl modules that you can
import into your own projects.

Here's an example of a file you could write using these modules:

```pkl
amends "@cfn/template.pkl"
import "@cfn/cloudformation.pkl" as cfn
import "@cfn/aws/s3/bucket.pkl" as bucket

Description = "Create a bucket"

Metadata { 
    ["Foo"] = "bar"
}

Parameters {
    ["Name"] {
        Type = "String"
        Default = "baz"
    }
}

Resources {
    ["TypedBucket"] = new bucket.Bucket {
        BucketName = cfn.Ref("Name")
    }
}
```

Note that the package alias `@cfn` is enabled by creating a `PklProject` file that looks like this:

```pkl
amends "pkl:Project"

dependencies {
    ["cfn"] {
        uri = "package://github.com/aws-cloudformation/rain/releases/download/v1.8.2-alpha1/cloudformation@1.8.2-alpha1"
    }
}
```

