const APIGATEWAY_URL = "__APIGW__"
const REDIRECT_URI = "__REDIRECT_URI__" 
const COGNITO_DOMAIN = "__COGNITO_DOMAIN__" 
const REGION = "__REGION__"
const APP_CLIENT_ID = "__APP_CLIENT_ID" 

const COGNITO_URL = `https://${COGNITO_DOMAIN}.auth.${REGION}.amazoncognito.com`
const PARAMS = `?response_type=code&client_id=${APP_CLIENT_ID}&redirect_uri=${REDIRECT_URI}`
const LOGIN_URL = `${COGNITO_URL}/login${PARAMS}`
const LOGOUT_URL = `${COGNITO_URL}/logout${PARAMS}`

const LOCAL_JWT = ""

export { APIGATEWAY_URL, LOGIN_URL, LOGOUT_URL, LOCAL_JWT }
