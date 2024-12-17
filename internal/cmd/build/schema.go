package build

import (
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

func getSchema(typeName string) (*cfn.Schema, error) {

	var schema *cfn.Schema
	var err error

	// Check to see if it's a SAM resource
	if isSAM(typeName) {

		// Convert the spec to a cfn.Schema and skip downloading from the registry
		schema, err = convertSAMSpec(samSpecSource, typeName)
		if err != nil {
			return nil, err
		}

	} else {

		// Call CCAPI to get the schema for the resource
		schemaSource, err := cfn.GetTypeSchema(typeName, getCacheUsage())
		if err != nil {
			return nil, err
		}

		// Parse the schema JSON string
		schema, err = cfn.ParseSchema(schemaSource)
		if err != nil {
			return nil, err
		}
	}

	// Apply patches to the schema
	if !omitPatches {
		err := schema.Patch()
		if err != nil {
			return nil, err
		}
	}

	return schema, nil
}
