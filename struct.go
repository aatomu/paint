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
	Id string `json:"id"`
	// c=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
	// s=>c :"success"|"error"|"transfer"
	Operation string `json:"operation"`
	Data      string `json:"data"`
}

type PacketEventMouse struct {
	Pos [2]float64 `json:"pos"`
}

type PacketEventCreate struct {
	Id       string  `json:"id"`
	Type     string  `json:"type"` // "pen"|"line"|"stamp"
	Bold     float64 `json:"bold"`
	Color    string  `json:"color"`
	Opacity  float64 `json:"opacity"`
	Property string  `json:"property"`
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
	Text string     `json:"text"`
}

// MARK: SQL commands
func CreateTables(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS boards (
		id               TEXT    PRIMARY KEY,
		name             TEXT    NOT NULL,
		create_timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("%s(in boards)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS elements (
		id       TEXT    NOT NULL,
		board_id TEXT    NOT NULL,
		type     TEXT    NOT NULL,
		bold     REAL    NOT NULL,
		color    TEXT    NOT NULL,
		opacity  REAL    NOT NULL,
		property TEXT    NOT NULL,
		deleted  INTEGER NOT NULL,
		PRIMARY KEY (id,board_id),
		FOREIGN KEY(board_id) REFERENCES boards(id)
	)`)
	if err != nil {
		return fmt.Errorf("%s(in elements)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS events (
		id         TEXT    PRIMARY KEY,
		board_id   TEXT    NOT NULL,
		element_id TEXT,
		username   TEXT    NOT NULL,
		operation  TEXT    NOT NULL,
		timestamp  INTEGER NOT NULL,
		FOREIGN KEY(board_id) REFERENCES boards(id),
		FOREIGN KEY(element_id) REFERENCES elements(id)
	)`)
	if err != nil {
		return fmt.Errorf("%s(in events)", err.Error())
	}

	return nil
}

func GetBoardId(name string) (id string, err error) {
	err = DB.QueryRow("SELECT id FROM boards WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		id = uuid.New().String()
		now := time.Now().Unix()
		_, err = DB.Exec("INSERT INTO boards (id, name, create_timestamp) VALUES (?, ?, ?)", id, name, now)
		return
	}
	return
}

// MARK: Websocket
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

func NewDecoder(t string) (d *json.Decoder) {
	d = json.NewDecoder(strings.NewReader(t))
	d.DisallowUnknownFields()
	return
}
