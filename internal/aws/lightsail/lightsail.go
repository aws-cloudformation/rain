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
