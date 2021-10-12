//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
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

		fmt.Print(".")

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

			fmt.Print(".")
		}

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}
}

func main() {
	schemas := make(map[string]map[string]interface{})

	getThirdPartyTypes(schemas)
	getFirstPartyTypes(schemas)

	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile("schemas.json", data, 0644)

	source := fmt.Sprintf(`package spec

// Cfn is generated from CloudFormation specifications
var Cfn = %s`, formatMap(schemas))

	result, err := format.Source([]byte(source))
	if err != nil {
		fmt.Println(source)
		panic(err)
	}

	err = ioutil.WriteFile("cfn.go", result, 0644)
	if err != nil {
		panic(err)
	}
}
