package main

import (
	"database/sql"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

type Room struct {
	sync.RWMutex
	Conn map[int64]*websocket.Conn
}

type Event struct {
	EventId   int64
	Timestamp int64 // UNIX timestamp
	User      string
	Operation string
}

type EventObjectAdd struct {
	EventId int64
	Id      string
	Bold    float64
	Color   string
	Opacity float64
	Type    string
}

type ObjectPen struct {
	Id string
	D  string
}
type ObjectLine struct {
	Id     string
	StartX float64
	StartY float64
	EndX   float64
	EndY   float64
}
type ObjectStamp struct {
	Id   string
	PosX float64
	PosY float64
	Text string // ["aaaa","bbb", ...]
}

// 削除操作
type EventObjectRemove struct {
	EventId int64
	Target  string // TypeScriptのstringに対応
}

func CreateTables(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS event (
		event_id INTEGER PRIMARY KEY,
		timestamp TEXT NOT NULL,
		user TEXT NOT NULL,
		operation TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS event_object_add (
		event_id INTEGER PRIMARY KEY,
		id TEXT NOT NULL,
		bold REAL NOT NULL,
		color TEXT NOT NULL,
		opacity REAL NOT NULL,
		type TEXT NOT NULL,
		FOREIGN KEY(event_id) REFERENCES event(event_id)
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS object_pen (
		id TEXT PRIMARY KEY,
		d TEXT NOT NULL,
		FOREIGN KEY(id) REFERENCES event_object_add(id)
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS object_line (
		id TEXT PRIMARY KEY,
		start_x REAL NOT NULL,
		start_y REAL NOT NULL,
		end_x REAL NOT NULL,
		end_y REAL NOT NULL,
		FOREIGN KEY(id) REFERENCES event_object_add(id)
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS object_stamp (
		id TEXT PRIMARY KEY,
		pos_x REAL NOT NULL,
		pos_y REAL NOT NULL,
		text TEXT NOT NULL,
		FOREIGN KEY(id) REFERENCES event_object_add(id)
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS event_object_remove (
		event_id INTEGER PRIMARY KEY,
		target TEXT NOT NULL,
		FOREIGN KEY(event_id) REFERENCES event(event_id)
	)`)
	if err != nil {
		return err
	}

	return nil
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
