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

	"gopkg.in/yaml.v3"
)

const (
	cfnSpecURL = "https://d1uauaxba7bl26.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json"
	cfnSpecFn  = "generate/CloudFormationResourceSpecification.json"
	iamSpecFn  = "generate/IamSpecification.json"
	samSpecFn  = "generate/SamSpecification.json"
)

func load(r io.Reader, s *models.Spec) {
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
	// Any fixes required for bugs in spec - None as at 2020-10-20
}

func patchSamSpec(s models.Spec) {
	// Any fixes required for bugs in spec - None as at 2020-10-20
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
