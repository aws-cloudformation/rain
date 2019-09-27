package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/spec/models"
)

const (
	cfnSpecUrl = "https://d2senuesg1djtx.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json"
	cfnSpecFn  = "generate/CloudFormationResourceSpecification.json"
	iamSpecFn  = "generate/IamSpecification.json"
)

func load(r io.Reader, s *models.Spec) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&s)
	if err != nil {
		panic(err)
	}
}

func loadUrl(url string) models.Spec {
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

var %s = %s`, name, s)

	out, err := format.Source([]byte(source))
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(strings.ToLower(name)+".go", out, 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	//saveSpec(loadUrl(cfnSpecUrl), "Cfn")
	saveSpec(loadFile(cfnSpecFn), "Cfn")
	saveSpec(loadFile(iamSpecFn), "Iam")
}
