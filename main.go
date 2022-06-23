package main

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atomu21263/atomicgo"
	"golang.org/x/net/websocket"
)

var (
	Listen = ":25300"
	Rooms  = map[string]*Room{}
	Count  = 0
	Save   = atomicgo.GetGoDir() + "rooms/"
)

type Room struct {
	Jsons      []string
	Websockets map[int]*websocket.Conn
}

type WebSocketRes struct {
	Type  string `json:"type"`
	Layer string `json:"layer"`
	Data  string `json:"data"`
}

func main() {
	defer func() {
		for room, roomData := range Rooms {
			log.Println("Data Saving Room:" + room)
			atomicgo.WriteFileBaffer(Save+room+".txt", []byte(Array2String(roomData.Jsons)), 0666)
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
	w.Write(bytes)
	if !atomicgo.CheckFile(Save + room + ".txt") {
		atomicgo.CreateFile(Save + room + ".txt")
		log.Println("Create SaveData:", Save+room+".txt")
	}
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
					if strings.Contains(Json, `id="`+jsonData.Data+`"`) {
						continue
					}
					dummyJsons = append(dummyJsons, Json)
				}
				roomData.Jsons = dummyJsons
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
