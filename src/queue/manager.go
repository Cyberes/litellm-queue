package queue

import (
	"server/backend"
	"server/config"
	"sync"
)

// QueueManager manages multiple RequestQueues, each for a specific model.
type QueueManager struct {
	queues         map[string]*RequestQueue
	defaultQueue   *RequestQueue
	backend        *backend.Client
	mutex          sync.Mutex
	defaultWorkers int
}

// NewQueueManager initializes a new QueueManager with model-specific queues.
func NewQueueManager(modelConfig []config.ModelConfigEntry, backendURL string) *QueueManager {
	qm := &QueueManager{
		queues:         make(map[string]*RequestQueue),
		backend:        backend.NewBackendClient(backendURL),
		defaultWorkers: 100, // Default concurrency limit for unspecified models
	}

	// Initialize queues for each configured model.
	for _, cfg := range modelConfig {
		model := cfg.Name
		log.Infof("Creating queue for model: %s", model)
		rq := NewRequestQueue(model, qm.backend)
		for i := 0; i < cfg.Size; i++ {
			go rq.process()
		}
		qm.queues[model] = rq
	}

	// Initialize a default queue for models not specified in the config.
	qm.defaultQueue = NewRequestQueue("default", qm.backend)
	for i := 0; i < qm.defaultWorkers; i++ {
		go qm.defaultQueue.process()
	}

	return qm
}

// Enqueue adds a request to the appropriate queue based on the model.
func (qm *QueueManager) Enqueue(model string, req *Request) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	if queue, exists := qm.queues[model]; exists {
		queue.Enqueue(req)
	} else {
		qm.defaultQueue.Enqueue(req)
	}
}

// Shutdown gracefully shuts down all queues.
func (qm *QueueManager) Shutdown() {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	// Close all model-specific queues.
	for _, queue := range qm.queues {
		queue.Shutdown()
	}

	// Close the default queue.
	qm.defaultQueue.Shutdown()
}
