// This is a minimal example of how to handle the auth flow with Cognito and 
// then interact with DynamoDB once the user is logged in.

import { checkAuthCode } from "./auth"
import { LOGIN_URL, LOGOUT_URL } from "./config"

(async function main() {

    const loginHref = document.getElementById("loginHref")
    if (loginHref) {
        loginHref.setAttribute("href", LOGIN_URL)
    }

    const logoutHref = document.getElementById("logoutHref")
    if (logoutHref) {
        logoutHref.setAttribute("href", LOGOUT_URL)
    }

    // Check to see if we're logged in
    const isCognitoRedirect = await checkAuthCode()
    if (isCognitoRedirect) {
        console.log("Cognito redirect")
        return // checkAuthCode does a redirect to /
    } else {
        console.log("Not a Cognito redirect")

    }

})()
