package main

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

/*

	"github.com/dgrijalva/jwt-go"
	replaced by github.com/golang-jwt/jwt

	"github.com/lestrrat-go/jwx/jwk"

	github.com/lestrrat-go/jwx/v2/jwk

	Looks like this lib does JWT also... ? Why both?

*/

func HandleRequest(ctx context.Context,
	request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("request: %+v\n", request)
	message := fmt.Sprintf("{\"message\": \"Request Resource: %s, Path: %s, HTTPMethod: %s\"}", request.Resource, request.Path, request.HTTPMethod)
	fmt.Printf("message: %s\n", message)

	headers := make(map[string]string)
	code := request.QueryStringParameters["code"]
	refresh := request.QueryStringParameters["refresh"]

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       "{\"message\": \"Success\"}",
	}

	addResponseHeaders(response)

	switch request.HTTPMethod {
	case "GET":
		// TODO
		return response, nil
	case "OPTIONS":
		jsonData, err := handleOptionsRequest(code, refresh)
		if err != nil {
			fmt.Printf("handleOptionsRequest: %v", err)
			return fail(400, "Unable to handle OPTIONS request"), nil
		}
		response.StatusCode = 204
		response.Body = jsonData
		return response, nil
	default:
		return fail(400, fmt.Sprintf("Unexpected HttpMethod: %s", request.HTTPMethod)), nil
	}

}

func addResponseHeaders(response events.APIGatewayProxyResponse) {
	// Put CORS headers on all responses
	response.Headers["Access-Control-Allow-Headers"] = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,X-Amz-User-Agent,X-KG-Partition'"
	response.Headers["Access-Control-Allow-Origin"] = "'*'"
	response.Headers["Access-Control-Allow-Methods"] = "'OPTIONS,GET,PUT,POST,DELETE,PATCH,HEAD'"
}

func fail(code int, msg string) events.APIGatewayProxyResponse {
	response := events.APIGatewayProxyResponse{
		StatusCode: code,
		Body:       "{\"message\": \"" + msg + "\"}",
	}
	response.Headers = make(map[string]string)
	addResponseHeaders(response)
	response.Headers["X-Rain-Webapp-Error"] = msg
	return response
}

func main() {
	lambda.Start(HandleRequest)
}

// LLM output

// getCognitoIssuer returns the Cognito issuer URL.
func getCognitoIssuer() (string, error) {
	region := os.Getenv("COGNITO_REGION")
	if region == "" {
		return "", errors.New("missing COGNITO_REGION")
	}

	cognitoPoolID := os.Getenv("COGNITO_POOL_ID")
	if cognitoPoolID == "" {
		return "", errors.New("missing COGNITO_POOL_ID")
	}

	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, cognitoPoolID), nil
}

// getPublicKeys retrieves the public keys from the Cognito issuer.
func getPublicKeys() (jwk.Set, error) {

	cognitoIssuer, err := getCognitoIssuer()
	if err != nil {
		return nil, err
	}

	url := cognitoIssuer + "/.well-known/jwks.json"
	fmt.Printf("JWK URL: %s\n", url)

	set, err := jwk.Fetch(context.Background(), url)
	if err != nil {
		fmt.Printf("failed to fetch JWK: %s\n", err)
		return nil, err
	}

	// Key sets can be serialized back to JSON
	{
		jsonbuf, err := json.Marshal(set)
		if err != nil {
			log.Printf("failed to marshal key set into JSON: %s\n", err)
			return nil, err
		}
		fmt.Printf("json jwk: %s\n", jsonbuf)
	}

	return set, nil
}

// verify verifies the JWT token.
/*
func verify(t string) (map[string]interface{}, error) {

	keys, err := getPublicKeys()
	if err != nil {
		return nil, err
	}

	tokenSections := strings.Split(t, ".")
	if len(tokenSections) < 2 {
		return nil, errors.New("requested token is invalid")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(tokenSections[0])
	if err != nil {
		return nil, err
	}

	var header map[string]interface{}
	err = json.Unmarshal(headerJSON, &header)
	if err != nil {
		return nil, err
	}
	key, ok := keys[header["kid"].(string)]
	if !ok {
		return nil, errors.New("claim made for unknown kid")
	}

	token, err := jwt.ParseWithClaims(t, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key.Materialize()
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to parse claims")
	}

	currentSeconds := time.Now().Unix()
	exp, ok := (*claims)["exp"].(float64)
	if !ok {
		return nil, errors.New("missing exp claim")
	}
	if int64(exp) < currentSeconds {
		return nil, errors.New("claim is expired")
	}

	authTime, ok := (*claims)["auth_time"].(float64)
	if !ok {
		return nil, errors.New("missing auth_time claim")
	}
	if int64(authTime) > currentSeconds {
		return nil, errors.New("claim is invalid")
	}

	cognitoIssuer, err := getCognitoIssuer()
	if err != nil {
		return nil, err
	}

	iss, ok := (*claims)["iss"].(string)
	if !ok || iss != cognitoIssuer {
		return nil, errors.New("claim issuer is invalid")
	}

	tokenUse, ok := (*claims)["token_use"].(string)
	if !ok || tokenUse != "access" {
		return nil, errors.New("claim use is not access")
	}

	return *claims, nil
}
*/

func handleOptionsRequest(code string, refresh string) (string, error) {

	redirectURI := os.Getenv("COGNITO_REDIRECT_URI")
	cognitoDomainPrefix := os.Getenv("COGNITO_DOMAIN_PREFIX")
	cognitoDomainPrefix = strings.ReplaceAll(cognitoDomainPrefix, ".", "-")
	cognitoClientID := os.Getenv("COGNITO_APP_CLIENT_ID")
	cognitoRegion := os.Getenv("COGNITO_REGION")

	tokenEndpoint := fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/oauth2/token",
		cognitoDomainPrefix, cognitoRegion)

	var postData url.Values

	if code != "" {
		postData = url.Values{
			"grant_type":   {"authorization_code"},
			"client_id":    {cognitoClientID},
			"code":         {code},
			"redirect_uri": {redirectURI},
		}
	} else {
		if refresh == "" {
			return "", errors.New("no refresh token")
		}

		postData = url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {cognitoClientID},
			"refresh_token": {refresh},
		}
	}

	fmt.Printf("About to post to %s: %+v\n", tokenEndpoint, postData)

	resp, err := http.PostForm(tokenEndpoint, postData)
	if err != nil {
		fmt.Printf("PostForm error from %s: %v\n", tokenEndpoint, err)
		return "", errors.New("token endpoint failed")
	}
	defer resp.Body.Close()

	fmt.Printf("resp: %+v\n", resp)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("request to %s failed with Status %d",
			tokenEndpoint, resp.StatusCode)
	}

	var token struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		fmt.Printf("json token response error: %v\n", err)
		return "", errors.New("failed to decode token response")
	}

	fmt.Printf("Got token: %+v\n", token)

	keys, err := getPublicKeys()
	if err != nil {
		return "", err
	}
	fmt.Printf("keys: %+v\n", keys)

	parsed, err := jwt.Parse([]byte(token.AccessToken), jwt.WithKeySet(keys))
	if err != nil {
		fmt.Printf("failed to verify: %s\n", err)
		return "", errors.New("failed to verify token")
	}

	fmt.Printf("parsed: %+v", parsed)

	userName, ok := parsed.Get("username")
	if !ok {
		return "", errors.New("missing username")
	}

	retval := struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		Username     string `json:"username"`
		ExpiresIn    int64  `json:"expiresIn"`
	}{
		IDToken:      parsed.JwtID(),
		RefreshToken: "?", // parsed.RefreshToken,
		Username:     strings.TrimPrefix(userName.(string), "AmazonFederate_"),
		ExpiresIn:    0, // parsed.ExpiresIn,
	}

	jsonData, err := json.Marshal(retval)
	if err != nil {
		return "", errors.New("failed to encode response")
	}

	return string(jsonData), nil

}
