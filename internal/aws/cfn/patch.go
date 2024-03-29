package cfn

import (
	"errors"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/lightsail"
)

func convertStrings(sa []string) []any {
	r := make([]any, 0)
	for _, s := range sa {
		r = append(r, s)
	}
	return r
}

func patchLightsailInstance(schema *Schema) error {
	blueprintId, found := schema.Properties["BlueprintId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Instance to have BlueprintId")
	}
	blueprints, err := lightsail.GetBlueprints()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail blueprints")
	}
	blueprintId.Enum = convertStrings(blueprints)

	bundleId, found := schema.Properties["BundleId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Instance to have BundleId")
	}
	bundles, err := lightsail.GetBundles()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail bundles")
	}
	bundleId.Enum = convertStrings(bundles)

	return nil
}

func patchLightsailBucket(schema *Schema) error {
	bundleId, found := schema.Properties["BundleId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Bucket to have BundleId")
	}
	bundles, err := lightsail.GetBucketBundles()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail bucket bundles")
	}
	bundleId.Enum = convertStrings(bundles)

	return nil
}

func patchLightsailDistribution(schema *Schema) error {
	bundleId, found := schema.Properties["BundleId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Distribution to have BundleId")
	}
	bundles, err := lightsail.GetDistributionBundles()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail distribution bundles")
	}
	bundleId.Enum = convertStrings(bundles)

	return nil
}

func patchLightsailDatabase(schema *Schema) error {
	blueprintId, found := schema.Properties["RelationalDatabaseBlueprintId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Database to have RelationalDatabaseBlueprintId")
	}
	blueprints, err := lightsail.GetRelationalDatabaseBlueprints()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail blueprints")
	}
	blueprintId.Enum = convertStrings(blueprints)

	bundleId, found := schema.Properties["RelationalDatabaseBundleId"]
	if !found {
		return fmt.Errorf("expected AWS::Lightsail::Database to have RelationalDatabaseBundleId")
	}
	bundles, err := lightsail.GetRelationalDatabaseBundles()
	if err != nil {
		return fmt.Errorf("unable to call aws api to get available lightsail bundles")
	}
	bundleId.Enum = convertStrings(bundles)

	return nil
}

func patchLightsailAlarm(schema *Schema) error {
	// These are documented but not in the schema
	valid := []string{
		"GreaterThanOrEqualToThreshold",
		"GreaterThanThreshold",
		"LessThanThreshold",
		"LessThanOrEqualToThreshold",
	}
	comparisonOperator, found := schema.Properties["ComparisonOperator"]
	if !found {
		return errors.New("expected AWS::Lightsail::Alarm to have ComparisonOperator")
	}
	comparisonOperator.Enum = convertStrings(valid)
	return nil
}

func patchSESConfigurationSetEventDestination(schema *Schema) error {
	valid := []string{
		"send",
		"reject",
		"bounce",
		"complaint",
		"delivery",
		"open",
		"click",
		"renderingFailure",
		"deliveryDelay",
		"subscription",
	}
	eventDest, found := schema.Definitions["EventDestination"]
	if !found {
		return errors.New("expected AWS::SES::ConfigurationSetEventDestination to have EventDestination")
	}
	eventTypes, found := eventDest.Properties["MatchingEventTypes"]
	if !found {
		return errors.New("expected AWS::SES::ConfigurationSetEventDestination.EventDestination to have MatchingEventTypes")
	}
	eventTypes.Items.Enum = convertStrings(valid)
	return nil
}

func patchSESContactList(schema *Schema) error {
	valid := []string{"OPT_IN", "OPT_OUT"}

	topic, found := schema.Definitions["Topic"]
	if !found {
		return errors.New("expected AWS::SES::ContactList to have Topic")
	}

	dss, found := topic.Properties["DefaultSubscriptionStatus"]
	if !found {
		return errors.New("expected AWS::SES::ContactList to have Topic.DefaultSubscriptionStatus")
	}
	dss.Enum = convertStrings(valid)
	return nil
}

func patchIAMRole(schema *Schema) error {
	policy, found := schema.Definitions["Policy"]
	if !found {
		return errors.New("expected AWS::IAM::Role to have Policy")
	}
	policyDocument, found := policy.Properties["PolicyDocument"]
	if !found {
		return errors.New("expected AWS::IAM::Role to have Policy.PolicyDocument")
	}
	policyDocument.Type = "object"
	arpd, found := schema.Properties["AssumeRolePolicyDocument"]
	if !found {
		return errors.New("expected AWS::IAM::Role to have AssumeRolePolicyDocument")
	}
	arpd.Type = "object"
	return nil
}

func patchDynamoDBTable(schema *Schema) error {
	keySchema, found := schema.Properties["KeySchema"]
	if !found {
		return errors.New("expected AWS::DynamoDB::Table to have KeySchema")
	}
	if len(keySchema.OneOf) != 2 {
		return errors.New("expected AWS::DynamoDB::Table.KeySchema to be oneOf")
	}
	// Replace the property with the correct, documented option, and
	// get rid of the "object" oneOf option[1]
	*keySchema = *keySchema.OneOf[0]
	return nil
}

func patchAEC2VerifiedAccessTrustProvider(schema *Schema) error {
	// This one is apparently a placeholder schema
	// Remove the extraneous def that points to itself
	delete(schema.Definitions, "SseSpecification")
	return nil
}
