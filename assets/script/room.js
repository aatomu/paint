// @ts-check

/** @type SVGElement */
//@ts-expect-error
const board = document.getElementById("board")
/** @type HTMLSpanElement */
//@ts-expect-error
const zoom = document.getElementById("zoom")

const boardConfig = {
  scale: 1,
  offsetX: 0,
  offsetY: 100,
}

/** 
 * @typedef {object} Pointer
 * @property {"move"|"pen"|"text"|"line"} mode tool Mode
 * @property {boolean} isDown Pointer down mode flag
 * @property {[x:number,y:number]} prev Pointer previous position
 * @property {string} current Modifiy object id
 * @property {any} data Modifiy object data
 */

/** @type Pointer */
const pointer = {
  mode: "move",
  isDown: false,
  prev: [0, 0],
  current: "",
  data: "",
}


window.addEventListener("wheel", (e) => {
  console.log("updateScale")

  const prevScale = boardConfig.scale
  if (e.deltaY < 0) {
    boardConfig.scale += 0.05
  } else {
    boardConfig.scale -= 0.05
  }
  boardConfig.scale = Math.max(boardConfig.scale, 0.05)
  boardConfig.scale = Math.min(boardConfig.scale, 30)

  const scaleRatio = boardConfig.scale / prevScale
  boardConfig.offsetX = e.clientX - ((e.clientX - boardConfig.offsetX) * scaleRatio);
  boardConfig.offsetY = e.clientY - ((e.clientY - boardConfig.offsetY) * scaleRatio);

  updateBoard()
})

window.addEventListener("mousedown", pointerDown)
window.addEventListener("mousemove", pointerMove)
window.addEventListener("mouseup", pointerUp)

function pointerDown(e) {
  pointer.isDown = true
  pointer.prev = [
    (e.clientX - boardConfig.offsetX),
    (e.clientY - boardConfig.offsetY)
  ]

  // @ts-expect-error
  pointer.mode = document.querySelector("input[name=tool]:checked").value
  console.log("pointerDown", pointer.mode, pointer.prev)

  switch (pointer.mode) {
    case "pen": {
      pointer.current = new Date().getTime().toString();
      const pen = document.createElementNS("http://www.w3.org/2000/svg", "path");
      pen.id = pointer.current
      pointer.data = `M${pointer.prev[0] / boardConfig.scale},${pointer.prev[1] / boardConfig.scale}`
      pen.setAttribute("d", pointer.data)
      const bold = 100
      const color = "#000000"
      const alpha = 1
      pen.setAttribute("style", `stroke-width: ${bold}px; stroke: ${color}; opacity: ${alpha};`)
      pen.setAttribute("stroke-linecap", "round")
      pen.setAttribute("stroke-linejoin", "round")
      board.appendChild(pen)
      break
    }
  }
}

function pointerMove(e) {
  const position = [
    (e.clientX - boardConfig.offsetX),
    (e.clientY - boardConfig.offsetY)
  ]
  console.log("pointerDown", pointer.mode, pointer.prev, position)

  if (!pointer.isDown) return
  switch (pointer.mode) {
    case "move": {
      boardConfig.offsetX += position[0] - pointer.prev[0]
      boardConfig.offsetY += position[1] - pointer.prev[1]
      console.log(position[0] - pointer.prev[0])
      updateBoard()
      break
    }
    case "pen": {
      const pen = document.getElementById(pointer.current)
      if (!pen) return
      pointer.data += `L${position[0] / boardConfig.scale},${position[1] / boardConfig.scale}`
      pen.setAttribute("d", pointer.data)
      break
    }
    case "text":
    case "line":
  }
}

function pointerUp(e) {
  console.log("pointerUp")
  pointer.isDown = false
}

function updateBoard() {
  board.style.transform = `scale(${boardConfig.scale})`
  board.style.top = `${boardConfig.offsetY}px`
  board.style.left = `${boardConfig.offsetX}px`
  zoom.textContent = "x" + boardConfig.scale.toFixed(2).padStart(5,"0")
}


updateBoard()
/**
 * @typedef {PacketMouse|PacketAdd|PacketRemove|PacketClear} Packet
 * @property {string} id
 * @property {string} user
 * @property {"mouse"|"add"|"remove"|"clear"} operation
 * @property {Object} data
 */

/**
 * @typedef {object} PacketMouse
 * @property {string} id
 * @property {string} user
 * @property {"mouse"} operation
 * @property {[x:number,y:number]} data
 */
/**
 * @typedef {object} PacketAdd
 * @property {string} id
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
 * @property {[x:number,y:number]} pos
 */

/**
 * @typedef {object} writeLine
 * @property {"line"} type
 * @property {string} color
 * @property {string} bold
 * @property {[x:number,y:number]} start
 * @property {[x:number,y:number]} end
 */


/**
 * @typedef {object} PacketRemove
 * @property {string} id
 * @property {string} user
 * @property {"remove"} operation
 * @property {null} data
 */

/**
 * @typedef {object} PacketClear
 * @property {string} id
 * @property {string} user
 * @property {"clear"} operation
 * @property {null} data
 */