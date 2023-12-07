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
