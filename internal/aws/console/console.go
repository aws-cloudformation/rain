package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/awslabs/smithy-go/ptr"
)

const signinURI = "https://signin.aws.amazon.com/federation"
const issuer = "https://github.com/aws-cloudformation/rain"
const consoleURI = "https://console.aws.amazon.com"
const defaultService = "cloudformation"
const sessionDuration = 43200

func buildSessionString() (string, error) {
	creds, err := aws.Config().Credentials.Retrieve(context.Background())
	if err != nil {
		return "", err
	}

	if creds.SessionToken == "" {
		panic(errors.New("sign-in URLs can only be constructed for assumed roles"))
	}

	return url.QueryEscape(fmt.Sprintf(`{"sessionId": "%s", "sessionKey": "%s", "sessionToken": "%s"}`,
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)), nil
}

func getSigninToken() (string, error) {
	sessionString, err := buildSessionString()
	if err != nil {
		return "", err
	}

	resp, err := http.Get(fmt.Sprintf("%s?Action=getSigninToken&Session=%s&SessionDuration=%d", signinURI, sessionString, sessionDuration))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var out map[string]string
	err = json.Unmarshal(body, &out)
	if err != nil {
		return "", err
	}

	token, ok := out["SigninToken"]
	if !ok {
		return "", errors.New("No token present in the response")
	}

	return token, nil
}

// GetURI returns a sign-in uri for the current credentials and region
func GetURI(service, stackName string) (string, error) {
	token, err := getSigninToken()
	if err != nil {
		return "", err
	}

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
