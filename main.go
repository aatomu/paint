package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/websocket"
)

var (
	Listen    = ":1026"
	Rooms     = map[string]*Room{}
	RoomsLock = sync.RWMutex{}
	DB        *sql.DB
)

// MARK: Main
func main() {
	// Work dir
	_, file, _, _ := runtime.Caller(0)
	os.Chdir(filepath.Dir(file))

	// Open SQL
	var err error
	DB, err = sql.Open("sqlite3", "./rooms.db")
	if err != nil {
		log.Panicf("Failed Open Database: %v", err)
	}
	defer DB.Close()

	err = CreateTables(DB)
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

// MARK: HTTP middle
func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("IP:%s, Method:%s, URI:%s, Header:%v", r.RemoteAddr, r.Method, r.URL, r.Header)
		// Compute
		h.ServeHTTP(w, r)
	})
}

// MARK: Websocket
func WebsocketRequest(w *websocket.Conn) {
	room := w.Request().URL.Query().Get("id")
	if room == "" {
		w.Close()
		return
	}
	id := time.Now().UnixNano()

	// Board check
	boardId, err := GetBoardId(room)
	if err != nil {
		w.Close()
		log.Println("SQL Error in \"func GetBoardId()\": ", err)
		return
	}

	// Save session
	RoomsLock.Lock()
	r, ok := Rooms[room]
	if !ok {
		r = &Room{
			Conn: map[int64]*websocket.Conn{},
		}
		Rooms[room] = r
	}

	r.Lock()
	r.Conn[id] = w
	r.Unlock()
	RoomsLock.Unlock()

	defer func() {
		r.Lock()
		delete(r.Conn, id)
		r.Unlock()
		if len(r.Conn) == 0 {
			RoomsLock.Lock()
			delete(Rooms, room)
			RoomsLock.Unlock()
		}
		w.Close()
	}()

	var packet string
	for {
		websocket.Message.Receive(w, &packet)

		// MARK: Validation
		var event PacketEvent
		err = NewDecoder(strings.NewReader(packet)).Decode(&event)
		if err != nil {
			// ! Invalid Event
			w.Write([]byte(err.Error()))
			continue
		}

		dataDecoder := NewDecoder(bytes.NewReader(event.Data))
		switch event.Operation {
		case "mouse": // MARK: >Mouse
			{
				err = dataDecoder.Decode(&PacketEventMouse{})
				if err != nil {
					// ! Invalid Event Property
					w.Write([]byte(err.Error()))
					continue
				}
			}

		case "create": // MARK: >Create
			{
				var create PacketEventCreate
				err = dataDecoder.Decode(&create)
				if err != nil {
					// ! Invalid Event Property
					w.Write([]byte(err.Error()))
					continue
				}

				// Property check
				propertyDecoder := NewDecoder(bytes.NewReader(create.Property))
				switch create.Type {
				case "pen":
					{
						err = propertyDecoder.Decode(&PropertyPen{})
						if err != nil {
							// ! Invalid Element Property
							w.Write([]byte(err.Error()))
							continue
						}
					}
				case "line":
					{
						err = propertyDecoder.Decode(&PropertyLine{})
						if err != nil {
							// ! Invalid Element Property
							w.Write([]byte(err.Error()))
							continue
						}
					}
				case "stamp":
					{
						err = propertyDecoder.Decode(&PropertyStamp{})
						if err != nil {
							// ! Invalid Element Property
							w.Write([]byte(err.Error()))
							continue
						}
					}
				default:
					{
						// ! Invalid Element Property
						w.Write([]byte("unknown type"))
						continue
					}
				}

				// Write SQL
				eventId := uuid.New().String()
				tx, err := DB.Begin()
				if err != nil {
					// ! SQL transaction Error
					w.Write([]byte(err.Error()))
					continue
				}
				_, err = tx.Exec(`
				INSERT INTO event 
					(id, board_id, element_id, username, operation, timestamp)
					VALUES (?, ?, ?, ?, ?, ?)`,
					eventId, boardId, create.Id, event.Name, "create", time.Now().Unix())
				if err != nil {
					tx.Rollback()
					// ! SQL Insert Event Error
					w.Write([]byte(err.Error()))
					continue
				}
				_, err = tx.Exec(`
				INSERT INTO elements 
					(id, board_id, type, bold, color, opacity, property, deleted)
					VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
					create.Id, boardId, create.Type, create.Bold, create.Color, create.Opacity, create.Property)
				if err != nil {
					tx.Rollback()
					// ! SQL Insert Element Error
					w.Write([]byte(err.Error()))
					continue
				}

				err = tx.Commit()
				if err != nil {
					// ! SQL Commit Error
					w.Write([]byte(err.Error()))
					continue
				}
			}
		case "delete": // MARK: >Delete
			{
				var delete PacketEventDelete
				err = dataDecoder.Decode(&delete)
				if err != nil {
					// ! Invalid Event Property
					w.Write([]byte(err.Error()))
					continue
				}
				// Write SQL
				eventId := uuid.New().String()
				tx, err := DB.Begin()
				if err != nil {
					// ! SQL transaction Error
					w.Write([]byte(err.Error()))
					continue
				}
				_, err = tx.Exec(`
				INSERT INTO event 
					(id, board_id, element_id, username, operation, timestamp)
					VALUES (?, ?, ?, ?, ?, ?)`,
					eventId, boardId, delete.Target, event.Name, "delete", time.Now().Unix())
				if err != nil {
					tx.Rollback()
					// ! SQL Insert Event Error
					w.Write([]byte(err.Error()))
					continue
				}
				var result sql.Result
				result, err = tx.Exec(`
				UPDATE elements 
					SET deleted = 1
					WHERE id = ? AND board_id = ?`,
					delete.Target, boardId)
				if err != nil {
					tx.Rollback()
					// ! SQL Update Element Error
					w.Write([]byte(err.Error()))
					continue
				}

				var n int64
				n, err = result.RowsAffected()
				if err != nil || n != 1 {
					tx.Rollback()
					// ! SQL Target Missing Error
					w.Write([]byte(err.Error()))
					continue
				}

				err = tx.Commit()
				if err != nil {
					// ! SQL Commit Error
					w.Write([]byte(err.Error()))
					continue
				}
			}
		default:
			{
				// ! Invalid Event Type
				continue
			}
		}

		// Packet Transfer
		Rooms[room].RLock()
		for _, v := range Rooms[room].Conn {
			websocket.Message.Send(v, packet)
		}
		Rooms[room].RUnlock()
	}
}
