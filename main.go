package main

import (
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/aatomu/atomicgo/files"
	"golang.org/x/net/websocket"
)

var (
	Listen = ":1026"
)

func main() {
	// Work dir
	_, file, _, _ := runtime.Caller(0)
	goDir := filepath.Dir(file) + "/"
	files.SetWorkDir(goDir)

	// Http handle
	assets := http.FileServer(http.Dir("./assets"))
	http.Handle("/", middleWare(assets))
	http.Handle("/ws", middleWare(websocket.Handler(WebsocketRequest)))

	// Boot Server
	log.Println("Http Server Boot")
	err := http.ListenAndServe(Listen, nil)
	if err != nil {
		log.Println("Failed Listen:", err)
		return
	}
}

func middleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("IP:%s, Method:%s, URI:%s, Header:%v", r.RemoteAddr, r.Method, r.URL, r.Header)
		// Compute
		h.ServeHTTP(w, r)
	})
}

func HttpRequest(w http.ResponseWriter, r *http.Request) {

}

func WebsocketRequest(w *websocket.Conn) {
	defer w.Close()
}
