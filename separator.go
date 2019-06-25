package stackfar

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func LoadStores(store *StreamStore, streamsDir string, sites []string) error {

	storedID := NewDedupQueue(10000)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := indexStore(store, storedID)
		if err != nil {
			log.Println("error indexing:", err)
			return
		}
	}()

	changes := 0
	for _, domain := range sites {

		streamFile := filepath.Join(streamsDir, domain, "unified-stream")
		newChanges, err := loadStore(store, domain, streamFile, storedID)
		if err != nil {
			log.Println("Error loading stream:", domain, err)
		}
		changes += newChanges
	}
	storedID.Done()
	wg.Wait()

	return nil
}

func indexStore(store *StreamStore, storedID *DedupQueue) error {
	fmt.Println("begining indexing")

	id, ok := storedID.Dequeue()
	for ok {
		deltas, err := store.GetStreamDeltas(id)
		if err != nil {
			return err
		}
		question, err := DeltasToQuestion(deltas)
		if err != nil {
			return nil
		}
		store.Index(id, question)

		id, ok = storedID.Dequeue()
	}
	fmt.Println("indexing finished")
	return nil
}

func loadStore(writer *StreamStore, domain string, streamFile string, storedID *DedupQueue) (int, error) {

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
			storedID.Queue(row.StreamID)
		}
		row, err = reader.Next()
	}

	return changes, writer.SetProgress(domain, fileinfo.Size())

}
