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
	rows, err := store.ListStreamIDs()
	if err != nil {
		return err
	}

	n := 0
	for rows.Next() {
		id := ""
		err := rows.Scan(&id)
		if err != nil {
			return err
		}
		if n%1000 == 0 {
			fmt.Println("indesing doc #:", n, id)
		}
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
	tx, err := writer.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	row, err := reader.Next()
	for row != nil && err == nil {
		new, err := WriteDeltaToDB(row, tx)
		if err == nil && new {
			changes++
		}
		row, err = reader.Next()
	}
	tx.Commit()

	return changes, writer.SetProgress(domain, fileinfo.Size())

}
