package stackfar

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func LoadStores(store *StreamStore, streamsDir string, sites []string) error {

	changes := 0
	for _, domain := range sites {

		streamFile := filepath.Join(streamsDir, domain, "unified-stream")
		newChanges, err := loadStore(store, domain, streamFile)
		if err != nil {
			log.Println("Error loading stream:", domain, err)
		}
		changes += newChanges
	}

	err := indexStore(store)
	if err != nil {
		log.Println("error indexing:", err)
		return err
	}

	return nil
}

func indexStore(store *StreamStore) error {
	fmt.Println("begining indexing")
	checkpoint, err := store.GetProgress("index")
	if err != nil {
		return err
	}
	maxID, err := store.LastDelta()
	if err != nil {
		return err
	}
	fmt.Println("last checkpoint", checkpoint, "/", maxID)
	if maxID == checkpoint {
		log.Println("no indexing needed:", checkpoint, "/", maxID)
		// finished indexing already
		return nil
	}

	rows, err := store.ListStreamIDs()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		id := ""
		rows.Scan(&id)
		deltas, err := store.GetStreamDeltas(id)
		if err != nil {
			return err
		}
		question, err := DeltasToQuestion(deltas)
		if err != nil {
			return nil
		}
		store.Index(id, question)
	}
	store.SetProgress("index", maxID)
	fmt.Println("indexing finished")
	return nil
}

func loadStore(writer *StreamStore, domain string, streamFile string) (int, error) {

	fileinfo, err := os.Stat(streamFile)
	for err != nil {
		log.Println("waiting for stream to appear:", streamFile)
		time.Sleep(1 * time.Minute)
		fileinfo, err = os.Stat(streamFile)
	}
	if fileinfo.IsDir() {
		return 0, fmt.Errorf("StreamFile cannot be a directory")
	}
	length, err := writer.GetProgress(domain)
	if err != nil {
		return 0, err
	}
	fmt.Println("Current progress seperating the stream:", length, "total length:", fileinfo.Size())
	if fileinfo.Size() == length {
		// already done!
		return 0, nil
	}

	reader, err := NewStreamReader(streamFile)
	if err != nil {
		return 0, err
	}

	changes := 0
	row, err := reader.Next()
	for row != nil && err == nil {
		new, err := writer.Write(row)
		if err == nil && new {
			changes++
		}
		row, err = reader.Next()
	}

	return changes, writer.SetProgress(domain, fileinfo.Size())

}
