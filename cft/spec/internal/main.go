// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft/spec"

	"gopkg.in/yaml.v3"
)

const (
	cfnSpecURL = "https://d1uauaxba7bl26.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json"
	cfnSpecFn  = "internal/CloudFormationResourceSpecification.json"
	iamSpecFn  = "internal/IamSpecification.json"
	samSpecFn  = "internal/SamSpecification.json"
)

func load(r io.Reader, s *spec.Spec) {
	var intermediate map[string]interface{}
	yamlDecoder := yaml.NewDecoder(r)
	err := yamlDecoder.Decode(&intermediate)
	if err != nil {
		panic(err)
	}

	inJSON, err := json.Marshal(intermediate)
	if err != nil {
		panic(err)
	}

	jsonReader := bytes.NewReader(inJSON)

	decoder := json.NewDecoder(jsonReader)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&s)
	if err != nil {
		panic(err)
	}
}

func loadURL(url string) spec.Spec {
	var s spec.Spec

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	load(resp.Body, &s)

	return s
}

func loadFile(fn string) spec.Spec {
	var s spec.Spec

	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	load(f, &s)

	return s
}

func saveSpec(s spec.Spec, name string) {
	// Write out the file
	source := fmt.Sprintf(`package spec

// %s is generated from the specification file
var %s = %s`, name, name, s)

	out, err := format.Source([]byte(source))
	if err != nil {
		fmt.Println(source)
		panic(err)
	}

	err = ioutil.WriteFile(strings.ToLower(name)+".go", out, 0644)
	if err != nil {
		panic(err)
	}
}

func saveJSON(s spec.Spec, name string) {
	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(name, data, 0644)
	if err != nil {
		panic(err)
	}
}

func patchCfnSpec(s spec.Spec) {
	for _, pt := range s.PropertyTypes {
		for _, p := range pt.Properties {
			if p.ItemType == "Json" {
				p.ItemType = ""
				p.PrimitiveItemType = "Json"
			}
		}
	}

	s.ResourceTypes["AWS::ECR::Repository"].Properties["RepositoryPolicyText"].PrimitiveType = "Json"

	s.ResourceTypes["AWS::IoT::ProvisioningTemplate"].Properties["Tags"].PrimitiveItemType = "Json"
	s.ResourceTypes["AWS::IoT::ProvisioningTemplate"].Properties["Tags"].Type = "List"

	s.ResourceTypes["AWS::KMS::Key"].Properties["KeyPolicy"].PrimitiveType = "Json"
}

func patchSamSpec(s spec.Spec) {
	s.PropertyTypes["AWS::Serverless::Api.ApiUsagePlan"].Properties["Quota"].Type = "AWS::ApiGateway::UsagePlan.QuotaSettings"
	s.PropertyTypes["AWS::Serverless::Api.ApiUsagePlan"].Properties["Throttle"].Type = "AWS::ApiGateway::UsagePlan.ThrottleSettings"
	s.ResourceTypes["AWS::Serverless::Api"].Properties["AccessLogSetting"].Type = "AWS::ApiGateway::Stage.AccessLogSetting"
	s.ResourceTypes["AWS::Serverless::Api"].Properties["CanarySetting"].Type = "AWS::ApiGateway::Stage.CanarySetting"
	s.ResourceTypes["AWS::Serverless::Api"].Properties["MethodSettings"].ItemType = "AWS::ApiGateway::Stage.MethodSetting"
	s.ResourceTypes["AWS::Serverless::Api"].Properties["MethodSettings"].Type = "List"

	s.ResourceTypes["AWS::Serverless::Function"].Properties["Environment"].Type = "AWS::Lambda::Function.Environment"
	s.ResourceTypes["AWS::Serverless::Function"].Properties["ProvisionedConcurrencyConfig"].Type = "AWS::Lambda::Alias.ProvisionedConcurrencyConfiguration"
	s.ResourceTypes["AWS::Serverless::Function"].Properties["VpcConfig"].Type = "AWS::Lambda::Function.VpcConfig"

	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["AccessLogSettings"].Type = "AWS::ApiGatewayV2::Stage.AccessLogSettings"
	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["DefaultRouteSettings"].Type = "AWS::ApiGatewayV2::Stage.RouteSettings"
	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["RouteSettings"].Type = "AWS::ApiGatewayV2::Stage.RouteSettings"

	s.ResourceTypes["AWS::Serverless::SimpleTable"].Properties["ProvisionedThroughput"].Type = "AWS::DynamoDB::Table.ProvisionedThroughput"
	s.ResourceTypes["AWS::Serverless::SimpleTable"].Properties["SSESpecification"].Type = "AWS::DynamoDB::Table.SSESpecification"

	s.ResourceTypes["AWS::Serverless::StateMachine"].Properties["DefinitionUri"].Type = "AWS::StepFunctions::StateMachine.S3Location"
	s.ResourceTypes["AWS::Serverless::StateMachine"].Properties["Logging"].Type = "AWS::StepFunctions::StateMachine.LoggingConfiguration"
	s.ResourceTypes["AWS::Serverless::StateMachine"].Properties["Tracing"].Type = "AWS::StepFunctions::StateMachine.TracingConfiguration"
}

func main() {
	// Merge cfn and sam specs
	cfnSpec := loadURL(cfnSpecURL)
	patchCfnSpec(cfnSpec)
	saveJSON(cfnSpec, cfnSpecFn)

	samSpec := loadFile(samSpecFn)
	patchSamSpec(samSpec)
	saveJSON(samSpec, samSpecFn)

	for name, res := range samSpec.ResourceTypes {
		cfnSpec.ResourceTypes[name] = res
	}

	for name, prop := range samSpec.PropertyTypes {
		cfnSpec.PropertyTypes[name] = prop
	}

	// Save specs
	saveSpec(cfnSpec, "Cfn")
	saveSpec(loadFile(iamSpecFn), "Iam")

	// Clean up
	os.Remove(cfnSpecFn)
	os.Remove(samSpecFn)
}
