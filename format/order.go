package format

func ordering(path []interface{}) []string {
	switch {
	case len(path) == 0:
		return []string{
			"AWSTemplateFormatVersion",
			"Description",
			"Metadata",
			"Parameters",
			"Mappings",
			"Conditions",
			"Transform",
			"Resources",
			"Outputs",
		}
	case path[0] == "Parameters" && len(path) == 2:
		return []string{
			"Type",
			"Default",
		}
	case path[0] == "Transform" || path[len(path)-1] == "Fn::Transform":
		return []string{
			"Name",
			"Parameters",
		}
	case path[0] == "Resources" && len(path) == 2:
		return []string{
			"Type",
		}
	case path[0] == "Outputs" && len(path) == 2:
		return []string{
			"Description",
			"Value",
			"Export",
		}
	case len(path) > 2 && path[len(path)-2] == "Policies":
		return []string{
			"PolicyName",
			"PolicyDocument",
		}
	case path[len(path)-1] == "PolicyDocument" || path[len(path)-1] == "AssumeRolePolicyDocument":
		return []string{
			"Version",
			"Id",
			"Statement",
		}
	case len(path) > 2 && path[len(path)-2] == "Statement":
		return []string{
			"Sid",
			"Effect",
			"Principal",
			"NotPrincipal",
			"Action",
			"NotAction",
			"Resource",
			"NotResource",
			"Condition",
		}
	case len(path) > 2 && path[len(path)-3] == "Resources" && path[len(path)-1] == "Properties":
		return []string{
			"Name",
			"Description",
		}
	default:
		return []string{}
	}
}
