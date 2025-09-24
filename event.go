package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (p PacketEvent) Error(msg string) PacketEvent {
	return PacketEvent{
		PacketId:  "notify",
		Name:      "server",
		Operation: "error",
	}.Set(
		PacketEventError{
			PacketId: p.PacketId,
			Message:  msg,
		})
}

// MARK: Create
func (p PacketEvent) CreateEvent(d *json.Decoder, eventId, boardId string) (fr FunctionResult) {
	var create PacketEventCreate
	err := d.Decode(&create)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Invalid PacketEventCreate",
				server: "Invalid PacketEventCreate",
			},
		}
	}

	// MARK: > Property check
	propertyDecoder := NewDecoder(bytes.NewReader(create.Property))
	switch create.ElementType {
	case "pen":
		err = propertyDecoder.Decode(&PropertyPen{})
		if err != nil {
			return FunctionResult{
				err: err,
				msg: FunctionMessage{
					client: "Invalid PacketEventCreate<pen>.property",
					server: "Invalid PacketEventCreate<pen>.property",
				},
			}
		}
	case "line":
		err = propertyDecoder.Decode(&PropertyLine{})
		if err != nil {
			return FunctionResult{
				err: err,
				msg: FunctionMessage{
					client: "Invalid PacketEventCreate<line>.property",
					server: "Invalid PacketEventCreate<line>.property",
				},
			}
		}
	case "stamp":
		err = propertyDecoder.Decode(&PropertyStamp{})
		if err != nil {
			return FunctionResult{
				err: err,
				msg: FunctionMessage{
					client: "Invalid PacketEventCreate<stamp>.property",
					server: "Invalid PacketEventCreate<stmap>.property",
				},
			}
		}
	default:
		return FunctionResult{
			err: fmt.Errorf("missing type: %s", create.ElementType),
			msg: FunctionMessage{
				client: "Invalid PacketEventCreate<unknown>.property",
				server: "Invalid PacketEventCreate<unknown>.property",
			},
		}
	}

	// MARK: > Write DB
	tx, fr := NewTx()
	if !fr.Ok() {
		return fr
	}
	defer tx.transaction.Rollback()

	fr = tx.InsertEvent(TableEvents{
		eventId:   eventId,
		boardId:   boardId,
		elementId: create.ElementId,
		username:  p.Name,
		operation: "create",
		undo:      false,
	})
	if !fr.Ok() {
		return fr
	}

	fr = tx.InsertElement(TableElements{
		elementId:   create.ElementId,
		boardId:     boardId,
		elementType: create.ElementType,
		bold:        create.Bold,
		color:       create.Color,
		opacity:     create.Opacity,
		property:    string(create.Property),
		deleted:     false,
	})
	if !fr.Ok() {
		return fr
	}

	fr = tx.Commit()
	if !fr.Ok() {
		return fr
	}

	return
}

// MARK: Delete
func (p PacketEvent) DeleteEvent(d *json.Decoder, eventId, boardId string) (se FunctionResult) {
	var delete PacketEventDelete
	err := d.Decode(&delete)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Invalid PacketEventDelete",
				server: "Invalid PacketEventDelete",
			},
		}
	}

	// MARK: > Write DB
	tx, fr := NewTx()
	if !fr.Ok() {
		return fr
	}
	defer tx.transaction.Rollback()

	fr = tx.InsertEvent(TableEvents{
		eventId:   eventId,
		boardId:   boardId,
		elementId: delete.Target,
		username:  p.Name,
		operation: "delete",
		undo:      false,
	})
	if !fr.Ok() {
		return fr
	}

	result, fr := tx.SelectElement(boardId, delete.Target)
	if !fr.Ok() {
		return fr
	}

	if result.deleted {
		return FunctionResult{
			err: fmt.Errorf("element deleted flag has true"),
			msg: FunctionMessage{
				client: "Invalid target element",
				server: "Invalid target element",
			},
		}
	}

	fr = tx.UpdateElementDeleted(boardId, delete.Target, true)
	if !fr.Ok() {
		return fr
	}

	fr = tx.Commit()
	if !fr.Ok() {
		return fr
	}

	return
}

// MARK: Undo
func (p PacketEvent) UndoEvent(room, boardId string) (se FunctionResult) {
	// MARK: > Write DB
	tx, fr := NewTx()
	if !fr.Ok() {
		return fr
	}
	defer tx.transaction.Rollback()

	qr := tx.transaction.QueryRow(`
	SELECT event_id, element_id, operation
		FROM events 
		WHERE board_id = ? AND undo = 0 AND disable = 0
		ORDER BY id DESC 
		LIMIT 1`,
		boardId)

	var eventId, elementId, operation string
	err := qr.Scan(&eventId, &elementId, &operation)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Nothing undo event",
				server: "Nothing undo event",
			},
		}
	}

	fr = tx.UpdateEventUndo(eventId, true)
	if !fr.Ok() {
		return fr
	}

	switch operation {
	case "create": // MARK: >> create
		fr = tx.UpdateElementDeleted(boardId, elementId, true)
		if !fr.Ok() {
			return fr
		}

		fr = tx.Commit()
		if !fr.Ok() {
			return fr
		}

		TransferAll(room,
			PacketEvent{
				PacketId:  "undo",
				Name:      "server",
				Operation: "delete",
			}.Set(
				PacketEventDelete{
					Target: elementId,
				},
			),
		)
	case "delete": // MARK: >> delete
		fr = tx.UpdateElementDeleted(boardId, elementId, false)
		if !fr.Ok() {
			return fr
		}

		result, fr := tx.SelectElement(boardId, elementId)
		if !fr.Ok() {
			return fr
		}

		data := PacketEventCreate{
			ElementId:   result.elementId,
			ElementType: result.elementType,
			Bold:        result.bold,
			Color:       result.color,
			Opacity:     result.opacity,
			Property:    json.RawMessage(result.property),
		}

		fr = tx.Commit()
		if !fr.Ok() {
			return fr
		}

		TransferAll(room, PacketEvent{
			PacketId:  "undo",
			Name:      "server",
			Operation: "create",
		}.Set(data),
		)
	}

	return
}

// MARK: Redo
func (p PacketEvent) RedoEvent(room, boardId string) (se FunctionResult) {
	// MARK: > Write DB
	tx, fr := NewTx()
	if !fr.Ok() {
		return fr
	}
	defer tx.transaction.Rollback()

	qr := tx.transaction.QueryRow(`
	SELECT event_id, element_id, operation
		FROM events 
		WHERE board_id = ? AND undo = 1 AND disable = 0
		ORDER BY id ASC 
		LIMIT 1`,
		boardId)

	var eventId, elementId, operation string
	err := qr.Scan(&eventId, &elementId, &operation)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Nothing redo event",
				server: "Nothing redo event",
			},
		}
	}

	fr = tx.UpdateEventUndo(eventId, false)
	if !fr.Ok() {
		return fr
	}

	switch operation {
	case "create": // MARK: >> create
		fr = tx.UpdateElementDeleted(boardId, elementId, false)
		if !fr.Ok() {
			return fr
		}

		result, fr := tx.SelectElement(boardId, elementId)
		if !fr.Ok() {
			return fr
		}

		data := PacketEventCreate{
			ElementId:   result.elementId,
			ElementType: result.elementType,
			Bold:        result.bold,
			Color:       result.color,
			Opacity:     result.opacity,
			Property:    json.RawMessage(result.property),
		}

		fr = tx.Commit()
		if !fr.Ok() {
			return fr
		}

		TransferAll(room, PacketEvent{
			PacketId:  "undo",
			Name:      "server",
			Operation: "create",
		}.Set(data),
		)

	case "delete": // MARK: >> delete
		fr = tx.UpdateElementDeleted(boardId, elementId, true)
		if !fr.Ok() {
			return fr
		}

		fr = tx.Commit()
		if !fr.Ok() {
			return fr
		}

		TransferAll(room,
			PacketEvent{
				PacketId:  "undo",
				Name:      "server",
				Operation: "delete",
			}.Set(
				PacketEventDelete{
					Target: elementId,
				},
			),
		)
	}

	return
}

func (p PacketEvent) ClearEvent(boardId string) (se FunctionResult) {
	// MARK: > Write DB
	tx, fr := NewTx()
	if !fr.Ok() {
		return fr
	}
	defer tx.transaction.Rollback()

	result, err := tx.transaction.Exec(`
		DELETE
			FROM events
			WHERE board_id = ?`,
		boardId)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed reset board",
				server: "Failed SQL \"delete events\"",
			},
		}
	}

	var n int64
	n, err = result.RowsAffected()
	if err != nil || n < 1 {
		return FunctionResult{
			err: fmt.Errorf("%s, rows:%d", err, n),
			msg: FunctionMessage{
				client: "Failed reset board",
				server: "Failed SQL \"delete events\" not match the expected count more than 1row",
			},
		}
	}

	_, err = tx.transaction.Exec(`
		DELETE
			FROM elements
			WHERE board_id = ?`,
		boardId)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed reset board",
				server: "Failed SQL \"delete elements\"",
			},
		}
	}

	fr = tx.Commit()
	if !fr.Ok() {
		return fr
	}

	return
}
