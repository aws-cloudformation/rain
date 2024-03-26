# Pkl Patterns

**NOTE** These high level patterns are experimental, and may change in minor version releases of the package.

In the examples below, modules are prefixed by `@cfn`. To enable this, put your
template into a Pkl project folder, with a file called `PklProject`. A sample
of that file is below (you may need to adjust the version number in the URI.

```pkl
amends "pkl:Project"

dependencies {
    ["cfn"] {
        uri = "package://github.com/aws-cloudformation/rain/releases/download/v1.8.3/cloudformation@1.8.3"
    }
}
```

# bucket.pkl

# vpc.pkl

```pkl
amends "@cfn/template.pkl"
import "@cfn/cloudformation.pkl" as cfn
import "@cfn/patterns/vpc.pkl"

local pub1 = new vpc.Subnet {
    LogicalId = "Pub1"
    IsPublic = true
    Az = cfn.Select(0, cfn.GetAZs("us-east-1")) 
    Cidr = "10.0.0.0/18"
}

local pub2 = new vpc.Subnet {
    LogicalId = "Pub2"
    IsPublic = true
    Az = cfn.Select(1, cfn.GetAZs("us-east-1")) 
    Cidr = "10.0.64.0/18"
}

local priv1 = new vpc.Subnet {
    LogicalId = "Priv1"
    IsPublic = false
    Az = cfn.Select(0, cfn.GetAZs("us-east-1")) 
    Cidr = "10.0.128.0/18"
    PublicNATGateway = pub1.natGateway
}

local priv2 = new vpc.Subnet {
    LogicalId = "Priv2"
    IsPublic = false
    Az = cfn.Select(1, cfn.GetAZs("us-east-1")) 
    Cidr = "10.0.192.0/18"
    PublicNATGateway = pub2.natGateway
}

local myvpc = new vpc {
    LogicalId = "MyVPC"
    Subnets {
        pub1
        priv1
        pub2
        priv2
    }
}

Resources {
    // Create the VPC
    ...myvpc.resources

    // Create other resources inside the VPC...
}

Outputs {
    ...myvpc.outputs
}
```


