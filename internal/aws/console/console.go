package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/sts"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/smithy-go/ptr"
)

var signinURI string
var consoleURI string

const signoutURI = "https://signin.aws.amazon.com/oauth?Action=logout&redirect_uri=https://aws.amazon.com"
const issuer = "https://aws-cloudformation.github.io/rain/rain_console.html"
const defaultService = "cloudformation"
const sessionDuration = 43200

func buildSessionString(sessionName string) (string, error) {
	if sessionName == "" {
		id, err := sts.GetCallerID()
		if err != nil {
			return "", err
		}

		idParts := strings.Split(ptr.ToString(id.Arn), ":")
		nameParts := strings.Split(idParts[len(idParts)-1], "/")

		if nameParts[0] == "user" {
			panic(errors.New("sign-in URLs can only be constructed for assumed roles"))
		}

		sessionName = nameParts[1]
	}

	config.Debugf("sessionName: %v", sessionName)

	creds, err := aws.NamedConfig(sessionName).Credentials.Retrieve(context.Background())
	if err != nil {
		return "", err
	}

	unescaped := fmt.Sprintf(`{"sessionId": "%s", "sessionKey": "%s", "sessionToken": "%s"}`,
		creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)

	config.Debugf("unescaped session string: %v", unescaped)

	return url.QueryEscape(unescaped), nil
}

func getSigninToken(userName string) (string, error) {
	sessionString, err := buildSessionString(userName)
	if err != nil {
		config.Debugf("buildSessionString failed")
		return "", err
	}
	config.Debugf("sessionString: %v", sessionString)

	// Broken with source_profile and a role arn in .aws/config
	uriWithSessionDuration := fmt.Sprintf("%s?Action=getSigninToken&Session=%s&SessionDuration=%d",
		signinURI, sessionString, sessionDuration)

	// Try it without session duration (console sessions will be limited to 1 hour)
	uriWithoutSessionDuration := fmt.Sprintf("%s?Action=getSigninToken&Session=%s",
		signinURI, sessionString)

	// SessionDuration is only valid when AssumeRole
	// is called, so when source_profile is used, it must cause a call to
	// GetFederationToken, which would require the use of DurationSeconds.
	// It's not clear how we could predict reliably which one will be used,
	// so we try both URIs and see which one works. Not ideal.

	// This page provides a good explanation of what we're doing here:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_enable-console-custom-url.html

	resp, err := http.Get(uriWithSessionDuration)
	config.Debugf("resp.StatusCode: %v", resp.StatusCode)
	if resp.StatusCode >= 300 && err == nil {
		config.Debugf("Retrying without SessionDuration after call to signin.aws.amazon.com resulted in a %v: %v",
			resp.StatusCode, resp.Status)
		resp, err = http.Get(uriWithoutSessionDuration)
		if resp.StatusCode >= 300 && err == nil {
			err = fmt.Errorf("call to signin.aws.amazon.com resulted in a %v: %v", resp.StatusCode, resp.Status)
		}
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	config.Debugf("body: %s", body)

	var out map[string]string
	err = json.Unmarshal(body, &out)
	if err != nil {
		return "", err
	}

	token, ok := out["SigninToken"]
	if !ok {
		return "", errors.New("no token present in the response")
	}

	return token, nil
}

// GetURI returns a sign-in uri for the current credentials and region
func GetURI(logout bool, service, stackName, userName string) (string, error) {
	if logout {
		return signoutURI, nil
	}

	config.Debugf("GetURI %v, %v, %v", service, stackName, userName)

	region := aws.Config().Region

	signinURI = "https://signin.aws.amazon.com/federation"
	consoleURI = "https://console.aws.amazon.com"

	// Different URIs for GovCloud
	if strings.HasPrefix(region, "us-gov-") {
		signinURI = "https://signin.amazonaws-us-gov.com/federation"
		consoleURI = "https://console.amazonaws-us-gov.com"
	}

	token, err := getSigninToken(userName)
	if err != nil {
		config.Debugf("getSigninToken failed")
		return "", err
	}

	config.Debugf("token: %v", token)

	if service == "" {
		service = defaultService
	}

	destination := fmt.Sprintf("%s/%s/home?region=%s", consoleURI, service, aws.Config().Region)

	if service == defaultService && stackName != "" {
		if stack, err := cfn.GetStack(stackName); err == nil {
			if stack.StackId != nil {
				destination += fmt.Sprintf("#/stacks/stackinfo?stackId=%s&hideStacks=false&viewNested=true",
					ptr.ToString(stack.StackId),
				)
			}
		}
	}

	return fmt.Sprintf("%s?Action=login&Issuer=%s&Destination=%s&SigninToken=%s",
		signinURI,
		url.QueryEscape(issuer),
		url.QueryEscape(destination),
		url.QueryEscape(token),
	), nil
}
