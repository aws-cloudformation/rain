// Package codeartifact provides functions that interact with the AWS CodeArtifact API
package codeartifact

import (
	"context"
	"errors"
	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/smithy-go"
)
import "github.com/aws/aws-sdk-go-v2/service/codeartifact"

func getClient() *codeartifact.Client {
	return codeartifact.NewFromConfig(rainaws.Config())
}

// DomainExists checks if a domain exists
func DomainExists(name string) (bool, error) {
	client := getClient()
	res, err := client.DescribeDomain(context.Background(),
		&codeartifact.DescribeDomainInput{Domain: &name})
	if err != nil {
		// Check to see if this is a ResourceNotFoundException
		var ae smithy.APIError
		if errors.As(err, &ae) {
			config.Debugf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())

			if ae.ErrorCode() == "ResourceNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}
	return res.Domain != nil, nil
}

// CreateDomain creates a domain
func CreateDomain(name string) error {
	client := getClient()
	_, err := client.CreateDomain(context.Background(),
		&codeartifact.CreateDomainInput{Domain: &name})
	return err
}

// RepoExists checks if a repo exists
func RepoExists(name string, domain string) (bool, error) {
	client := getClient()
	res, err := client.DescribeRepository(context.Background(),
		&codeartifact.DescribeRepositoryInput{Domain: &domain, Repository: &name})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			config.Debugf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())

			if ae.ErrorCode() == "ResourceNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}
	return res.Repository != nil, nil
}

// CreateRepo creates a repo
func CreateRepo(name string, domain string) error {
	client := getClient()
	_, err := client.CreateRepository(context.Background(),
		&codeartifact.CreateRepositoryInput{Domain: &domain, Repository: &name})
	return err
}

//func Publish() error {
//
// Publish the package
/*
	aws codeartifact publish-package-version --domain nissan --repository templates \
	      --format generic --namespace my-ns --package my-package --package-version 1.0.0 \
	      --asset-content modules.zip --asset-name modules.zip \
	      --asset-sha256 $ASSET_SHA256
*/
//}
//
//func Install() error {
//
//}
