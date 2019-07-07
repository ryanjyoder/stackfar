package stackfar

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/ryanjyoder/sofp"
)

type StreamStore struct {
	db    *sql.DB
	index bleve.Index
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
		task TEXT,
		finished t TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		length INTEGER)`}
)

func NewStreamStore(storeDir string) (*StreamStore, error) {
	err := os.MkdirAll(storeDir, 0755)
	if err != nil {
		return nil, err
	}
	dbFilepath := filepath.Join(storeDir, "streams.sqlite")
	indexDir := filepath.Join(storeDir, "index")
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
	index, err := createIndex(indexDir)
	if err != nil {
		return nil, err
	}

	return &StreamStore{
		db:    database,
		index: index,
	}, nil
}

func createIndex(indexDir string) (bleve.Index, error) {
	index, err := bleve.Open(indexDir)
	if err != nil {
		mapping := defaultMapping()
		index, err = bleve.New(indexDir, mapping)
		if err != nil {
			return nil, err
		}
	}
	return index, err
}

func defaultMapping() *mapping.IndexMappingImpl {
	questionMapping := bleve.NewDocumentMapping()
	noStoreMapping := bleve.NewTextFieldMapping()
	noStoreMapping.Store = false
	questionMapping.AddFieldMappingsAt("_default", noStoreMapping)

	storeMapping := bleve.NewTextFieldMapping()
	storeMapping.Store = true
	questionMapping.AddFieldMappingsAt("Body", storeMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("_default", questionMapping)
	return indexMapping
}

// Write returns if the if the delta is new
func WriteDeltaToDB(d Delta, tx *sql.Tx) (bool, error) {
	text, err := json.Marshal(d)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(`
	INSERT INTO deltas
		(id, streamID, msg)
	VALUES 
		(?, ?, ?)
	`, d.GetID(), d.GetStreamID(), string(text))

	if err == nil {
		return true, nil // new delta inserted

	}
	if sqlErr, ok := err.(sqlite3.Error); ok {
		if sqlErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			return false, nil // not new but no error
		}
	}
	return false, err

}

func (w *StreamStore) GetStreamDeltas(id string) ([]*sofp.Row, error) {
	rows, err := w.db.Query(`
	SELECT msg FROM deltas WHERE streamID = ? order by ordering
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deltas := []*sofp.Row{}
	for rows.Next() {
		msg := ""
		delta := &sofp.Row{}
		err := rows.Scan(&msg)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(msg), delta)
		if err != nil {
			return nil, err
		}
		deltas = append(deltas, delta)
	}

	return deltas, nil
}

func (w *StreamStore) ListStreamIDs() (*sql.Rows, error) {
	return w.db.Query(`
	SELECT streamID FROM deltas group by streamID
	`)
}

func (w *StreamStore) SetProgress(task string, checkpoint int64) error {
	_, err := w.db.Exec(`
	INSERT INTO progress
		(task, length)
	VALUES 
		(?, ?)
	`, task, checkpoint)
	return err

}

func (w *StreamStore) GetProgress(task string) (int64, error) {
	rows, err := w.db.Query(`
	SELECT length FROM progress WHERE task = ? order by id desc limit 1
	`, task)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var checkpoint int64
	for rows.Next() {
		err := rows.Scan(&checkpoint)
		if err != nil {
			return 0, err
		}
		return checkpoint, nil
	}

	return 0, nil
}

func (w *StreamStore) LastDelta() (int64, error) {
	rows, err := w.db.Query(`
	SELECT max(ordering) FROM deltas
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

func (w *StreamStore) Begin() (*sql.Tx, error) {
	return w.db.Begin()
}

func (s *StreamStore) Index(id string, doc interface{}) error {
	return s.index.Index(id, doc)
}

func (s *StreamStore) Search(queryStr string) (*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(queryStr)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"Title", "Body"}
	searchResult, err := s.index.Search(searchRequest)
	return searchResult, err
}
