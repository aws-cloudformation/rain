// This is a minimal example of how to handle the auth flow with Cognito and 
// then interact with DynamoDB once the user is logged in.

import { checkAuthCode } from "./auth"
import { LOGIN_URL, LOGOUT_URL } from "./config"

(async function main() {

    const loginBtn = document.getElementById("login-btn")
    if (loginBtn) {
        loginBtn.onclick = function() {
            location.href = LOGIN_URL;
        }
    }

    const logoutBtn = document.getElementById("logout-btn")
    if (logoutBtn) {
        logoutBtn.onclick = function() {
            location.href = LOGOUT_URL
        }
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
