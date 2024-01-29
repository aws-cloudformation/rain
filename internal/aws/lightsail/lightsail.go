package lightsail

import (
	"context"
	"errors"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
)

func getClient() *lightsail.Client {
	return lightsail.NewFromConfig(rainaws.Config())
}

// GetBlueprints gets all available lightsail instance blueprints in this region
func GetBlueprints() ([]string, error) {
	var nextPageToken *string
	retval := make([]string, 0)
	for sanity := 0; sanity < 10; sanity += 1 {
		res, err := getClient().GetBlueprints(context.Background(),
			&lightsail.GetBlueprintsInput{PageToken: nextPageToken})
		if err != nil {
			return nil, err
		}

		for _, b := range res.Blueprints {
			retval = append(retval, *b.BlueprintId)
		}

		if res.NextPageToken == nil {
			return retval, nil
		}
		nextPageToken = res.NextPageToken
	}
	return nil, errors.New("unexpected: GetBlueprints API called 10 times")
}

// GetBundles gets all available lightsail instance bundles in this region
func GetBundles() ([]string, error) {
	var nextPageToken *string
	retval := make([]string, 0)
	for sanity := 0; sanity < 10; sanity += 1 {
		res, err := getClient().GetBundles(context.Background(),
			&lightsail.GetBundlesInput{PageToken: nextPageToken})
		if err != nil {
			return nil, err
		}

		for _, b := range res.Bundles {
			retval = append(retval, *b.BundleId)
		}

		if res.NextPageToken == nil {
			return retval, nil
		}
		nextPageToken = res.NextPageToken
	}
	return nil, errors.New("unexpected: GetBundles API called 10 times")
}

// GetRelationalDatabaseBlueprints gets all available lightsail database blueprints in this region
func GetRelationalDatabaseBlueprints() ([]string, error) {
	var nextPageToken *string
	retval := make([]string, 0)
	for sanity := 0; sanity < 10; sanity += 1 {
		res, err := getClient().GetRelationalDatabaseBlueprints(context.Background(),
			&lightsail.GetRelationalDatabaseBlueprintsInput{PageToken: nextPageToken})
		if err != nil {
			return nil, err
		}

		for _, b := range res.Blueprints {
			retval = append(retval, *b.BlueprintId)
		}

		if res.NextPageToken == nil {
			return retval, nil
		}
		nextPageToken = res.NextPageToken
	}
	return nil, errors.New("unexpected: GetRelationalDatabaseBlueprints API called 10 times")
}

// GetRelationalDatabaseBundles gets all available lightsail database bundles in this region
func GetRelationalDatabaseBundles() ([]string, error) {
	var nextPageToken *string
	retval := make([]string, 0)
	for sanity := 0; sanity < 10; sanity += 1 {
		res, err := getClient().GetRelationalDatabaseBundles(context.Background(),
			&lightsail.GetRelationalDatabaseBundlesInput{PageToken: nextPageToken})
		if err != nil {
			return nil, err
		}

		for _, b := range res.Bundles {
			retval = append(retval, *b.BundleId)
		}

		if res.NextPageToken == nil {
			return retval, nil
		}
		nextPageToken = res.NextPageToken
	}
	return nil, errors.New("unexpected: GetRelationalDatabaseBundles API called 10 times")
}

// GetBucketBundles gets all available lightsail bucket bundles in this region
func GetBucketBundles() ([]string, error) {
	retval := make([]string, 0)
	res, err := getClient().GetBucketBundles(context.Background(),
		&lightsail.GetBucketBundlesInput{})
	if err != nil {
		return nil, err
	}

	for _, b := range res.Bundles {
		retval = append(retval, *b.BundleId)
	}

	return retval, nil
}

// GetDistributionBundles gets all available lightsail distribution bundles in this region
func GetDistributionBundles() ([]string, error) {
	retval := make([]string, 0)
	res, err := getClient().GetDistributionBundles(context.Background(),
		&lightsail.GetDistributionBundlesInput{})
	if err != nil {
		return nil, err
	}

	for _, b := range res.Bundles {
		retval = append(retval, *b.BundleId)
	}

	return retval, nil
}
