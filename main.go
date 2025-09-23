package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/exp/slog"
	"golang.org/x/net/websocket"
)

var (
	Listen    = ":1026"
	Rooms     = map[string]*Room{}
	RoomsLock = sync.RWMutex{}
	DB        *sql.DB
	logger    = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02T15:04:05.000MST"))
			}
			return a
		},
	}))
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
		logger.Error("Failed open database", slog.Group("message", err))
		return
	}
	defer DB.Close()

	err = CreateTables(DB)
	if err != nil {
		logger.Error("Failed create table", slog.Group("message", err))
		return
	}

	// Http handle
	assets := http.FileServer(fallbackFileSystem{http.Dir("./assets")})
	http.Handle("/", middleware(assets))
	http.Handle("/ws", middleware(websocket.Handler(WebsocketRequest)))

	// Boot Server
	logger.Info("Listen http server boot")
	err = http.ListenAndServe(Listen, nil)
	if err != nil {
		logger.Error("Failed listen http server", slog.Group("message", err))
		return
	}
}

// MARK: HTTP middle
func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("new request", "IP", r.RemoteAddr, "Method", r.Method, "URI", r.URL, "Header", r.Header)

		if strings.HasPrefix(r.URL.Path, "/room") {
			if _, err := r.Cookie("name"); err != nil {
				logger.Info("cookie(\"name\") is not found, 307redirect", "IP", r.RemoteAddr, "Transfer", "/")
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
		}

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
	connId := time.Now().UnixNano()

	// Board check
	boardId, err := GetBoardId(room)
	if err != nil {
		logger.Error("Failed board get/create", "message", err)
		w.Close()
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
	r.Conn[connId] = w
	r.Unlock()
	RoomsLock.Unlock()

	defer func() {
		r.Lock()
		delete(r.Conn, connId)
		r.Unlock()
		if len(r.Conn) == 0 {
			RoomsLock.Lock()
			delete(Rooms, room)
			RoomsLock.Unlock()
		}
		w.Close()
	}()

	var packet string
	var source = w.Request().RemoteAddr
	// MARK: > Read loop
	for {
		err := websocket.Message.Receive(w, &packet)
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Error("new message", "IP", source, "ID", connId, "packet", packet, "message", err)
			return
		}
		logger.Debug("new message", "IP", source, "ID", connId, "packet", packet)

		eventId := uuid.New().String()

		// MARK: >> Validation
		var event PacketEvent
		err = json.Unmarshal([]byte(packet), &event)
		if err != nil {
			// ! Invalid Event
			websocket.JSON.Send(w, event.Error("PacketEvent marshal error."))
			logger.Debug("PacketEvent marshal error", "ID", connId, "message", event)
			continue
		}

		dataDecoder := NewDecoder(bytes.NewReader(event.Data))
		switch event.Operation {
		case "mouse": // MARK: >>> Mouse
			{
				err = dataDecoder.Decode(&PacketEventMouse{})
				if err != nil {
					// ! Invalid Event Property
					websocket.JSON.Send(w, event.Error("PacketEventMouse marshal error."))
					logger.Debug("PacketEventMouse marshal error", "ID", connId, "message", event)
					continue
				}
			}
		case "create": // MARK: >>> Create
			result := event.CreateEvent(dataDecoder, eventId, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, result.msg.client)
				logger.Debug(result.msg.server, "ID", connId, "message", result.err)
				continue
			}
		case "delete": // MARK: >>> Delete
			result := event.DeleteEvent(dataDecoder, eventId, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, result.msg.client)
				logger.Debug(result.msg.server, "ID", connId, "message", result.err)
				continue
			}

		case "undo": // MARK: >>> Undo
			result := event.UndoEvent(room, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, result.msg.client)
				logger.Debug(result.msg.server, "ID", connId, "message", result.err)
				continue
			}

		case "redo": // MARK: >>> Redo
			result := event.RedoEvent(room, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Debug(result.msg.server, "ID", connId, "message", result.err)
				continue
			}

						TransferAll(room,
							PacketEvent{
								PacketId:  "redo",
								Name:      "server",
								Operation: "delete",
							}.Set(
								PacketEventDelete{
									Target: element_id,
								},
							),
						)
					}
				}
			}
		default: // MARK: >>> default
			{
				// ! Invalid Event.Operation Type
				websocket.JSON.Send(w,
					PacketEvent{
						PacketId:  "notify",
						Name:      "server",
						Operation: "error",
					}.Set(
						PacketEventError{
							PacketId: event.PacketId,
							Message:  "Unknown packet error",
						}))
				logger.Debug("PacketEvent.Operation unknown", "ID", connId, "message", event)
				continue
			}
		}

		// Send success
		websocket.JSON.Send(w,
			PacketEvent{
				PacketId:  "notify",
				Name:      "server",
				Operation: "success",
			}.Set(
				PacketEventSuccess{
					EventId:  eventId,
					PacketId: event.PacketId,
				}))

		// MARK: >> Packet transfer
		// Packet Transfer
		Rooms[room].RLock()
		for cId, v := range Rooms[room].Conn {
			if cId == connId {
				continue
			}
			websocket.Message.Send(v, packet)
		}
		Rooms[room].RUnlock()
	}
}

func TransferAll(r string, p PacketEvent) {
	Rooms[r].RLock()
	for _, v := range Rooms[r].Conn {
		websocket.JSON.Send(v, p)
	}
	Rooms[r].RUnlock()
}
