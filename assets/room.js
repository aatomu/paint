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
 * @typedef {PacketMouse|PacketAdd|PacketRemove|PacketClear} Packet
 * @property {number} timestamp
 * @property {string} user
 * @property {"mouse"|"add"|"remove"|"clear"} operation
 * @property {Object} data
 */

/**
 * @typedef {object} PacketMouse
 * @property {number} timestamp
 * @property {string} user
 * @property {"mouse"} operation
 * @property {{x:number,y:number}} data
 */
/**
 * @typedef {object} PacketAdd
 * @property {number} timestamp
 * @property {string} user
 * @property {"add"} operation
 * @property {writePen|writeText|writeLine} data
 */

/**
 * @typedef {object} writePen
 * @property {"pen"} type
 * @property {string} d
 * @property {string} color
 * @property {string} bold
 */

/**
 * @typedef {object} writeText
 * @property {"text"} type
 * @property {string} text
 * @property {string} color
 * @property {string} bold
 * @property {{x:number,y:number}} pos
 */

/**
 * @typedef {object} writeLine
 * @property {"line"} type
 * @property {string} color
 * @property {string} bold
 * @property {{x:number,y:number}} start
 * @property {{x:number,y:number}} end
 */


/**
 * @typedef {object} PacketRemove
 * @property {number} timestamp
 * @property {string} user
 * @property {"remove"} operation
 * @property {{timestamp:number}} data
 */

/**
 * @typedef {object} PacketClear
 * @property {number} timestamp
 * @property {string} user
 * @property {"clear"} operation
 * @property {null} data
 */