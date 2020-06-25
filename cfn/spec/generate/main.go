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

	"github.com/aws-cloudformation/rain/cfn/spec/models"
	yamlwrapper "github.com/sanathkr/yaml"
)

const (
	cfnSpecURL = "https://d1uauaxba7bl26.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json"
	cfnSpecFn  = "generate/CloudFormationResourceSpecification.json"
	iamSpecFn  = "generate/IamSpecification.json"
	samSpecFn  = "generate/SamSpecification.json"
)

func load(r io.Reader, s *models.Spec) {
	inYAML, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	inJSON, err := yamlwrapper.YAMLToJSON(inYAML)
	if err != nil {
		panic(err)
	}

	yamlReader := bytes.NewReader(inJSON)

	decoder := json.NewDecoder(yamlReader)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&s)
	if err != nil {
		panic(err)
	}
}

func loadURL(url string) models.Spec {
	var s models.Spec

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	load(resp.Body, &s)

	return s
}

func loadFile(fn string) models.Spec {
	var s models.Spec

	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	load(f, &s)

	return s
}

func saveSpec(s models.Spec, name string) {
	// Write out the file
	source := fmt.Sprintf(`package spec

import "github.com/aws-cloudformation/rain/cfn/spec/models"

// %s is generated from the specification file
var %s = %s`, name, name, s)

	out, err := format.Source([]byte(source))
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(strings.ToLower(name)+".go", out, 0644)
	if err != nil {
		panic(err)
	}
}

func saveJSON(s models.Spec, name string) {
	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(name, data, 0644)
	if err != nil {
		panic(err)
	}
}

func patchCfnSpec(s models.Spec) {
	s.PropertyTypes["AWS::SSM::Association.ParameterValues"] = models.PropertyType{
		Property: models.Property{
			Type:              "List",
			PrimitiveItemType: "String",
		},
	}

	s.ResourceTypes["AWS::IoT::ProvisioningTemplate"].Properties["Tags"] = models.Property{
		Type:              "List",
		PrimitiveItemType: "Json",
	}
}

func patchSamSpec(s models.Spec) {
	s.ResourceTypes["AWS::Serverless::Api"].Properties["MethodSettings"] = models.Property{
		Type:     "List",
		ItemType: "AWS::ApiGateway::Stage.MethodSetting",
	}

	s.ResourceTypes["AWS::Serverless::Api"].Properties["CanarySetting"] = models.Property{
		Type: "AWS::ApiGateway::Stage.CanarySetting",
	}

	s.ResourceTypes["AWS::Serverless::Api"].Properties["AccessLogSetting"] = models.Property{
		Type: "AWS::ApiGateway::Stage.AccessLogSetting",
	}

	s.PropertyTypes["AWS::Serverless::Api.ApiUsagePlan"].Properties["Throttle"] = models.Property{
		Type: "AWS::ApiGateway::UsagePlan.ThrottleSettings",
	}

	s.PropertyTypes["AWS::Serverless::Api.ApiUsagePlan"].Properties["Quota"] = models.Property{
		Type: "AWS::ApiGateway::UsagePlan.QuotaSettings",
	}

	s.PropertyTypes["AWS::Serverless::Function.EventSource"].Properties["Properties"] = models.Property{
		PrimitiveType: "Json",
	}

	s.ResourceTypes["AWS::Serverless::Function"].Properties["ProvisionedConcurrencyConfig"] = models.Property{
		Type: "AWS::Lambda::Alias.ProvisionedConcurrencyConfiguration",
	}

	s.ResourceTypes["AWS::Serverless::Function"].Properties["VpcConfig"] = models.Property{
		Type: "AWS::Lambda::Function.VpcConfig",
	}

	s.ResourceTypes["AWS::Serverless::Function"].Properties["Environment"] = models.Property{
		Type: "AWS::Lambda::Function.Environment",
	}

	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["AccessLogSettings"] = models.Property{
		Type: "AWS::ApiGatewayV2::Stage.AccessLogSettings",
	}

	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["DefaultRouteSettings"] = models.Property{
		Type: "AWS::ApiGatewayV2::Stage.RouteSettings",
	}

	s.ResourceTypes["AWS::Serverless::HttpApi"].Properties["RouteSettings"] = models.Property{
		Type: "AWS::ApiGatewayV2::Stage.RouteSettings",
	}

	s.ResourceTypes["AWS::Serverless::SimpleTable"].Properties["ProvisionedThroughput"] = models.Property{
		Type: "AWS::DynamoDB::Table.ProvisionedThroughput",
	}

	s.ResourceTypes["AWS::Serverless::SimpleTable"].Properties["SSESpecification"] = models.Property{
		Type: "AWS::DynamoDB::Table.SSESpecification",
	}

	s.ResourceTypes["AWS::Serverless::StateMachine"].Properties["DefinitionUri"] = models.Property{
		Type: "AWS::StepFunctions::StateMachine.S3Location",
	}

	s.ResourceTypes["AWS::Serverless::StateMachine"].Properties["Logging"] = models.Property{
		Type: "AWS::StepFunctions::StateMachine.LoggingConfiguration",
	}
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

	saveSpec(cfnSpec, "Cfn")

	saveSpec(loadFile(iamSpecFn), "Iam")
}
