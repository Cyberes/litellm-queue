package queue

import (
	"server/backend"
	"sync"
	"time"
)

// RequestQueue manages the queue of HTTP requests
type RequestQueue struct {
	requests        []*Request
	mutex           sync.Mutex
	cond            *sync.Cond
	backend         *backend.Client
	shutdownCh      chan struct{}
	lastPrintedSize int
	lastPrintTime   time.Time
}

// NewRequestQueue initializes a new RequestQueue and starts the processing worker
func NewRequestQueue(backendUrl string) *RequestQueue {
	q := &RequestQueue{
		requests:   make([]*Request, 0),
		backend:    backend.NewBackendClient(backendUrl), // Replace with your backend URL
		shutdownCh: make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mutex)
	go q.process()
	go q.printQueueSize()
	return q
}

// Enqueue adds a new request to the queue
func (q *RequestQueue) Enqueue(req *Request) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	req.EnqueuedAt = time.Now()
	q.requests = append(q.requests, req)
	q.cond.Signal()
}

// Remove removes a specific request from the queue if the client disconnects
func (q *RequestQueue) Remove(req *Request) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for i, r := range q.requests {
		if r == req {
			q.requests = append(q.requests[:i], q.requests[i+1:]...)
			log.Printf("Request removed from queue: %s %s", r.Method, r.URL)
			break
		}
	}
}

// Shutdown gracefully shuts down the request queue processing
func (q *RequestQueue) Shutdown() {
	close(q.shutdownCh)
	q.cond.Broadcast()
}

func (q *RequestQueue) printQueueSize() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			q.mutex.Lock()
			currentSize := len(q.requests)
			if currentSize != q.lastPrintedSize {
				log.Infof("Queue size: %d", currentSize)
				q.lastPrintedSize = currentSize
				q.lastPrintTime = time.Now()
			}
			q.mutex.Unlock()
		case <-q.shutdownCh:
			return
		}
	}
}
