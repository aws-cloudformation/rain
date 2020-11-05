//+build func_test

package ec2

// GetRegions returns all region names as strings
func GetRegions() ([]string, error) {
	return []string{
		"mock-region-1",
		"mock-region-2",
		"mock-region-3",
	}, nil
}
