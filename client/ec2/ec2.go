package ec2

import (
	"context"
	"sort"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func getClient() *ec2.Client {
	return ec2.New(client.Config())
}

func GetRegions() ([]string, client.Error) {
	req := getClient().DescribeRegionsRequest(&ec2.DescribeRegionsInput{})

	res, err := req.Send(context.Background())
	if err != nil {
		return nil, client.NewError(err)
	}

	regions := make([]string, len(res.Regions))
	for i, region := range res.Regions {
		regions[i] = *region.RegionName
	}

	sort.Strings(regions)

	return regions, nil
}
