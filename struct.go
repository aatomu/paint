package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

// MARK: HTTP
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

type Room struct {
	sync.RWMutex
	Conn map[int64]*websocket.Conn
}

// MARK: Packet
type PacketEvent struct {
	PacketId string `json:"packet_id"`
	Name     string `json:"name"`
	// c<=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
	// s=>c :"success"|"error"
	Operation string          `json:"operation"`
	Data      json.RawMessage `json:"data"`
}

type PacketEventMouse struct {
	Pos [2]float64 `json:"pos"`
}

type PacketEventCreate struct {
	ElementId   string          `json:"element_id"`
	ElementType string          `json:"element_type"` // "pen"|"line"|"stamp"
	Bold        float64         `json:"bold"`
	Color       string          `json:"color"`
	Opacity     float64         `json:"opacity"`
	Property    json.RawMessage `json:"property"`
}

type PacketEventDelete struct {
	Target string `json:"target"`
}

type PacketEventUndo struct{}

type PacketEventRedo struct{}

type PacketEventClear struct{}

type PacketEventSuccess struct {
	EventId  string `json:"event_id"`
	PacketId string `json:"packet_id"`
}
type PacketEventError struct {
	PacketId string `json:"packet_id"`
	Message  string `json:"message"`
}

// MARK: SQL
type TableBoards struct {
	boardId    string
	name       string
	created_at int64
}

type TableElements struct {
	elementId   string
	boardId     string
	elementType string
	bold        float64
	color       string
	opacity     float64
	property    string
	deleted     bool
}

type TableEvents struct {
	id         int64 // Result only
	eventId    string
	boardId    string
	elementId  string
	username   string
	operation  string
	undo       bool
	disable    bool  // Result only
	created_at int64 // Result only
}

type Transaction struct {
	transaction *sql.Tx
}

// MARK: Generic
type PropertyPen struct {
	D string `json:"d"`
}
type PropertyLine struct {
	Start [2]float64 `json:"start"`
	End   [2]float64 `json:"end"`
}
type PropertyStamp struct {
	Pos  [2]float64 `json:"pos"`
	Text string     `json:"text"` // ["aaa","bbb", ...]
}

type FunctionResult struct {
	err error
	msg FunctionMessage
}

type FunctionMessage struct {
	server string
	client string
}

func (se FunctionResult) Ok() bool {
	return se.err == nil
}

func NewDecoder(r io.Reader) (d *json.Decoder) {
	d = json.NewDecoder(r)
	d.DisallowUnknownFields()
	return
}

func (p PacketEvent) Set(d any) PacketEvent {
	data, _ := json.Marshal(d)
	p.Data = data
	return p
}
