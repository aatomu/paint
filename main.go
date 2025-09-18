package main

import (
	"database/sql"
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/websocket"
)

var (
	Listen    = ":1026"
	Rooms     = map[string]*Room{}
	RoomsLock = sync.RWMutex{}
)

func main() {
	// Work dir
	_, file, _, _ := runtime.Caller(0)
	os.Chdir(filepath.Dir(file))

	// Open SQL
	db, err := sql.Open("sqlite3", "./rooms.db")
	if err != nil {
		log.Panicf("Failed Open Database: %v", err)
	}
	defer db.Close()

	err = CreateTables(db)
	if err != nil {
		log.Panicf("Failed Open Database: %v", err)
	}

	// Http handle
	assets := http.FileServer(fallbackFileSystem{http.Dir("./assets")})
	http.Handle("/", middleware(assets))
	http.Handle("/ws", middleware(websocket.Handler(WebsocketRequest)))

	// Boot Server
	log.Println("Http Server Boot")
	err = http.ListenAndServe(Listen, nil)
	if err != nil {
		log.Panicf("Failed Listen Http Server: %v", err)
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
			Conn: map[int64]*websocket.Conn{},
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

		_, err = io.ReadFull(w, buf)
		if err != nil {
			return
		}

		Rooms[room].RLock()
		for _, v := range Rooms[room].Conn {
			v.Write(buf)
		}
		Rooms[room].RUnlock()
	}
}
