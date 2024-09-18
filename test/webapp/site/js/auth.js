import { getParameterByName } from "./web-util"
import * as restApi from "./rest-api"
import Cookies from "js-cookie"
import { TARGET_ENV } from "./config"

/**
 * Set a secure cookie.
 * 
 * @param {*} name 
 * @param {*} value 
 */
function setCookie(name, value) {
    console.log(`Setting cookie ${name} ${value}`)

    if (window.location.hostname === "localhost") {
        Cookies.set(name, value)
    } else {
        Cookies.set(name, value, { secure: true, sameSite: "strict" })
    }
}

/**
 * Get the value of a cookie.
 * 
 * @param {*} name 
 * @returns 
 */
function getCookie(name) {
    return Cookies.get(name)
}

/**
 * Delete a cookie.
 * 
 * @param {*} name 
 */
function removeCookie(name) {
    Cookies.remove(name)
}

/**
 * Save data from the JWT token.
 */
function setAuthCookies(data) {

    const idToken = data.idToken
    const refreshToken = data.refreshToken
    const expiresIn = data.expiresIn

    console.info(expiresIn)

    setCookie("jwt.id", idToken)

    // Put the jwt in a link so it's easy to copy out for local dev
    const targetEnvDiv = document.getElementById("target-env")
    targetEnvDiv.innerHTML = `<a href = "#jwt-modal">${TARGET_ENV}</a>`
    document.getElementById("jwt-modal-content").innerHTML = refreshToken

    setCookie("jwt.refresh", refreshToken)
    const exp = new Date()
    const totalSeconds = exp.getSeconds() + expiresIn

    console.info(totalSeconds)

    exp.setSeconds(totalSeconds)
    setCookie("jwt.expires", exp.toISOString())
    setCookie("username", data.username)

    console.log("JWT cookies set")
}

/**
 * Check to see if this request is Cognito sending us the auth code redirect.
 */
async function checkAuthCode() {
    const code = getParameterByName("code")

    if (code) {
        console.log("Found code in query string: " + code)

        const data = await restApi.get(`jwt-get?code=${code}`, null, null, false)
        console.log("jwt-get response: " + JSON.stringify(data, null, 0))

        setAuthCookies(data)

        // Redirect to the bare URL without code=
        window.location = "/"
        return true
    }

    return false
}

export { checkAuthCode, setAuthCookies, setCookie, getCookie, removeCookie }
