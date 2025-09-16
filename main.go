package main

import (
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var (
	Listen    = ":1026"
	Rooms     = map[string]*Room{}
	RoomsLock = sync.RWMutex{}
)

type Room struct {
	sync.RWMutex
	Objects []string
	Conn    map[int64]*websocket.Conn
}

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
	os.Chdir(filepath.Dir(file))

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

func WebsocketRequest(w *websocket.Conn) {
	room := w.Request().URL.Query().Get("id")
	if room == "" {
		w.Close()
		return
	}
	id := time.Now().UnixNano()

	// Save session
	RoomsLock.Lock()
	r, ok := Rooms[room]
	if !ok {
		r = &Room{
			Objects: []string{},
			Conn:    map[int64]*websocket.Conn{},
		}
		Rooms[room] = r
	}
	RoomsLock.Unlock()

	r.Lock()
	r.Conn[id] = w
	r.Unlock()

	defer func() {
		r.Lock()
		delete(r.Conn, id)
		r.Unlock()
		w.Close()
	}()

	br := &byteReader{w}
	for {
		size, err := binary.ReadUvarint(br)
		if err != nil {
			return
		}
		buf := make([]byte, size)

		log.Println("Size", size)
		_, err = io.ReadFull(w, buf)
		if err != nil {
			return
		}
		log.Println("Readed", size)

		Rooms[room].RLock()
		for _, v := range Rooms[room].Conn {
			v.Write(buf)
		}
		Rooms[room].RUnlock()
	}
}

type byteReader struct {
	io.Reader
}

func (br *byteReader) ReadByte() (byte, error) {
	var b [1]byte
	if _, err := br.Read(b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}
