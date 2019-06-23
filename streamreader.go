package stackfar

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/ryanjyoder/sofp"
)

type StreamReader struct {
	rowChan chan *sofp.Row
	errChan chan error
}

func NewStreamReader(filename string) (*StreamReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	rowChan := make(chan *sofp.Row)
	errChan := make(chan error)

	go func() {
		defer file.Close()
		defer close(rowChan)
		defer close(errChan)
		for scanner.Scan() {
			rowStr := scanner.Text()
			row := sofp.Row{}
			err = json.Unmarshal([]byte(rowStr), &row)
			rowChan <- &row
			errChan <- err
		}

		if err := scanner.Err(); err != nil {
			rowChan <- nil
			errChan <- err
		}
	}()

	return &StreamReader{
		rowChan: rowChan,
		errChan: errChan,
	}, nil
}

func (r *StreamReader) Next() (*sofp.Row, error) {
	return <-r.rowChan, <-r.errChan
}
