package stackfar

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func LoadStores(stores map[string]*StreamStore, streamsDir string) error {

	for domain, writer := range stores {

		streamFile := filepath.Join(streamsDir, domain, "unified-stream")
		err := loadStore(writer, streamFile)
		if err != nil {
			log.Println("Error loading stream:", domain, err)
		}
	}

	return nil
}

func loadStore(writer *StreamStore, streamFile string) error {

	fileinfo, err := os.Stat(streamFile)
	for err != nil {
		log.Println("waiting for stream to appear:", streamFile)
		time.Sleep(1 * time.Minute)
		fileinfo, err = os.Stat(streamFile)
	}
	if fileinfo.IsDir() {
		return fmt.Errorf("StreamFile cannot be a directory")
	}
	length, err := writer.GetProgress()
	if err != nil {
		return err
	}
	fmt.Println("Current progress seperating the stream:", length, "total length:", fileinfo.Size())
	if fileinfo.Size() == length {
		// already done!
		return nil
	}

	reader, err := NewStreamReader(streamFile)
	if err != nil {
		return err
	}

	row, err := reader.Next()
	for row != nil && err == nil {
		writer.Write(row)
		row, err = reader.Next()
	}

	return writer.SetProgress(fileinfo.Size())

}
