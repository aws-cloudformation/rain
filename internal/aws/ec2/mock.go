//go:build func_test

package ec2

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
