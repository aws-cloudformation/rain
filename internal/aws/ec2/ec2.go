//go:build !func_test

package ec2

import (
	"context"
	"sort"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func getClient() *ec2.Client {
	return ec2.NewFromConfig(aws.Config())
}

// GetRegions returns all region names as strings
func GetRegions() ([]string, error) {
	res, err := getClient().DescribeRegions(context.Background(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}

	regions := make([]string, len(res.Regions))
	for i, region := range res.Regions {
		regions[i] = *region.RegionName
	}

	sort.Strings(regions)

	return regions, nil
}
