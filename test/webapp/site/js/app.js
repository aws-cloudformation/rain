// This is a minimal example of how to handle the auth flow with Cognito and 
// then interact with DynamoDB once the user is logged in.

import { checkAuthCode } from "./auth"


(async function main() {

    // Check to see if we're logged in
    const isCognitoRedirect = await checkAuthCode()
    if (isCognitoRedirect) return

    // TODO
    alert("main") 
})()
