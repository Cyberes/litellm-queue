package manager

import (
	"sync"
	"time"

	"server/config" // Adjust the import path based on your project structure
)

// ModelMetrics holds the metrics for a specific model.
type ModelMetrics struct {
	Model                  string
	QueueSize              int
	ProcessingCount        int
	LastLogTime            time.Time
	queueSizeChanged       bool
	processingCountChanged bool
	mu                     sync.Mutex
}

// ConcurrencyManager manages concurrency limits and metrics for models.
type ConcurrencyManager struct {
	semMap      map[string]chan struct{}
	metricsMap  map[string]*ModelMetrics
	mu          sync.Mutex
	defaultSize int
}

// NewConcurrencyManager initializes a new ConcurrencyManager with model configurations and a default concurrency limit.
func NewConcurrencyManager(modelConfigs []config.ModelConfigEntry, defaultSize int) *ConcurrencyManager {
	cm := &ConcurrencyManager{
		semMap:      make(map[string]chan struct{}),
		metricsMap:  make(map[string]*ModelMetrics),
		defaultSize: defaultSize,
	}

	for _, cfg := range modelConfigs {
		// Validate size
		size := cfg.Size
		if size <= 0 {
			size = 10 // Set a sensible default if invalid
			log.Warnf("Model '%s' has invalid size %d. Setting to default size %d.", cfg.Name, cfg.Size, size)
		}
		cm.semMap[cfg.Name] = make(chan struct{}, size)
		cm.metricsMap[cfg.Name] = &ModelMetrics{
			Model: cfg.Name,
		}
	}

	// Initialize default model
	cm.semMap["default"] = make(chan struct{}, cm.defaultSize)
	cm.metricsMap["default"] = &ModelMetrics{
		Model: "default",
	}

	// Start monitoring goroutines for each model
	for _, metrics := range cm.metricsMap {
		go cm.monitorMetrics(metrics)
	}

	return cm
}

// Acquire attempts to acquire a semaphore slot for the given model.
// It increments the queue size, and upon successful acquisition, decrements the queue size and increments the processing count.
func (cm *ConcurrencyManager) Acquire(model string) (func(), bool) {
	cm.mu.Lock()
	sem, exists := cm.semMap[model]
	if !exists {
		sem = cm.semMap["default"]
		model = "default"
	}
	metrics := cm.metricsMap[model]
	cm.mu.Unlock()

	// Increment queue size
	metrics.incrementQueue()

	select {
	case sem <- struct{}{}:
		// Acquired
		metrics.incrementProcessing()
		metrics.decrementQueue()

		return func() {
			metrics.decrementProcessing()
			<-sem
		}, true
	case <-time.After(75 * time.Second):
		// Timeout after 75 seconds
		metrics.decrementQueue()
		return nil, false
	}
}

// monitorMetrics monitors changes in the metrics and logs them appropriately.
func (cm *ConcurrencyManager) monitorMetrics(metrics *ModelMetrics) {
	for {
		time.Sleep(500 * time.Millisecond) // Check twice every second

		metrics.mu.Lock()
		currentTime := time.Now()
		if (metrics.queueSizeChanged || metrics.processingCountChanged) &&
			currentTime.Sub(metrics.LastLogTime) >= time.Second {
			log.Infof("Model: %s | Queued: %d | Processing: %d",
				metrics.Model, metrics.QueueSize, metrics.ProcessingCount)
			metrics.LastLogTime = currentTime
			metrics.resetChangeFlags()
		}
		metrics.mu.Unlock()
	}
}

// Methods for ModelMetrics

func (m *ModelMetrics) incrementQueue() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueueSize++
	m.queueSizeChanged = true
}

func (m *ModelMetrics) decrementQueue() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.QueueSize > 0 {
		m.QueueSize--
		m.queueSizeChanged = true
	}
}

func (m *ModelMetrics) incrementProcessing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProcessingCount++
	m.processingCountChanged = true
}

func (m *ModelMetrics) decrementProcessing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ProcessingCount > 0 {
		m.ProcessingCount--
		m.processingCountChanged = true
	}
}

func (m *ModelMetrics) resetChangeFlags() {
	m.queueSizeChanged = false
	m.processingCountChanged = false
}

// Shutdown performs any necessary cleanup. (No action needed for semaphores)
func (cm *ConcurrencyManager) Shutdown() {
	// Implement if needed
}
