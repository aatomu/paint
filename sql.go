package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MARK: Generic
func CreateTables(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS boards (
		board_id               TEXT    PRIMARY KEY,
		name             TEXT    NOT NULL,
		created_at INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("%s(in boards)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS elements (
		element_id        TEXT    NOT NULL,
		board_id          TEXT    NOT NULL,
		element_type      TEXT    NOT NULL,
		bold              REAL    NOT NULL,
		color             TEXT    NOT NULL,
		opacity           REAL    NOT NULL,
		property          TEXT    NOT NULL,
		deleted           INTEGER NOT NULL,
		PRIMARY KEY (element_id,board_id),
		FOREIGN KEY(board_id) REFERENCES boards(board_id)
	)`)
	if err != nil {
		return fmt.Errorf("%s(in elements)", err.Error())
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS events (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id   TEXT    NOT NULL UNIQUE,
		board_id   TEXT    NOT NULL,
		element_id TEXT,
		username   TEXT    NOT NULL,
		operation  TEXT    NOT NULL,
		undo       INTEGER NOT NULL,
		created_at  INTEGER NOT NULL,
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
		_, err = DB.Exec("INSERT INTO boards (board_id, name, created_at) VALUES (?, ?, ?)", boardId, name, now)
		return
	}
	return
}

func NewTx() (t *Transaction, fr FunctionResult) {
	var err error
	tx, err := DB.Begin()
	if err != nil {
		return nil, FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "internal server error",
				server: "Failed transaction start",
			},
		}
	}

	t = &Transaction{
		transaction: tx,
	}

	return
}

func (tx *Transaction) Commit() (fr FunctionResult) {
	err := tx.transaction.Commit()
	if err != nil {
		tx.transaction.Rollback()
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save packet",
				server: "Failed transaction commit",
			},
		}
	}

	return
}

func (tx *Transaction) InsertEvent(e TableEvents) (fr FunctionResult) {
	undoValue := 0
	if e.undo {
		undoValue = 1
	}

	_, err := tx.transaction.Exec(`
		INSERT INTO events 
			(event_id, board_id, element_id, username, operation, undo, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.eventId, e.boardId, e.elementId, e.username, e.operation, undoValue, time.Now().UnixMilli())
	if err != nil {
		tx.transaction.Rollback()
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save event",
				server: "Failed SQL \"insert events\"",
			},
		}
	}

	_, err = tx.transaction.Exec(`
		UPDATE events 
			SET disable = 1
			WHERE board_id = ? AND undo = 1`,
		e.boardId)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save event",
				server: "Failed SQL \"update events.disable\"",
			},
		}
	}

	return
}

// MARK: > UpdateEventUndo
func (tx *Transaction) UpdateEventUndo(eventId string, flag bool) (fr FunctionResult) {
	undoValue := 0
	if flag {
		undoValue = 1
	}

	result, err := tx.transaction.Exec(`
	UPDATE events 
		SET undo = ?
		WHERE event_id = ?`,
		undoValue,
		eventId)
	if err != nil {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed read event",
				server: "Failed SQL \"update events.undo\"",
			},
		}
	}

	var n int64
	n, err = result.RowsAffected()
	if err != nil || n != 1 {
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed read event",
				server: "Failed SQL \"update events.undo\" not match the expected count of 1row",
			},
		}
	}

	return
}

func (tx *Transaction) InsertElement(e TableElements) (fr FunctionResult) {
	deletedValue := 0
	if e.deleted {
		deletedValue = 1
	}

	_, err := tx.transaction.Exec(`
		INSERT INTO elements 
			(element_id, board_id, element_type, bold, color, opacity, property, deleted)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.elementId, e.boardId, e.elementType, e.bold, e.color, e.opacity, e.property, deletedValue)
	if err != nil {
		tx.transaction.Rollback()
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save element",
				server: "Failed SQL \"insert elements\"",
			},
		}
	}

	return
}

// MARK: > SelectElement
func (tx *Transaction) SelectElement(boardId, elementId string) (e TableElements, fr FunctionResult) {
	qr := tx.transaction.QueryRow(`
	SELECT element_id, board_id, element_type, bold, color, opacity, property, deleted
		FROM elements 
		WHERE board_id = ? AND element_id = ?`,
		boardId, elementId)

	var deletedValue int
	err := qr.Scan(&e.elementId, &e.boardId, &e.elementType, &e.bold, &e.color, &e.opacity, &e.property, &deletedValue)
	if err != nil {
		return e, FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Nothing element",
				server: "Nothing element",
			},
		}
	}
	e.deleted = deletedValue == 1

	return
}

func (tx *Transaction) UpdateElementDeleted(boardId, elementId string, flag bool) (fr FunctionResult) {
	flagValue := 0
	if flag {
		flagValue = 1
	}
	result, err := tx.transaction.Exec(`
		UPDATE elements 
			SET deleted = ?
			WHERE element_id = ? AND board_id = ?`,
		flagValue,
		elementId, boardId)
	if err != nil {
		tx.transaction.Rollback()
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save event",
				server: "Failed SQL \"update elements.deleted\"",
			},
		}
	}

	var n int64
	n, err = result.RowsAffected()
	if err != nil || n != 1 {
		tx.transaction.Rollback()
		return FunctionResult{
			err: err,
			msg: FunctionMessage{
				client: "Failed save event",
				server: "Failed SQL \"update elements.deleted\" not match the expected count of 1row",
			},
		}
	}

	return
}
