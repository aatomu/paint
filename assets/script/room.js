// @ts-check

// MARK: Const
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
/** @type HTMLInputElement */
//@ts-expect-error
const undoInput = document.getElementById("undo-input")
/** @type HTMLInputElement */
//@ts-expect-error
const redoInput = document.getElementById("redo-input")
/** @type HTMLInputElement */
//@ts-expect-error
const zoomIn = document.getElementById("zoom-in")
/** @type HTMLInputElement */
//@ts-expect-error
const clearInput = document.getElementById("clear-input")
/** @type HTMLInputElement */
//@ts-expect-error
const zoomOut = document.getElementById("zoom-out")
/** @type HTMLInputElement */
//@ts-expect-error
const pngInput = document.getElementById("png-input")
/** @type HTMLInputElement */
//@ts-expect-error
const svgInput = document.getElementById("svg-input")

// MARK: Vars
/** @type {BoardConfigration} */
const boardConfig = {
  scale: 0.3,
  offset: [50, 150],
  color: "",
  opacity: 1,
  bold: 15,
}

/** @type {PointerConfiguration} */
const pointer = {
  mode: "move",
  isDownPrev: false,
  isDown: false,
  prev: [0, 0],
  notify: 0,
  current: null
}

/** @type {TransactionConfigration} */
const transaction = {
  requestHistory: true,
  ws: null,
  heartbeatId: -1,
  startTime: 0,
  retry: 0,
}

const url = new URL(window.location.href)
/** @type {string} */
// @ts-expect-error
const room = url.searchParams.get("id")
/** @type {string} */
// @ts-expect-error
const user = url.searchParams.get("user")

/** @type {{[packet_id: string]:PacketEvent}} */
let packetStack = {
  "example": {
    packet_id: "",
    user: "",
    operation: "",
    data: null,
  }
}

// MARK: Zoom in/out
window.addEventListener("wheel", (event) => {
  if (isSidemenuArea([event.clientX, event.clientY])) {
    return
  }
  if (event.deltaY < 0) {
    zoom(true, [event.clientX, event.clientY])
  } else {
    zoom(false, [event.clientX, event.clientY])
  }
})

zoomIn.addEventListener("click", () => {
  zoom(true, [window.innerWidth / 2, window.innerHeight / 2])
})
zoomOut.addEventListener("click", () => {
  zoom(false, [window.innerWidth / 2, window.innerHeight / 2])
})

/**
 * @param {boolean} zoomIn
 * @param {[number,number]} center 
 */
function zoom(zoomIn, center) {
  const prevScale = boardConfig.scale
  if (zoomIn) {
    boardConfig.scale += 0.05
    console.log(JSON.stringify({ "event": "zoom()", "callback": "zoom-in" }))
  } else {
    boardConfig.scale -= 0.05
    console.log(JSON.stringify({ "event": "zoom()", "callback": "zoom-out" }))
  }
  boardConfig.scale = Math.max(boardConfig.scale, 0.05)
  boardConfig.scale = Math.min(boardConfig.scale, 30)

  const scaleRatio = boardConfig.scale / prevScale
  boardConfig.offset[0] = center[0] - ((center[0] - boardConfig.offset[0]) * scaleRatio);
  boardConfig.offset[1] = center[1] - ((center[1] - boardConfig.offset[1]) * scaleRatio);

  updateBoard()
}

// MARK: pointerEvent
window.addEventListener("mousemove", pointerEvent)

/**
 * @param {MouseEvent} event 
 * @return {void}
 */
function pointerEvent(event) {
  const isDown = event.buttons !== 0

  let callback = null
  // change to down
  if (isDown && !pointer.isDownPrev) {
    pointerDown(event)
    callback = "down"
  }
  // keep to down
  if (isDown && pointer.isDownPrev) {
    pointerMove(event)
    callback = "move"
  }
  // change to up
  if (!isDown && pointer.isDownPrev) {
    pointerUp(event)
    callback = "up"
  }

  // if (!(isDown && pointer.mode == "move")) {
  //   pointer.notify++
  //   if (pointer.notify > 50) {
  //     pointer.notify = 0
  //     sendPacket({
  //       packet_id: `mouse-${(new Date()).getTime()}`,
  //       user: user,
  //       operation: "mouse",
  //       data: {
  //         pos: [
  //           fixedNumber(event.clientX - boardConfig.offset[0]),
  //           fixedNumber(event.clientY - boardConfig.offset[1])
  //         ]
  //       }
  //     })
  //   }
  // }

  if (callback) console.log(JSON.stringify({ "event": "pointer", "callback": callback, "mode": pointer.mode }))
  pointer.isDownPrev = isDown
}

//MARK: > pointerDown
/**
 * @param {MouseEvent} event
 * @return {void}
 */
function pointerDown(event) {
  pointer.isDown = true
  pointer.prev = [
    (event.clientX - boardConfig.offset[0]),
    (event.clientY - boardConfig.offset[1])
  ]

  // @ts-expect-error
  pointer.mode = document.querySelector("input[name=tool]:checked").value

  if (isSidemenuArea([event.clientX, event.clientY])) {
    pointer.isDown = false
    console.log("cancel by .side-menu")
    return
  }

  /** @type {[number,number]} */
  const pos = [(pointer.prev[0] / boardConfig.scale), (pointer.prev[1] / boardConfig.scale)]
  switch (pointer.mode) {
    case "move": { //MARK: >> move
      break
    }
    case "pen": { //MARK: >> pen
      pointer.current = {
        element_id: new Date().getTime(),
        bold: boardConfig.bold,
        color: boardConfig.color,
        opacity: boardConfig.opacity,
        element_type: "pen",
        property: {
          d: `M${fixedString(pos[0])},${fixedString(pos[1])}`
        }
      }
      createElement(pointer.current)
      break
    }
    case "line": { //MARK: >> line
      pointer.current = {
        element_id: new Date().getTime(),
        bold: boardConfig.bold,
        color: boardConfig.color,
        opacity: boardConfig.opacity,
        element_type: "line",
        property: {
          start: [fixedNumber(pos[0]), fixedNumber(pos[1])],
          end: [fixedNumber(pos[0]), fixedNumber(pos[1])],
        }
      }
      createElement(pointer.current)
      break
    }
    case "stamp": { //MARK: >> stamp
      if (stampInput.value.length === 0) break
      const stampLines = stampInput.value.split("\n")

      pointer.current = {
        element_id: new Date().getTime(),
        bold: boardConfig.bold,
        color: boardConfig.color,
        opacity: boardConfig.opacity,
        element_type: "stamp",
        property: {
          pos: [fixedNumber(pos[0]), fixedNumber(pos[1])],
          text: JSON.stringify(stampLines)
        }
      }
      createElement(pointer.current)
      break
    }
    case "delete": { //MARK: >> delete
      break
    }
  }
}

//MARK: > pointerMove
/**
 * @param {MouseEvent} event 
 * @return {void}
 */
function pointerMove(event) {
  const position = [
    (event.clientX - boardConfig.offset[0]),
    (event.clientY - boardConfig.offset[1])
  ]

  if (!pointer.isDown) return
  switch (pointer.mode) {
    case "move": {  //MARK: >> move
      boardConfig.offset[0] += position[0] - pointer.prev[0]
      boardConfig.offset[1] += position[1] - pointer.prev[1]
      updateBoard()
      break
    }
    case "pen": { //MARK: >> pen
      if (!pointer.current) return
      if (pointer.current.element_type != "pen") return
      const pen = document.getElementById(pointer.current.element_id.toString())
      if (!pen) return

      pointer.current.property.d += `L${fixedString(position[0] / boardConfig.scale)},${fixedString(position[1] / boardConfig.scale)}`
      pen.setAttribute("d", pointer.current.property.d)
      break
    }
    case "line": { //MARK: >> line
      if (!pointer.current) return
      if (pointer.current.element_type != "line") return
      const line = document.getElementById(pointer.current.element_id.toString())
      if (!line) return

      const pos = [position[0] / boardConfig.scale, position[1] / boardConfig.scale]
      pointer.current.property.end = [fixedNumber(pos[0]), fixedNumber(pos[1])]
      line.setAttribute("x2", fixedString(pos[0]))
      line.setAttribute("y2", fixedString(pos[1]))
      break
    }
    case "stamp": { //MARK: >> stamp
      if (!pointer.current) return
      if (pointer.current.element_type != "stamp") return
      const stamp = document.getElementById(pointer.current.element_id.toString())
      if (!stamp) return

      const pos = [position[0] / boardConfig.scale, position[1] / boardConfig.scale]
      pointer.current.property.pos = [fixedNumber(pos[0]), fixedNumber(pos[1])]
      stamp.setAttribute("x", fixedString(pos[0]))
      stamp.setAttribute("y", fixedString(pos[1]))
      for (let i = 0; i < stamp.children.length; i++) {
        stamp.children[i].setAttribute("x", fixedString(pos[0]))
      }
      break
    }
    case "delete": { //MARK: >> delete
      /** @type {SVGElement[]|HTMLElement[]} */
      // @ts-expect-error
      const targetElements = document.elementsFromPoint(event.clientX, event.clientY)
      if (targetElements.length < 1) break
      let target = targetElements[0]
      if (!["path", "line", "text", "tspan"].includes(target.localName)) break
      if (target.localName === "tspan" && target.parentElement) target = target.parentElement

      target.style.display = "none"
      /** @type {PacketEvent} */
      const packet = {
        packet_id: UUIDv7(),
        user: user,
        operation: "delete",
        data: {
          target: parseInt(target.id)
        }
      }
      sendPacket(packet)
      packetStack[packet.packet_id] = packet
      break
    }
  }
}

//MARK: > pointerUp
/**
 * @param {MouseEvent} event 
 * @return {void}
 */
function pointerUp(event) {
  pointer.isDown = false

  if (pointer.current) {
    /** @type {PacketEvent} */
    const packet = {
      packet_id: UUIDv7(),
      user: user,
      operation: "create",
      data: pointer.current
    }
    pointer.current = null
    sendPacket(packet)
    packetStack[packet.packet_id] = packet
  }
}

// MARK: updateBoard()
function updateBoard() {
  board.style.transform = `scale(${boardConfig.scale})`
  board.style.top = `${boardConfig.offset[1]}px`
  board.style.left = `${boardConfig.offset[0]}px`
  zoomValue.textContent = "x" + boardConfig.scale.toFixed(2).padStart(5, "0")
}

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

/**
 * @param {string} value
 * @return {void}
 */
function updateColor(value) {
  boardConfig.color = value
  colorInput.value = value
}

// MARK: #opacity
opacityRange.addEventListener("input", () => {
  opacityValue.textContent = opacityRange.value.padStart(3, "0") + "%"
  opacityValue.style.opacity = opacityRange.value.padStart(3, "0") + "%"
  boardConfig.opacity = parseFloat(opacityRange.value) / 100
})

// MARK: #bold
document.querySelectorAll("div.bold-template").forEach((element) => {
  /** @type {HTMLElement|null} */
  const preview = element.querySelector("div.bold-template-preview")
  if (!preview) return
  const bold = preview.dataset.bold
  if (!bold) return
  preview.style.height = `${bold}px`
  element.addEventListener("click", () => {
    updateBold(bold)
  })
})

boldRange.addEventListener("input", () => {
  updateBold(boldRange.value)
})
boldInput.addEventListener("input", () => {
  updateBold(boldInput.value)
})

/**
 * @param {string} value
 * @return {void}
 */
function updateBold(value) {
  boardConfig.bold = parseInt(value)
  boldRange.value = value
  boldInput.value = value
}

// MARK: #redo
undoInput.addEventListener("click", () => {
  sendPacket({
    packet_id: UUIDv7(),
    user: user,
    operation: "undo",
    data: {}
  })
})

// MARK: #redo
redoInput.addEventListener("click", () => {
  sendPacket({
    packet_id: UUIDv7(),
    user: user,
    operation: "redo",
    data: {}
  })
})

// MARK: #clear
clearInput.addEventListener("input", () => {
  if (clearInput.value === "clear board") {
    /** @type {PacketEvent} */
    const packet = {
      packet_id: UUIDv7(),
      user: user,
      operation: "clear",
      data: {}
    }
    sendPacket(packet)
    packetStack[packet.packet_id] = packet
    clearInput.value = ""
  }
})

// MARK: ContentLoaded()
if (document.readyState == "complete") {
  Initialize()
} else {
  window.addEventListener("DOMContentLoaded", Initialize)
}

function Initialize() {

  updateBoard()
  updateColor("#000000")
  updateBold("15")

  document.title += `- ${room}`

  NewWebsocket()
}

// MARK: NewWebsocket()
function NewWebsocket() {
  const url = new URL(window.location.href)
  url.pathname = "/ws"
  transaction.ws = new WebSocket(url.href)

  transaction.startTime = new Date().getTime()

  transaction.ws.addEventListener("open", WebsocketOpen)
  transaction.ws.addEventListener("message", WebsocketMessage)
  transaction.ws.addEventListener("error", WebSocketError)
  transaction.ws.addEventListener("close", WebsocketClose)
}

// MARK: WebsocketOpen()
/**
 * @param {Event} event
 */
function WebsocketOpen(event) {
  console.log(JSON.stringify({ "event": "websocket", "callback": "open" }), event)

  if (transaction.requestHistory) {
    sendPacket({
      packet_id: new Date().getTime().toString(),
      user: user,
      operation: "history",
      data: {}
    })
    transaction.requestHistory = false
  }

  // @ts-expect-error
  transaction.heartbeatId = setInterval(() => {
    sendPacket({
      packet_id: new Date().getTime().toString(),
      user: user,
      operation: "heatbeat",
      data: {}
    })
  }, 5000)
}

// MARK: WebsocketMessage()
/**
 * @param {MessageEvent} event
 */
function WebsocketMessage(event) {
  console.log(JSON.stringify({ "event": "websocket", "callback": "message" }), event)


  //   c<=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
  //   s=>c :"success"|"error"
  /** @type {PacketEvent} */
  const receivePacket = JSON.parse(event.data)
  switch (receivePacket.operation) {
    case "mouse": { //MARK: >> mouse
      break
    }
    case "create": { //MARK: >> create
      createElement(receivePacket.data)
      break
    }
    case "delete": { //MARK: >> delete
      const target = document.getElementById(receivePacket.data.target.toString())
      if (target) target.remove()
      break
    }
    case "undo": { //MARK: >> undo

      break
    }
    case "redo": { //MARK: >> redo

      break
    }
    case "clear": { //MARK: >> clear
      window.location.reload()
      break
    }
    case "success": { //MARK: >> success
      const success_packet = packetStack[receivePacket.data.packet_id]
      if (!success_packet) break

      switch (success_packet.operation) {
        case "create": {
          const target = document.getElementById(success_packet.data.element_id.toString())
          if (target) target.remove()
          break
        }
        case "delete": {
          const target = document.getElementById(success_packet.data.target.toString())
          if (target) target.remove()
          break
        }
        case "clear": {
          window.location.reload()
        }
      }

      delete packetStack[receivePacket.data.packet_id]
      break
    }
    case "error": { //MARK: >> error
      const error_packet = packetStack[receivePacket.data.packet_id]
      if (!error_packet) break

      switch (error_packet.operation) {
        case "mouse": {
          break
        }
        case "create": {
          const target = document.getElementById(error_packet.data.element_id.toString())
          if (target) target.remove()
          break
        }
        case "delete": {
          const target = document.getElementById(error_packet.data.target.toString())
          if (target) target.style.display = ""
          break
        }
        case "undo": {
          break
        }
        case "redo": {
          break
        }
        case "clear": {
          break
        }
      }
      break
    }
  }
}

// MARK: WebsocketError()
/**
 * @param {Event|ErrorEvent} event
 */
function WebSocketError(event) {
  console.log(JSON.stringify({ "event": "websocket", "callback": "error" }), event)
}

// MARK: WebsocketClose()
/**
* @param {CloseEvent} event
*/
function WebsocketClose(event) {
  console.log(JSON.stringify({ "event": "websocket", "callback": "close" }), event)

  clearInterval(transaction.heartbeatId)
  if (transaction.ws) {
    transaction.ws.removeEventListener("open", WebsocketOpen)
    transaction.ws.removeEventListener("message", WebsocketMessage)
    transaction.ws.removeEventListener("error", WebSocketError)
    transaction.ws.removeEventListener("close", WebsocketClose)
    transaction.ws = null

    if (new Date().getTime() - transaction.startTime < 1000 * 10) {
      transaction.retry += 1
    }
    if (transaction.retry < 5) {
      NewWebsocket()
    } else {
      window.location.href = "/"
    }
  }
}

/**
 * @param {string} text
 */
function sendMessage(text) {
  if (!transaction.ws) return
  transaction.ws.send(text)
}

/**
 * @param {PacketEvent} packet
 */
function sendPacket(packet) {
  sendMessage(JSON.stringify(packet))
}


// MARK: createElement()
/**
 * @param {PacketEventCreate} element
 */
function createElement(element) {
  switch (element.element_type) {
    case "pen": { // MARK: > pen
      const pen = document.createElementNS("http://www.w3.org/2000/svg", "path");
      pen.id = element.element_id.toString()
      pen.setAttribute("d", element.property.d)
      pen.setAttribute("style", `stroke-width: ${element.bold}px; stroke: ${element.color}; opacity: ${element.opacity};`)
      pen.setAttribute("stroke-linecap", "round")
      pen.setAttribute("stroke-linejoin", "round")
      board.appendChild(pen)
      break
    }
    case "line": { // MARK: > line
      const line = document.createElementNS("http://www.w3.org/2000/svg", "line");
      line.id = element.element_id.toString()
      line.setAttribute("x1", fixedString(element.property.start[0]))
      line.setAttribute("y1", fixedString(element.property.start[1]))
      line.setAttribute("x2", fixedString(element.property.end[0]))
      line.setAttribute("y2", fixedString(element.property.end[1]))
      line.setAttribute("style", `stroke-width: ${element.bold}px; stroke: ${element.color}; opacity: ${element.opacity};`)
      line.setAttribute("stroke-linecap", "round")
      board.appendChild(line)
      break
    }
    case "stamp": { // MARK: > stamp
      const stamp = document.createElementNS("http://www.w3.org/2000/svg", "text");
      stamp.id = element.element_id.toString()
      stamp.setAttribute("x", fixedString(element.property.pos[0]))
      stamp.setAttribute("y", fixedString(element.property.pos[1]))
      stamp.setAttribute("text-anchor", "middle")
      const fontSize = element.bold * 4
      stamp.setAttribute("style", `font-size: ${fontSize}px; fill: ${element.color}; opacity: ${element.opacity};`)
      const stampLines = JSON.parse(element.property.text)
      for (let i = 0; i < stampLines.length; i++) {
        const line = document.createElementNS("http://www.w3.org/2000/svg", "tspan");
        line.textContent = stampLines[i]
        line.setAttribute("x", fixedString(element.property.pos[0]))
        line.setAttribute("dy", fontSize.toFixed(2))
        line.setAttribute("text-anchor", "middle")
        stamp.append(line)
      }
      board.appendChild(stamp)
    }
  }
}

/**
 * @param {HTMLElement|SVGAElement} element 
 */
function appendChild(element) {
  const id = parseInt(element.id)
  let inserted = false;
  for (const child of board.children) {
    if (parseInt(child.id) < id) {
      board.insertBefore(element, child);
      inserted = true;
      break;
    }
  }
  if (!inserted) {
    board.appendChild(element);
  }
}

function UUIDv7() {
  let uuid = 'tttttttt-tttt-7xxx-yxxx-xxxxxxxxxxxx'
  uuid = uuid.replace(/[xy]/g, function (c) {
    const r = Math.trunc(Math.random() * 16);
    const v = c == 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  })
  uuid = uuid.replace(/^[t]{8}-[t]{4}/, function () {
    const unixtimestamp = Date.now().toString(16).padStart(12, '0');
    return unixtimestamp.slice(0, 8) + '-' + unixtimestamp.slice(8);
  });
  return uuid
}

// MARK: Export *
function downloadPNG() {// DL
  const svg = board.cloneNode(true)
  // @ts-expect-error
  svg.setAttribute("style", "fill:none;")
  const svgData = new XMLSerializer().serializeToString(svg);

  const img = new Image()
  const url = URL.createObjectURL(new Blob([svgData], { type: "image/svg+xml" }))

  img.onload = function () {
    const canvas = document.createElement("canvas");
    canvas.width = board.clientWidth;
    canvas.height = board.clientHeight;
    console.log(board.getBoundingClientRect())
    const ctx = canvas.getContext("2d");
    if (!ctx) return
    ctx.drawImage(img, 0, 0);

    const dl = document.createElement("a");
    dl.href = canvas.toDataURL("image/png");
    const now = new Date()
    const timestamp = `${now.getFullYear()}-${(now.getMonth() + 1).toString().padStart(2, "0")}-${(now.getDate()).toString().padStart(2, "0")}_${(now.getHours()).toString().padStart(2, "0")}-${(now.getMinutes()).toString().padStart(2, "0")}-${(now.getSeconds()).toString().padStart(2, "0")}`
    dl.setAttribute("download", `${room}_${timestamp}.png`);
    dl.dispatchEvent(new MouseEvent("click"));
  }
  img.src = url
};
pngInput.addEventListener("click", downloadPNG)

function downloadSVG() {// DL
  const svg = board.cloneNode(true)
  // @ts-expect-error
  svg.setAttribute("style", "fill:none;")
  const svgData = new XMLSerializer().serializeToString(svg);
  const dl = document.createElement("a");
  dl.href = "data:image/svg+xml;charset=utf-8;base64," + btoa(unescape(encodeURIComponent(svgData)))
  const now = new Date()
  const timestamp = `${now.getFullYear()}-${(now.getMonth() + 1).toString().padStart(2, "0")}-${(now.getDate()).toString().padStart(2, "0")}_${(now.getHours()).toString().padStart(2, "0")}-${(now.getMinutes()).toString().padStart(2, "0")}-${(now.getSeconds()).toString().padStart(2, "0")}`
  dl.setAttribute("download", `${room}_${timestamp}.svg`);
  dl.dispatchEvent(new MouseEvent("click"));
};
svgInput.addEventListener("click", downloadSVG)

// MARK: generic method()
/**
 * @param {number} n 
 * @returns {number}
*/
function fixedNumber(n) {
  return parseFloat(fixedString(n))
}
/**
 * @param {number} n 
 * @returns {string}
*/
function fixedString(n) {
  return n.toFixed(2)
}


/**
 * 
 * @param {[number,number]} pos 
 */
function isSidemenuArea(pos) {
  /** @type {SVGElement[]|HTMLElement[]} */
  // @ts-expect-error
  const targetElements = document.elementsFromPoint(pos[0], pos[1])

  for (let i = 0; i < targetElements.length; i++) {
    if (targetElements[i].classList.contains("side-menu") || targetElements[i].classList.contains("side-menu-area")) {
      return true
    }
  }
}