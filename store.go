package stackfar

import (
	"database/sql"
	"encoding/json"
	"path/filepath"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/ryanjyoder/sofp"
)

type StreamStore struct {
	db *sql.DB
}

type Delta interface {
	GetStreamID() string
	GetID() string
}

var (
	dbSetupQueries = []string{`CREATE TABLE IF NOT EXISTS deltas (
		ordering INTEGER PRIMARY KEY AUTOINCREMENT,
		id text UNIQUE,
		streamID text,
		msg TEXT
	)`, `CREATE INDEX IF NOT EXISTS
		streamIDIndex ON
		deltas(streamID)
	`, `CREATE TABLE IF NOT EXISTS progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		finished t TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		length INTEGER)`}
)

func NewStores(storageDir string, sites []string) (map[string]*StreamStore, error) {
	stores := map[string]*StreamStore{}
	for i := range sites {
		filename := filepath.Join(storageDir, sites[i]+".sqlite")
		store, err := NewStreamStore(filename)
		if err != nil {
			return stores, err
		}
		stores[sites[i]] = store
	}
	return stores, nil
}

func NewStreamStore(dbFilepath string) (*StreamStore, error) {
	database, err := sql.Open("sqlite3", dbFilepath)
	if err != nil {
		return nil, err
	}

	for _, q := range dbSetupQueries {
		statement, err := database.Prepare(q)
		if err != nil {
			return nil, err
		}
		_, err = statement.Exec()
		if err != nil {
			return nil, err
		}
	}
	return &StreamStore{
		db: database,
	}, nil
}

func (w *StreamStore) Write(d Delta) error {
	text, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = w.db.Exec(`
	INSERT INTO deltas
		(id, streamID, msg)
	VALUES 
		(?, ?, ?)
	`, d.GetID(), d.GetStreamID(), string(text))

	if err == nil {
		return nil

	}
	if sqlErr, ok := err.(sqlite3.Error); ok {
		if sqlErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			return nil
		}
	}
	return err

}

func (w *StreamStore) GetStreamDeltas(domain string, id string) ([]*sofp.Row, error) {
	rows, err := w.db.Query(`
	SELECT msg FROM deltas WHERE streamID = ? order by ordering
	`, domain+"/"+id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deltas := []*sofp.Row{}
	for rows.Next() {
		msg := ""
		delta := &sofp.Row{}
		rows.Scan(&msg)
		err := json.Unmarshal([]byte(msg), delta)
		if err != nil {
			return nil, err
		}
		deltas = append(deltas, delta)
	}

	return deltas, nil
}

func (w *StreamStore) SetProgress(length int64) error {
	_, err := w.db.Exec(`
	INSERT INTO progress
		(length)
	VALUES 
		(?)
	`, length)
	return err

}

func (w *StreamStore) GetProgress() (int64, error) {
	rows, err := w.db.Query(`
	SELECT length FROM progress order by id desc limit 1
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var length int64
	for rows.Next() {
		err := rows.Scan(&length)
		if err != nil {
			return 0, err
		}
		return length, nil
	}

	return 0, nil
}
