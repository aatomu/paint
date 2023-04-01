package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atomu21263/atomicgo"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/net/websocket"
)

var (
	Listen = ":1025"
	Rooms  = map[string]*RoomInfo{}
	Save   = atomicgo.GetGoDir() + "rooms/"
)

type RoomInfo struct {
	Password     string                  `json:"password"`
	Limit        int                     `json:"limit"`
	Jsons        []string                `json:"jsons"`
	Websockets   map[int]*websocket.Conn `json:"-"`
	isUpUpdating bool                    `json:"-"`
}

type WebSocketRes struct {
	Type  string `json:"type"`
	Layer string `json:"layer"`
	Data  string `json:"data"`
}

func main() {
	defer func() {
		for roomID, roomData := range Rooms {
			if len(roomData.Jsons) > 0 {
				saveJson(roomID)
			} else {
				os.Remove(Save + roomID + ".json")
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
	// RoomID取得
	roomID := r.URL.Query().Get("room")
	// IDがなかったらエラーを返す
	if roomID == "" {
		w.Write([]byte("<body style=\"text-align: center;\"><h1>Unknown Room</h1>\nPlease Access " + r.Host + "/?room=&lt;RoomName&gt;</body>"))
		return
	}

	// アクセスログ
	log.Println("Access:", r.RemoteAddr, "Room:", roomID)

	// 読み込み
	bytes, _ := os.ReadFile("./art.html")
	indexHTML := string(bytes)
	roomData, isRoomEnable := Rooms[roomID]
	// 変数書き換え
	indexHTML = strings.ReplaceAll(indexHTML, "{Room}", roomID)
	if isRoomEnable {
		indexHTML = strings.ReplaceAll(indexHTML, "{Connect}", fmt.Sprintf("%d", len(roomData.Websockets)))
	} else {
		indexHTML = strings.ReplaceAll(indexHTML, "{Connect}", "0")
	}
	indexHTML = strings.ReplaceAll(indexHTML, "{HeadURL}", fmt.Sprintf("http:/%s/photo.png?room=%s&date=%d", r.Host, roomID, time.Now().Unix()))
	// 送信
	w.Write([]byte(indexHTML))

	// 部屋作成
	if !isRoomEnable {
		file, _ := os.ReadFile(Save + roomID + ".json")
		Room := RoomInfo{
			Limit:        1000,
			Websockets:   map[int]*websocket.Conn{},
			isUpUpdating: false,
		}
		json.Unmarshal(file, &Room)
		log.Println("Load SaveData:", Save+roomID+".json")
		Rooms[roomID] = &Room
	}
}

// 現在のを表示
func NowPaint(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		return
	}

	go func(roomIDFunc string) {
		roomData, isRoomEnable := Rooms[roomIDFunc]
		if isRoomEnable && !roomData.isUpUpdating {
			roomData.isUpUpdating = true
			makePng(roomData.Jsons, roomIDFunc)
			roomData.isUpUpdating = false
		}
	}(roomID)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	image, _ := os.ReadFile("./image/" + roomID + ".png")
	w.Write(image)
}

// ウェブソケット処理
func WebSocketResponse(ws *websocket.Conn) {
	defer func() {
		ws.Close()
	}()

	// Socket保存用
	roomID := ""
	websocket.Message.Receive(ws, &roomID)
	log.Println("WebSocket:", ws.RemoteAddr(), "Room:", roomID)
	roomData := Rooms[roomID]
	pos := len(roomData.Websockets)
	roomData.Websockets[pos] = ws
	defer func(posFunc int) {
		delete(roomData.Websockets, posFunc)
	}(pos)

	// 今までのデータを転送
	for _, svg := range roomData.Jsons {
		if svg == "" {
			continue
		}
		websocket.Message.Send(ws, svg)
	}
	websocket.Message.Send(ws, fmt.Sprintf(`{"type":"info","layer":"%d","data":"line_max"}`, roomData.Limit))
	websocket.Message.Send(ws, `{"type":"end","layer":"","data":""}`)

	// CloseCheck
	var isClose = false
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			<-ticker.C
			err := websocket.Message.Send(ws, ``)
			if err != nil {
				isClose = true
				return
			}
		}
	}()

	// 相互通信
	for {
		// 受信
		str := ""
		websocket.Message.Receive(ws, &str)
		if str == "" {
			if isClose {
				return
			}
			continue
		}
		// 受信データ
		var jsonData WebSocketRes
		json.Unmarshal([]byte(str), &jsonData)

		// ファイルチェック
		if _, err := os.Stat(Save + roomID + ".json"); err != nil {
			os.Create(Save + roomID + ".json")
			log.Println("Create:", Save+roomID+".json")
		}

		// 表示
		go func() {
			// ほかのに送信
			for _, socket := range roomData.Websockets {
				if socket != ws {
					websocket.Message.Send(socket, str)
				}
			}
			switch jsonData.Type {
			case "append", "eraser":
				if len(roomData.Jsons) >= roomData.Limit {
					websocket.Message.Send(ws, `{"type":"info","layer":"","data":"line_limit"}`)
					lineID := atomicgo.StringReplace(jsonData.Data, "$1", `.*id=\"([0-9]+)\".*`)
					websocket.Message.Send(ws, fmt.Sprintf(`{"type":"delete","layer":"","data":"%s"}`, lineID))
					return
				}
				roomData.Jsons = append(roomData.Jsons, str)
			case "clear":
				for _, socket := range roomData.Websockets {
					websocket.Message.Send(socket, `{"type":"info","layer":"","data":"line_unlimit"}`)
				}
				roomData.Jsons = []string{}
			case "delete":
				for _, socket := range roomData.Websockets {
					websocket.Message.Send(socket, `{"type":"info","layer":"","data":"line_unlimit"}`)
				}
				dummyJsons := []string{}
				for _, Json := range roomData.Jsons {
					if strings.Contains(Json, jsonData.Data) {
						continue
					}
					dummyJsons = append(dummyJsons, Json)
				}
				roomData.Jsons = dummyJsons
			case "save":
				saveJson(roomID)
			case "limit":
				// 切り分け
				data := strings.Split(jsonData.Data, ",")
				if len(data) < 2 {
					websocket.Message.Send(ws, `{"type":"info","layer":"","data":"input_unmatch"}`)
					break
				}
				// パスワード設定
				if roomData.Password == "" {
					roomData.Password = data[0]
				}
				// 変更
				if roomData.Password == data[0] {
					limit, err := strconv.Atoi(data[1])
					if err != nil {
						websocket.Message.Send(ws, `{"type":"info","layer":"","data":"input_unmatch"}`)
						break
					}
					roomData.Limit = limit
					for _, socket := range roomData.Websockets {
						websocket.Message.Send(socket, fmt.Sprintf(`{"type":"info","layer":"%d","data":"line_max"}`, limit))
					}
				} else {
					websocket.Message.Send(ws, `{"type":"info","layer":"","data":"input_unmatch"}`)
				}
			}
			// 自動セーブ
			if len(roomData.Jsons)%50 == 0 {
				saveJson(roomID)
			}
			log.Println("Catch:  Room:", roomID, "Data:", atomicgo.StringCut(str, 100)+"...")
		}()
	}
}

func Array2String(sArray []string) (s string) {
	for _, svg := range sArray {
		if svg == "" {
			continue
		}
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

func saveJson(roomID string) {
	roomData, isRoomEnable := Rooms[roomID]
	if !isRoomEnable {
		return
	}
	bytes, _ := json.MarshalIndent(roomData, "", "  ")
	jsonFile, _ := os.Create(Save + roomID + ".json")
	defer jsonFile.Close()
	writer := bufio.NewWriter(jsonFile)
	writer.Write(bytes)
	writer.Flush()
	log.Println("Save:", Save+roomID+".json")
}
