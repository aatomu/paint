// @ts-check

/** @type SVGElement */
//@ts-expect-error
const board = document.getElementById("board")
/** @type HTMLSpanElement */
//@ts-expect-error
const zoomValue = document.getElementById("zoom-value")
/** @type HTMLTextAreaElement */
//@ts-expect-error
const stampInput = document.getElementById("stamp-input")
/** @type HTMLInputElement */
//@ts-expect-error
const colorInput = document.getElementById("color-input")
/** @type HTMLInputElement */
//@ts-expect-error
const opacityRange = document.getElementById("opacity-range")
/** @type HTMLSpanElement */
//@ts-expect-error
const opacityValue = document.getElementById("opacity-value")
/** @type HTMLInputElement */
//@ts-expect-error
const boldRange = document.getElementById("bold-range")
/** @type HTMLInputElement */
//@ts-expect-error
const boldInput = document.getElementById("bold-input")

const boardConfig = {
  scale: 1,
  offsetX: 0,
  offsetY: 0,
  color: "",
  opacity: 1,
  bold: 15,
}

/** 
 * @typedef {object} Pointer
 * @property {"move"|"pen"|"line"|"stamp"} mode tool Mode
 * @property {boolean} isDownPrev previous Pointer down mode flag
 * @property {boolean} isDown Pointer down mode flag
 * @property {[x:number,y:number]} prev Pointer previous position
 * @property {string} current Modifiy object id
 * @property {any} data Modifiy object data
 */

/** @type Pointer */
const pointer = {
  mode: "move",
  isDownPrev: false,
  isDown: false,
  prev: [0, 0],
  current: "",
  data: "",
}


window.addEventListener("wheel", (e) => {

  const prevScale = boardConfig.scale
  if (e.deltaY < 0) {
    boardConfig.scale += 0.05
    console.log(JSON.stringify({ "event": "scroll", "callback": "zoom-in" }))
  } else {
    boardConfig.scale -= 0.05
    console.log(JSON.stringify({ "event": "scroll", "callback": "zoom-out" }))
  }
  boardConfig.scale = Math.max(boardConfig.scale, 0.05)
  boardConfig.scale = Math.min(boardConfig.scale, 30)

  const scaleRatio = boardConfig.scale / prevScale
  boardConfig.offsetX = e.clientX - ((e.clientX - boardConfig.offsetX) * scaleRatio);
  boardConfig.offsetY = e.clientY - ((e.clientY - boardConfig.offsetY) * scaleRatio);

  updateBoard()
})

// MARK: pointerEvent
window.addEventListener("mousemove", pointerEvent)

function pointerEvent(e) {
  const isDown = e.buttons !== 0

  let callback = "null"
  // change to down
  if (isDown && !pointer.isDownPrev) {
    pointerDown(e)
    callback = "down"
  }
  // keep to down
  if (isDown && pointer.isDownPrev) {
    pointerMove(e)
    callback = "move"
  }
  // change to up
  if (!isDown && pointer.isDownPrev) {
    pointerUp(e)
    callback = "up"
  }

  console.log(JSON.stringify({ "event": "pointer", "callback": callback, "mode": pointer.mode }))
  pointer.isDownPrev = isDown
}

function pointerDown(e) {
  pointer.isDown = true
  pointer.prev = [
    (e.clientX - boardConfig.offsetX),
    (e.clientY - boardConfig.offsetY)
  ]

  // @ts-expect-error
  pointer.mode = document.querySelector("input[name=tool]:checked").value

  /** @type {SVGElement[]|HTMLElement[]} */
  // @ts-expect-error
  const targetElements = document.elementsFromPoint(e.clientX, e.clientY)
  let isBypass = false
  for (let i = 0; i < targetElements.length; i++) {
    if (targetElements[i].classList.contains("side-menu") || targetElements[i].classList.contains("side-menu-area")) {
      isBypass = true
      break
    }
  }
  if (isBypass) pointer.isDown = false
  console.log("cancel by .side-menu")

  switch (pointer.mode) {
    case "move": {
      break
    }
    case "pen": {
      pointer.current = new Date().getTime().toString();
      const pen = document.createElementNS("http://www.w3.org/2000/svg", "path");
      pen.id = pointer.current
      pointer.data = `M${pointer.prev[0] / boardConfig.scale},${pointer.prev[1] / boardConfig.scale}`
      pen.setAttribute("d", pointer.data)
      pen.setAttribute("style", `stroke-width: ${boardConfig.bold}px; stroke: ${boardConfig.color}; opacity: ${boardConfig.opacity};`)
      pen.setAttribute("stroke-linecap", "round")
      pen.setAttribute("stroke-linejoin", "round")
      board.appendChild(pen)
      break
    }
    case "line": {
      pointer.current = new Date().getTime().toString();
      const line = document.createElementNS("http://www.w3.org/2000/svg", "line");
      line.id = pointer.current
      line.setAttribute("x1", (pointer.prev[0] / boardConfig.scale).toFixed(0))
      line.setAttribute("y1", (pointer.prev[1] / boardConfig.scale).toFixed(0))
      line.setAttribute("x2", (pointer.prev[0] / boardConfig.scale).toFixed(0))
      line.setAttribute("y2", (pointer.prev[1] / boardConfig.scale).toFixed(0))
      line.setAttribute("style", `stroke-width: ${boardConfig.bold}px; stroke: ${boardConfig.color}; opacity: ${boardConfig.opacity};`)
      line.setAttribute("stroke-linecap", "round")
      board.appendChild(line)
      break
    }
    case "stamp": {
      if (stampInput.value.length === 0) break
      pointer.current = new Date().getTime().toString();
      const stamp = document.createElementNS("http://www.w3.org/2000/svg", "text");
      stamp.id = pointer.current
      stamp.setAttribute("x", (pointer.prev[0] / boardConfig.scale).toFixed(0))
      stamp.setAttribute("y", (pointer.prev[1] / boardConfig.scale).toFixed(0))
      stamp.setAttribute("text-anchor", "middle")
      const color = "#000000"
      const fontSize = boardConfig.bold * 2
      stamp.setAttribute("style", `font-size: ${fontSize}px; fill: ${boardConfig.color}; opacity: ${boardConfig.opacity};`)
      const stampLines = stampInput.value.split("\n")
      for (let i = 0; i < stampLines.length; i++) {
        const line = document.createElementNS("http://www.w3.org/2000/svg", "tspan");
        line.textContent = stampLines[i]
        line.setAttribute("dy", fontSize.toFixed(2))
        line.setAttribute("text-anchor", "middle")
        stamp.append(line)
      }
      board.appendChild(stamp)
    }
  }
}

function pointerMove(e) {
  const position = [
    (e.clientX - boardConfig.offsetX),
    (e.clientY - boardConfig.offsetY)
  ]

  if (!pointer.isDown) return
  switch (pointer.mode) {
    case "move": {
      boardConfig.offsetX += position[0] - pointer.prev[0]
      boardConfig.offsetY += position[1] - pointer.prev[1]
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
    case "stamp": {
      const stamp = document.getElementById(pointer.current)
      if (!stamp) return
      stamp.setAttribute("x", (position[0] / boardConfig.scale).toFixed(0))
      stamp.setAttribute("y", (position[1] / boardConfig.scale).toFixed(0))
      for (let i = 0; i < stamp.children.length; i++) {
        stamp.children[i].setAttribute("x", (position[0] / boardConfig.scale).toFixed(0))
      }
      break
    }
    case "line": {
      const line = document.getElementById(pointer.current)
      if (!line) return
      line.setAttribute("x2", (position[0] / boardConfig.scale).toFixed(0))
      line.setAttribute("y2", (position[1] / boardConfig.scale).toFixed(0))
      break
    }
  }
}

function pointerUp(e) {
  pointer.isDown = false
}

// MARK: updateBoard()
function updateBoard() {
  board.style.transform = `scale(${boardConfig.scale})`
  board.style.top = `${boardConfig.offsetY}px`
  board.style.left = `${boardConfig.offsetX}px`
  zoomValue.textContent = "x" + boardConfig.scale.toFixed(2).padStart(5, "0")
}
updateBoard()

// MARK: #color
document.querySelectorAll("div.color-template").forEach((element) => {
  /** @type {HTMLElement|null} */
  const preview = element.querySelector("div.color-template-preview")
  if (!preview) return
  const color = preview.dataset.color
  if (!color) return
  preview.style.backgroundColor = color
  element.addEventListener("click", () => {
    updateColor(color)
  })
})

colorInput.addEventListener("input", () => {
  updateColor(colorInput.value)
})

function updateColor(value) {
  boardConfig.color = value
  colorInput.value = value
}

// MARK: #opacity
opacityRange.addEventListener("input", () => {
  opacityValue.textContent = opacityRange.value.padStart(3, "0") + "%"
  boardConfig.opacity = parseFloat(opacityRange.value) / 100
})

// MARK: #bold
document.querySelectorAll("div.bold-template").forEach((element) => {
  /** @type {HTMLElement|null} */
  const preview = element.querySelector("div.bold-template-preview")
  if (!preview) return
  const bold = preview.dataset.bold
  if (!bold) return
  preview.style.height = bold + "px"
  element.addEventListener("click", () => {
    updateBold(parseInt(bold))
  })
})

boldRange.addEventListener("input", () => {
  updateBold(boldRange.value)
})
boldInput.addEventListener("input", () => {
  updateBold(boldInput.value)
})

function updateBold(value) {
  boardConfig.bold = value
  boldRange.value = value
  boldInput.value = value
}
updateBold(15)

// MARK: #typedef

/**
 * @typedef {PacketMouse|PacketAdd|PacketRemove|PacketClear} Packet
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
 * @property {writePen|writeLine|writeStamp} data
 */

/**
 * @typedef {object} writePen
 * @property {"pen"} type
 * @property {string} d
 * @property {string} color
 * @property {string} bold
 */

/**
 * @typedef {object} writeStamp
 * @property {"stamp"} type
 * @property {string[]} text
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
 * @property {{id:string}} data
 */

/**
 * @typedef {object} PacketClear
 * @property {string} id
 * @property {string} user
 * @property {"clear"} operation
 * @property {null} data
 */