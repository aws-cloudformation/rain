package ec2

import (
	"context"
	"fmt"
	"sort"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func getClient() *ec2.Client {
	return ec2.NewFromConfig(rainaws.Config())
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

// CheckKeyPairExists checks to see if a key pair exists by name
func CheckKeyPairExists(name string) (bool, error) {
	res, err := getClient().DescribeKeyPairs(context.Background(), &ec2.DescribeKeyPairsInput{
		KeyNames: []string{name},
	})
	if err != nil {
		return false, err
	}

	return len(res.KeyPairs) > 0, nil
}

func GetInstanceType(instanceType string) (*types.InstanceTypeInfo, error) {
	res, err := getClient().DescribeInstanceTypes(context.Background(),
		&ec2.DescribeInstanceTypesInput{
			InstanceTypes: []types.InstanceType{types.InstanceType(instanceType)},
		})

	if err != nil {
		return nil, err
	}

	if len(res.InstanceTypes) == 0 {
		return nil, fmt.Errorf("no instance type found for %s", instanceType)
	}

	return &res.InstanceTypes[0], nil
}

func GetImage(imageID string) (*types.Image, error) {
	res, err := getClient().DescribeImages(context.Background(),
		&ec2.DescribeImagesInput{
			ImageIds: []string{imageID},
		})

	if err != nil {
		return nil, err
	}

	if len(res.Images) == 0 {
		return nil, fmt.Errorf("no image found for %s", imageID)
	}

	return &res.Images[0], nil
}

var typesByArchCache map[string][]string

func GetInstanceTypesForArchitecture(architecture string) ([]string, error) {
	var nextToken *string
	var input *ec2.DescribeInstanceTypesInput
	retval := make([]string, 0)

	if cached, ok := typesByArchCache[architecture]; ok {
		config.Debugf("GetInstanceTypesForArchitecture returning cached result for %s",
			architecture)
		return cached, nil
	}

	for {
		input = &ec2.DescribeInstanceTypesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("processor-info.supported-architecture"),
					Values: []string{architecture},
				},
			},
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		res, err := getClient().DescribeInstanceTypes(context.Background(), input)

		if err != nil {
			return nil, err
		}

		for _, instanceType := range res.InstanceTypes {
			retval = append(retval, string(instanceType.InstanceType))
		}

		if res.NextToken != nil {
			nextToken = res.NextToken
		} else {
			break
		}
	}

	config.Debugf("Instance types for architecture %s: %v", architecture, retval)

	typesByArchCache[architecture] = retval

	return retval, nil
}

// GetDefaultVPCId returns the default VPC Id, or a blank string
func GetDefaultVPCId() (string, error) {

	output, err := getClient().DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{})
	if err != nil {
		return "", err
	}

	// Find the default VPC
	var defaultVpcID string
	for _, vpc := range output.Vpcs {
		if *vpc.IsDefault {
			defaultVpcID = *vpc.VpcId
			break
		}
	}

	return defaultVpcID, nil
}

func init() {
	typesByArchCache = make(map[string][]string)
}
