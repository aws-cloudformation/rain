package rds

import (
	"context"

	aws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

func getClient() *rds.Client {
	return rds.NewFromConfig(aws.Config())
}

func GetValidEngineVersions(engine string) ([]string, error) {
	retval := make([]string, 0)

	res, err := getClient().DescribeDBEngineVersions(context.Background(),
		&rds.DescribeDBEngineVersionsInput{Engine: &engine})

	if err != nil {
		return retval, err
	}

	for _, v := range res.DBEngineVersions {
		retval = append(retval, *v.EngineVersion)
	}

	return retval, nil
}

func GetNumClusters() (int, error) {
	res, err := getClient().DescribeDBClusters(context.Background(),
		&rds.DescribeDBClustersInput{})
	if err != nil {
		return -1, err
	}
	return len(res.DBClusters), nil
}
