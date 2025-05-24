package queue

import (
	"server/backend"
	"sync"
	"time"
)

// RequestQueue manages a queue of HTTP requests for a specific model.
type RequestQueue struct {
	modelName    string
	queue        chan *Request
	backend      *backend.Client
	closed       chan struct{}
	activeCount  int
	mutex        sync.Mutex
	needsLog     bool
	lastLogTime  time.Time
	logRateLimit time.Duration
}

// NewRequestQueue initializes a new RequestQueue with monitoring capabilities.
func NewRequestQueue(modelName string, backend *backend.Client) *RequestQueue {
	rq := &RequestQueue{
		modelName:    modelName,
		queue:        make(chan *Request, 10000), // Adjust buffer size as needed
		backend:      backend,
		closed:       make(chan struct{}),
		activeCount:  0,
		needsLog:     false,
		lastLogTime:  time.Time{},
		logRateLimit: 1 * time.Second,
	}

	go rq.monitor()
	return rq
}

// Enqueue adds a new request to the queue and notifies the monitor.
func (rq *RequestQueue) Enqueue(req *Request) {
	rq.queue <- req
	rq.notifyChange()
}

// monitor listens for changes and logs metrics accordingly.
func (rq *RequestQueue) monitor() {
	ticker := time.NewTicker(500 * time.Millisecond) // Check twice every second
	defer ticker.Stop()

	for {
		select {
		case <-rq.closed:
			return
		case <-ticker.C:
			if rq.needsLog {
				rq.logMetrics()
			}
		}
	}
}

// notifyChange flags that a change has occurred and logging is needed.
func (rq *RequestQueue) notifyChange() {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	rq.needsLog = true
}

// incrementActive safely increments the activeCount and notifies a change.
func (rq *RequestQueue) incrementActive() {
	rq.mutex.Lock()
	rq.activeCount++
	rq.needsLog = true
	rq.mutex.Unlock()
}

// decrementActive safely decrements the activeCount and notifies a change.
func (rq *RequestQueue) decrementActive() {
	rq.mutex.Lock()
	if rq.activeCount > 0 {
		rq.activeCount--
	}
	rq.needsLog = true
	rq.mutex.Unlock()
}

// logMetrics logs the current queue size and active processing count if rate limits allow.
func (rq *RequestQueue) logMetrics() {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	now := time.Now()
	if now.Sub(rq.lastLogTime) >= rq.logRateLimit {
		queueSize := len(rq.queue)
		log.Printf("Model: %s | Queue Size: %d | Processing: %d", rq.modelName, queueSize, rq.activeCount)
		rq.lastLogTime = now
		rq.needsLog = false
	}
}

// Shutdown gracefully shuts down the RequestQueue.
func (rq *RequestQueue) Shutdown() {
	close(rq.closed)
	close(rq.queue)
}
