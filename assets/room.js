// @ts-check

/** @type SVGElement */
//@ts-expect-error
const board = document.getElementById("board")

const boardConfig = {
  scale: 1.0,
  offsetX: 0,
  offsetY: 0,
}
const mouse = {
  isDragging: false,
  startX: 0,
  startY: 0
}


document.addEventListener("wheel", (e) => {
  console.log("updateScale")
  e.preventDefault()

  if (e.deltaY < 0) {
    boardConfig.scale += 0.05
  } else {
    boardConfig.scale -= 0.05
  }
  boardConfig.scale = Math.max(boardConfig.scale, 0.05)

  updateBoard()
})

window.addEventListener("mousedown", (e) => {
  mouse.isDragging = true
  mouse.startX = e.clientX - boardConfig.offsetX
  mouse.startY = e.clientY - boardConfig.offsetY
})
window.addEventListener("mousemove", (e) => {
  if (!mouse.isDragging) return
  boardConfig.offsetX = e.clientX - mouse.startX
  boardConfig.offsetY = e.clientY - mouse.startY
  updateBoard()
})
window.addEventListener("mouseup", (e) => {
  mouse.isDragging = false
})


function updateBoard() {
  board.style.transform =
    `translate(${boardConfig.offsetX}px, ${boardConfig.offsetY}px) scale(${boardConfig.scale})`
}


/**
 * @typedef {BoardDataMouse|BoardDataAdd|BoardDataRemove|BoardDataClear} BoardData
 * @property {number} timestamp
 * @property {string} user
 * @property {"mouse"|"add"|"remove"|"clear"} operation
 * @property {Object} data
 */

/**
 * @typedef {object} BoardDataMouse
 * @property {number} timestamp
 * @property {string} user
 * @property {"mouse"} operation
 * @property {{x:number,y:number}} data
 */
/**
 * @typedef {object} BoardDataAdd
 * @property {number} timestamp
 * @property {string} user
 * @property {"add"} operation
 * @property {string} data
 */

/**
 * @typedef {object} BoardDataRemove
 * @property {number} timestamp
 * @property {string} user
 * @property {"remove"} operation
 * @property {string} data
 */

/**
 * @typedef {object} BoardDataClear
 * @property {number} timestamp
 * @property {string} user
 * @property {"clear"} operation
 * @property {null} data
 */