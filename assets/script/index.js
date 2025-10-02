//@ts-check

/** @type {HTMLInputElement} */
// @ts-expect-error
const input_name = document.getElementById("input_name")
/** @type {HTMLInputElement} */
// @ts-expect-error
const input_room = document.getElementById("input_room")
/** @type {HTMLButtonElement} */
// @ts-expect-error
const input_join = document.getElementById("input_join")


input_join.addEventListener("click", () => {
  if (input_name.value !== "" && input_room.value !== "") {
    const url = new URL(window.location.href)
    url.pathname = "room"
    url.search = ""
    url.searchParams.append("id", input_room.value)
    url.searchParams.append("user", input_name.value)
    window.location.href = url.toString()
  }
})

window.addEventListener("DOMContentLoaded", () => {
  const name = getCookie("user")
  if (name) {
    input_name.value = name
  }
  const room = getCookie("room")
  if (room) {
    input_room.value = room
  }
})