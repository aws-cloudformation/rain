package iam

import "testing"

func TestTransformCallerArn2(t *testing.T) {
	if TransformCallerArn("arn:aws:sts::755952356119:assumed-role/Admin/khmoryz") != "arn:aws:iam::755952356119:role/Admin" {
		t.Errorf("Failed to transform assume-role type arn")
	}
	if TransformCallerArn("arn:aws:iam::755952356119:user/khmoryz") != "arn:aws:iam::755952356119:user/khmoryz" {
		t.Errorf("Failed to transform IAM user type arn")
	}
}

func TestGetRoleNameFromArn(t *testing.T) {
	roleArn := "arn:aws:iam::755952356119:role/my-role"
	roleName, err := GetRoleNameFromArn(roleArn)
	if err != nil {
		t.Fatal(err)
	}
	if roleName != "my-role" {
		t.Errorf("Failed to get role name from arn: %s", roleArn)
	}
}
