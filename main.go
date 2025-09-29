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

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/exp/slog"
	"golang.org/x/net/websocket"
)

var (
	Listen    = ":1025"
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
			room := r.URL.Query().Get("id")
			if room != "" {
				http.SetCookie(w, &http.Cookie{
					Name:  "room",
					Value: room,
					Path:  "/",
				})
			}
			user := r.URL.Query().Get("user")
			if user != "" {
				http.SetCookie(w, &http.Cookie{
					Name:  "user",
					Value: user,
					Path:  "/",
				})
			}

			if room == "" || user == "" || UserCheck(room, user) {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
		}

		// Compute
		h.ServeHTTP(w, r)
	})
}

func UserCheck(room, user string) bool {
	RoomsLock.RLock()
	r, ok := Rooms[room]
	RoomsLock.RUnlock()
	if !ok {
		return false
	}

	r.RLock()
	_, ok = r.Conn[user]
	r.RUnlock()
	return ok
}

// MARK: Websocket
func WebsocketRequest(w *websocket.Conn) {
	var source = w.Request().RemoteAddr

	room := w.Request().URL.Query().Get("id")
	if room == "" {
		w.Close()
		return
	}
	user := w.Request().URL.Query().Get("user")
	if user == "" {
		w.Close()
		return
	}

	// Board check
	boardId, err := GetBoardId(room)
	if err != nil {
		logger.Error("Failed board get/create", "IP", source, "Room", room, "ID", user, "message", err)
		w.Close()
		return
	}

	// Save session
	r, ok := NewRoom(room, user, w)
	if !ok {
		logger.Info("Duplicate access", "IP", source, "ID", user)
		w.Close()
		return
	}

	defer func() {
		r.Lock()
		defer r.Unlock()
		delete(r.Conn, user)

		if len(r.Conn) == 0 {
			RoomsLock.Lock()
			defer RoomsLock.Unlock()
			delete(Rooms, room)
		}

		w.Close()
	}()

	var packet string
	logger.Info("New Websocket connection", "IP", source, "Room", room, "ID", user)
	// MARK: > Read loop
	for {
		err := websocket.Message.Receive(w, &packet)
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Error("new message", "IP", source, "Room", room, "ID", user, "packet", packet, "message", err)
			return
		}

		eventId := uuid.New().String()

		// MARK: >> Validation
		var event PacketEvent
		err = json.Unmarshal([]byte(packet), &event)
		if err != nil {
			// ! Invalid Event
			websocket.JSON.Send(w, event.Error("PacketEvent marshal error."))
			logger.Error("PacketEvent marshal error", "Room", room, "ID", user, "packet", packet, "message", err)
			continue
		}

		// * Check user
		if event.User != user {
			// ! Invalid Event
			websocket.JSON.Send(w, event.Error("User has unmatching."))
			continue
		}

		dataDecoder := NewDecoder(bytes.NewReader(event.Data))
		switch event.Operation {
		case "heatbeat": // MARK: >>> Heatbeat
			websocket.JSON.Send(w,
				PacketEvent{
					PacketId:  "notify",
					User:      "server",
					Operation: "success",
				}.Set(
					PacketEventSuccess{
						EventId:  eventId,
						PacketId: event.PacketId,
					}))
			continue
		case "history": // MARK: >>> History
			result := event.HistoryEvent(boardId, w)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}

		case "mouse": // MARK: >>> Mouse
			err = dataDecoder.Decode(&PacketEventMouse{})
			if err != nil {
				// ! Invalid Event Property
				websocket.JSON.Send(w, event.Error("PacketEventMouse marshal error."))
				logger.Error("PacketEventMouse marshal error", "Room", room, "ID", user, "packet", packet, "message", event)
				continue
			}

		case "create": // MARK: >>> Create
			c, result := event.CreateEvent(dataDecoder, eventId, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}
			TransferAll(room, PacketEvent{
				PacketId:  event.PacketId,
				User:      event.User,
				Operation: "create",
			}.
				Set(c),
			)

		case "delete": // MARK: >>> Delete
			result := event.DeleteEvent(dataDecoder, eventId, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}

		case "undo": // MARK: >>> Undo
			result := event.UndoEvent(room, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}

		case "redo": // MARK: >>> Redo
			result := event.RedoEvent(room, boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}

		case "clear": // MARK: >>> Clear
			result := event.ClearEvent(boardId)
			if !result.Ok() {
				websocket.JSON.Send(w, event.Error(result.msg.client))
				logger.Error(result.msg.server, "Room", room, "ID", user, "packet", packet, "message", result.err)
				continue
			}

		default: // MARK: >>> default
			// ! Invalid Event.Operation Type
			websocket.JSON.Send(w,
				PacketEvent{
					PacketId:  "notify",
					User:      "server",
					Operation: "error",
				}.Set(
					PacketEventError{
						PacketId: event.PacketId,
						Message:  "Unknown packet error",
					}))
			logger.Warn("PacketEvent.Operation unknown", "Room", room, "ID", user, "packet", packet, "message", event)
			continue
		}

		// Send success
		if !(event.Operation == "heatbeat" || event.Operation == "mouse") {
			logger.Debug("success event", "IP", source, "Room", room, "ID", user, "packet", packet)
		}
		websocket.JSON.Send(w,
			PacketEvent{
				PacketId:  "notify",
				User:      "server",
				Operation: "success",
			}.Set(
				PacketEventSuccess{
					EventId:  eventId,
					PacketId: event.PacketId,
				}))

		// MARK: >> Packet transfer
		// Packet Transfer
		if !(event.Operation == "histroy" || event.Operation == "create") {
			Rooms[room].RLock()
			for cId, v := range Rooms[room].Conn {
				if cId == user {
					continue
				}
				websocket.Message.Send(v, packet)
			}
			Rooms[room].RUnlock()
		}
	}
}

func TransferAll(r string, p PacketEvent) {
	RoomsLock.RLock()
	room, ok := Rooms[r]
	RoomsLock.RUnlock()
	if !ok {
		return
	}

	room.RLock()
	for _, v := range room.Conn {
		websocket.JSON.Send(v, p)
	}
	room.RUnlock()
}

func NewRoom(room, user string, w *websocket.Conn) (cr *Room, ok bool) {
	RoomsLock.Lock()
	r, ok := Rooms[room]
	if !ok {
		r = &Room{
			Conn: map[string]*websocket.Conn{},
		}
		Rooms[room] = r
	}
	RoomsLock.Unlock()

	if _, ok := r.Conn[user]; ok {
		return r, false
	}

	r.Lock()
	r.Conn[user] = w
	r.Unlock()

	return r, true
}
