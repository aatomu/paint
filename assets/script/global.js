// @ts-check

/**
 * @param {string} key
 * @return {string|undefined}
 */
function getCookie(key) {
  return document.cookie.
    split("; ").
    find((row) => row.startsWith(`${key}=`))?.
    split("=")[1]
}

/**
 * @param {string} key
 * @param {string} value 
 */
function setCookie(key,value) {
  document.cookie = `${key}=${value}`
}

