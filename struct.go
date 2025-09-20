package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	// c=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
	// s=>c :"success"|"error"|"transfer"
	Operation string          `json:"operation"`
	Data      json.RawMessage `json:"data"`
}

type PacketEventMouse struct {
	Pos [2]float64 `json:"pos"`
}

type PacketEventCreate struct {
	ElementId string          `json:"element_id"`
	Type      string          `json:"type"` // "pen"|"line"|"stamp"
	Bold      float64         `json:"bold"`
	Color     string          `json:"color"`
	Opacity   float64         `json:"opacity"`
	Property  json.RawMessage `json:"property"`
}

type PacketEventDelete struct {
	Target string `json:"target"`
}

type PacketEventUndo struct {
	Target string `json:"target"`
}

type PacketEventRedo struct {
	Target string `json:"target"`
}

type PacketEventClear struct{}

type PacketEventSuccess struct {
	PacketId string `json:"packet_id"`
}
type PacketEventError struct {
	PacketId string `json:"packet_id"`
	Message  string `json:"message"`
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

// MARK: SQL commands
func CreateTables(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS boards (
		board_id               TEXT    PRIMARY KEY,
		name             TEXT    NOT NULL,
		create_timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("%s(in boards)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS elements (
		element_id TEXT    NOT NULL,
		board_id   TEXT    NOT NULL,
		type       TEXT    NOT NULL,
		bold       REAL    NOT NULL,
		color      TEXT    NOT NULL,
		opacity    REAL    NOT NULL,
		property   TEXT    NOT NULL,
		deleted    INTEGER NOT NULL,
		PRIMARY KEY (element_id,board_id),
		FOREIGN KEY(board_id) REFERENCES boards(board_id)
	)`)
	if err != nil {
		return fmt.Errorf("%s(in elements)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS events (
		event_id   TEXT    PRIMARY KEY,
		board_id   TEXT    NOT NULL,
		element_id TEXT,
		username   TEXT    NOT NULL,
		operation  TEXT    NOT NULL,
		timestamp  INTEGER NOT NULL,
		FOREIGN KEY(board_id) REFERENCES boards(board_id),
		FOREIGN KEY(element_id) REFERENCES elements(element_id)
	)`)
	if err != nil {
		return fmt.Errorf("%s(in events)", err.Error())
	}

	return nil
}

func GetBoardId(name string) (boardId string, err error) {
	err = DB.QueryRow("SELECT board_id FROM boards WHERE name = ?", name).Scan(&boardId)
	if err == sql.ErrNoRows {
		boardId = uuid.New().String()
		now := time.Now().Unix()
		_, err = DB.Exec("INSERT INTO boards (board_id, name, create_timestamp) VALUES (?, ?, ?)", boardId, name, now)
		return
	}
	return
}

func NewDecoder(r io.Reader) (d *json.Decoder) {
	d = json.NewDecoder(r)
	d.DisallowUnknownFields()
	return
}
