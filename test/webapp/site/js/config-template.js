// Template variables will be replaced by buildsite.sh
const APIGATEWAY_URL = "__APIGW__"

// TODO: After creating cognito stuff in the template, automate replacement
const REDIRECT_URI = "https://d1oswaizh0vr7s.cloudfront.net/index.html" 
const COGNITO_DOMAIN = "rain-webapp-manual-test1" 
const REGION = "us-east-1"
const APP_CLIENT_ID = "77p10r3qjrjeou9rvncrc1q33a" 

const COGNITO_URL = `https://${COGNITO_DOMAIN}.auth.${REGION}.amazoncognito.com`
const PARAMS = `?response_type=code&client_id=${APP_CLIENT_ID}&redirect_uri=${REDIRECT_URI}`
const LOGIN_URL = `${COGNITO_URL}/login${PARAMS}`
const LOGOUT_URL = `${COGNITO_URL}/logout${PARAMS}`

const LOCAL_JWT = ""

export { APIGATEWAY_URL, LOGIN_URL, LOGOUT_URL, LOCAL_JWT }
