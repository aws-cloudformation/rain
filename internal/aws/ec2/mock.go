//go:build func_test

package ec2

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

// GetRegions returns all region names as strings
func GetRegions() ([]string, error) {
	return []string{
		"mock-region-1",
		"mock-region-2",
		"mock-region-3",
	}, nil
}

func CheckKeyPairExists(name string) (bool, error) {
	return true, nil
}

func GetImage(imageID string) (*types.Image, error) {
	return nil, nil
}

func GetInstanceType(instanceType string) (*types.InstanceTypeInfo, error) {
	return nil, nil
}

func GetInstanceTypesForArchitecture(architecture string) ([]string, error) {
	return nil, nil
}

func GetDefaultVPCId() (string, error) {
	return "", nil
}
