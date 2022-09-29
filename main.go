package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atomu21263/atomicgo"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/net/websocket"
)

var (
	Listen = ":25300"
	Rooms  = map[string]*Room{}
	Count  = 0
	Save   = atomicgo.GetGoDir() + "rooms/"
)

type Room struct {
	Jsons        []string
	Websockets   map[int]*websocket.Conn
	isUpUpdating bool
}

type WebSocketRes struct {
	Type  string `json:"type"`
	Layer string `json:"layer"`
	Data  string `json:"data"`
}

func main() {
	defer func() {
		for room, roomData := range Rooms {
			if len(roomData.Jsons) > 0 {
				log.Println("Data Saving Room:" + room)
				atomicgo.WriteFileBaffer(Save+room+".txt", []byte(Array2String(roomData.Jsons)), 0666)
			}
		}
	}()
	// 移動
	_, file, _, _ := runtime.Caller(0)
	goDir := filepath.Dir(file) + "/"
	atomicgo.MoveWorkDir(goDir)
	// 保存先
	if !atomicgo.CheckFile(Save) {
		atomicgo.CreateDir(Save, 0766)
		log.Println("Create SaveDir:", Save)
	}
	// アクセス先
	http.HandleFunc("/", HttpResponse)
	http.HandleFunc("/photo.png", NowPaint)
	http.Handle("/websocket", websocket.Handler(WebSocketResponse))
	// Web鯖 起動
	go func() {
		log.Println("Http Server Boot")
		err := http.ListenAndServe(Listen, nil)
		if err != nil {
			log.Println("Failed Listen:", err)
			return
		}
	}()
	atomicgo.StopWait()
}

// ページ表示
func HttpResponse(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		w.Write([]byte("<body style=\"text-align: center;\"><h1>Unknown Room</h1>\nPlease Access " + r.Host + "/?room=&lt;RoomName&gt;</body>"))
		return
	}
	log.Println("Access:", r.RemoteAddr, "Room:", room)
	bytes, _ := atomicgo.ReadFile("./art.html")
	str := string(bytes)
	roomData, ok := Rooms[room]
	if ok {
		str = atomicgo.StringReplace(str, fmt.Sprintf("%d", len(roomData.Websockets)), "{Connect}")
	} else {
		str = atomicgo.StringReplace(str, "0", "{Connect}")
	}
	str = atomicgo.StringReplace(str, fmt.Sprintf("http:/%s/photo.png?room=%s&date=%d", r.Host, room, time.Now().Unix()), "{HeadURL}")
	w.Write([]byte(str))
	if _, ok := Rooms[room]; !ok {
		s, _ := atomicgo.ReadFile(Save + room + ".txt")
		jsons := strings.Split(string(s), "\n")
		log.Println("Load SaveData:", Save+room+".txt")
		Rooms[room] = &Room{
			Jsons:      jsons,
			Websockets: map[int]*websocket.Conn{},
		}
	}
}

// 現在のを表示
func NowPaint(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		return
	}

	roomData, ok := Rooms[room]
	go func(r *Room) {
		if ok && !roomData.isUpUpdating {
			roomData.isUpUpdating = true
			go makePng(roomData.Jsons, room)
			roomData.isUpUpdating = false
		}
	}(roomData)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	image, _ := os.ReadFile("./image/" + room + ".png")
	w.Write(image)
}

// ウェブソケット処理
func WebSocketResponse(ws *websocket.Conn) {
	var err error
	defer func() {
		ws.Close()
	}()

	// Socket保存用
	room := ""
	websocket.Message.Receive(ws, &room)
	log.Println("WebSocket:", ws.RemoteAddr(), "Room:", room)
	pos := Count
	Count++
	roomData := Rooms[room]
	roomData.Websockets[pos] = ws
	defer func() {
		delete(roomData.Websockets, pos)
	}()

	// 今までのデータを転送
	for _, svg := range roomData.Jsons {
		if svg == "" {
			continue
		}
		err = websocket.Message.Send(ws, svg)
		if err != nil {
			return
		}
	}
	err = websocket.Message.Send(ws, `{"type":"end","layer":"","data":""}`)
	if err != nil {
		return
	}

	// 相互通信
	for {
		// 受信
		str := ""
		err := websocket.Message.Receive(ws, &str)
		if err != nil {
			return
		}
		// 受信データ
		log.Println("Catch: Room:", room, "Data:", atomicgo.StringCut(str, 100)+"...")
		var jsonData WebSocketRes
		json.Unmarshal([]byte(str), &jsonData)
		// ファイルチェック
		if !atomicgo.CheckFile(Save + room + ".txt") {
			atomicgo.CreateFile(Save + room + ".txt")
			log.Println("Create SaveData:", Save+room+".txt")
		}
		// 表示
		go func() {
			for _, socket := range roomData.Websockets {
				if socket != ws {
					websocket.Message.Send(socket, str)
				}
			}
			switch jsonData.Type {
			case "append", "eraser":
				roomData.Jsons = append(roomData.Jsons, str)
			case "clear":
				roomData.Jsons = []string{}
			case "delete":
				dummyJsons := []string{}
				for _, Json := range roomData.Jsons {
					if strings.Contains(Json, jsonData.Data) {
						continue
					}
					dummyJsons = append(dummyJsons, Json)
				}
				roomData.Jsons = dummyJsons
			case "save":
				atomicgo.WriteFileBaffer(Save+room+".txt", []byte(Array2String(roomData.Jsons)), 0666)
			}
			if len(roomData.Jsons)%10 == 0 {
				atomicgo.WriteFileBaffer(Save+room+".txt", []byte(Array2String(roomData.Jsons)), 0666)
			}
		}()
	}
}

func Array2String(sArray []string) (s string) {
	for _, svg := range sArray {
		s += "\n" + svg
	}
	return
}

func makePng(jsons []string, roomName string) {
	var jsonData WebSocketRes

	layer0, layer1, layer2, layer3, layer4 := "", "", "", "", ""
	for _, str := range jsons {
		json.Unmarshal([]byte(str), &jsonData)
		if jsonData.Type != "append" {
			continue
		}
		switch jsonData.Layer {
		case "layer0":
			layer0 += jsonData.Data + "\n"
		case "layer1":
			layer1 += jsonData.Data + "\n"
		case "layer2":
			layer2 += jsonData.Data + "\n"
		case "layer3":
			layer3 += jsonData.Data + "\n"
		case "layer4":
			layer4 += jsonData.Data + "\n"
		}
	}

	svg := `` +
		"<svg id=\"layers\" class=\"area\" width=\"1280px\" height=\"720px\" xmlns=\"http://www.w3.org/2000/svg\" style=\"fill: none;\">\n" +
		"	<rect x=\"0\" y=\"0\" width=\"1280\" height=\"720\" fill=\"white\"/>\n" +
		"	<g id=\"layer0\" style=\"display: inline;\">" + layer0 + "</g>\n" +
		"	<g id=\"layer1\" style=\"display: inline;\">" + layer1 + "</g>\n" +
		"	<g id=\"layer2\" style=\"display: inline;\">" + layer2 + "</g>\n" +
		"	<g id=\"layer3\" style=\"display: inline;\">" + layer3 + "</g>\n" +
		"	<g id=\"layer4\" style=\"display: inline;\">" + layer4 + "</g>\n" +
		"</svg>"

	readerSVG := strings.NewReader(svg)
	icon, _ := oksvg.ReadIconStream(readerSVG)
	width, height := 1280, 720
	icon.SetTarget(0, 0, float64(width), float64(height))
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	icon.Draw(rasterx.NewDasher(width, height, rasterx.NewScannerGV(width, height, rgba, rgba.Bounds())), 1)

	f, _ := os.Create("./image/" + roomName + ".png")
	defer f.Close()
	png.Encode(f, rgba)
	log.Println("Update: ./image/" + roomName + ".png")
}
