package main

import (
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aatomu/atomicgo/files"
	"golang.org/x/net/websocket"
)

var (
	Listen = ":1026"
)

type fallbackFileSystem struct {
	fs http.FileSystem
}

func (ff fallbackFileSystem) Open(name string) (http.File, error) {
	f, err := ff.fs.Open(name)
	if err == nil {
		return f, nil
	}

	// ファイルが見つからない場合は .html を付けて再試行
	if !strings.HasSuffix(name, ".html") {
		if f2, err2 := ff.fs.Open(name + ".html"); err2 == nil {
			return f2, nil
		}
	}

	// ディレクトリなら index.html を見る
	if strings.HasSuffix(name, "/") {
		if f3, err3 := ff.fs.Open(path.Join(name, "index.html")); err3 == nil {
			return f3, nil
		}
	}

	return nil, err
}

func main() {
	// Work dir
	_, file, _, _ := runtime.Caller(0)
	goDir := filepath.Dir(file) + "/"
	files.SetWorkDir(goDir)

	// Http handle
	assets := http.FileServer(fallbackFileSystem{http.Dir("./assets")})
	http.Handle("/", middleware(assets))
	http.Handle("/ws", middleware(websocket.Handler(WebsocketRequest)))

	// Boot Server
	log.Println("Http Server Boot")
	err := http.ListenAndServe(Listen, nil)
	if err != nil {
		log.Println("Failed Listen:", err)
		return
	}
}

func middleware(h http.Handler) http.Handler {
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
