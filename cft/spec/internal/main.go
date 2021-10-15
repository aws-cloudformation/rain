//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws-cloudformation/rain/cft/spec"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const schemaUri = "https://schema.cloudformation.us-east-1.amazonaws.com/CloudformationSchema.zip"

func getFirstPartyTypes(schemas map[string]map[string]interface{}) {
	resp, err := http.Get(schemaUri)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()

	br := bytes.NewReader(body)
	r, err := zip.NewReader(br, br.Size())
	if err != nil {
		panic(err)
	}

	for _, f := range r.File {
		fr, err := f.Open()
		if err != nil {
			panic(err)
		}

		fb, err := ioutil.ReadAll(fr)
		if err != nil {
			panic(err)
		}
		fr.Close()

		var schema map[string]interface{}
		err = json.Unmarshal(fb, &schema)
		if err != nil {
			panic(err)
		}

		schemas[schema["typeName"].(string)] = schema
	}
}

func getThirdPartyTypes(schemas map[string]map[string]interface{}) {
	client := cloudformation.NewFromConfig(aws.Config())

	var token *string

	for {
		res, err := client.ListTypes(context.Background(), &cloudformation.ListTypesInput{
			NextToken: token,
			Filters: &types.TypeFilters{
				Category: types.CategoryThirdParty,
			},
			Visibility: types.VisibilityPublic,
		})

		if err != nil {
			panic(err)
		}

		for _, summary := range res.TypeSummaries {
			desc, err := client.DescribeType(context.Background(), &cloudformation.DescribeTypeInput{
				Arn: summary.TypeArn,
			})
			if err != nil {
				panic(err)
			}

			var schema map[string]interface{}
			err = json.Unmarshal([]byte(ptr.ToString(desc.Schema)), &schema)
			if err != nil {
				panic(err)
			}

			schemas[ptr.ToString(summary.TypeName)] = schema

			time.Sleep(time.Second / 2)
		}

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}
}

// stripPatterns removes custom regexes
// as the cfn-spec uses extended regexes
// which the jsonschema package does not support
func stripPatterns(in interface{}) {
	switch v := in.(type) {
	case spec.Schema:
		stripPatterns(map[string]interface{}(v))
	case map[string]interface{}:
		delete(v, "patternProperties")
		for key, value := range v {
			if key == "pattern" || key == "format" {
				v[key] = ".*"
			} else {
				stripPatterns(value)
			}
		}
	case []interface{}:
		for _, child := range v {
			stripPatterns(child)
		}
	}
}

func main() {
	schemas := make(map[string]map[string]interface{})

	getThirdPartyTypes(schemas)
	getFirstPartyTypes(schemas)

	/*
		schemas = map[string]map[string]interface{}{
			"AWS::S3::Bucket": schemas["AWS::S3::Bucket"],
		}
	*/

	// Ensure type name is in place
	for typeName, schema := range schemas {
		schema["typeName"] = typeName
	}

	// Test that the schemas validate correctly
	for typeName, schema := range spec.Cfn {
		stripPatterns(schema)

		data, err := json.Marshal(schema)
		if err != nil {
			panic(fmt.Errorf("%s: %w", typeName, err))
		}

		_, err = jsonschema.CompileString("schema.json", string(data))
		if err != nil {
			panic(fmt.Errorf("%s: %w", typeName, err))
		}
	}

	// Write out as JSON
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile("schemas.json", data, 0644)

	for key := range schemas {
		fmt.Println(key)
	}
}
