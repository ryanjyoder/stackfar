package stackfar

type DedupQueue struct {
	pipeline chan string
	inPipe   map[string]struct{}
	done     bool
}

func NewDedupQueue(size int) *DedupQueue {
	return &DedupQueue{
		pipeline: make(chan string, size),
		inPipe:   map[string]struct{}{},
		done:     false,
	}
}

func (q *DedupQueue) Queue(s string) {
	if q.done {
		return
	}
	if _, ok := q.inPipe[s]; ok {
		return
	}
	q.inPipe[s] = struct{}{}
	q.pipeline <- s
}

func (q *DedupQueue) Dequeue() (string, bool) {
	if len(q.inPipe) < 1 && q.done {
		return "", false
	}
	s := <-q.pipeline
	delete(q.inPipe, s)
	return s, true
}

func (q *DedupQueue) IsEmpty() bool {
	return len(q.inPipe) == 0
}

func (q *DedupQueue) Done() {
	q.done = true
	close(q.pipeline)
}

func (q *DedupQueue) IsDone() bool {
	return q.done
}
