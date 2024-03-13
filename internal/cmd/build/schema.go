package build

import (
	"encoding/json"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
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

		j, _ := json.Marshal(schema)
		config.Debugf("Converted SAM schema: %s", j)

	} else {

		// Call CCAPI to get the schema for the resource
		schemaSource, err := cfn.GetTypeSchema(typeName)
		config.Debugf("schema source: %s", schemaSource)
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
		j, _ := json.MarshalIndent(schema, "", "    ")
		config.Debugf("patched schema: %s", j)
	}

	return schema, nil
}
