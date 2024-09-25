// This is a minimal example of how to handle the auth flow with Cognito and 
// then interact with DynamoDB once the user is logged in.

import { checkAuthCode } from "./auth"
import { LOGIN_URL, LOGOUT_URL } from "./config"
import * as restApi from "./rest-api"

(async function main() {

    const loginBtn = get("login-btn")
    loginBtn.onclick = function() {
        location.href = LOGIN_URL;
    }

    const logoutBtn = get("logout-btn")
    logoutBtn.onclick = function() {
        location.href = LOGOUT_URL
    }

    const testBtn = get("test-btn")
    testBtn.onclick = async function() {
        get("about").style.display = "none"
        get("test-data").style.display = "block"
        const data = await restApi.get("test", null, null, true)
        console.log("test data: " + JSON.stringify(data, null, 0))
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

function get(id) {
    return document.getElementById(id)
}

