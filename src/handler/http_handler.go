package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/queue"
	"time"
)

// HTTPHandler handles incoming HTTP requests and interacts with the QueueManager.
type HTTPHandler struct {
	QueueManager *queue.QueueManager
}

// NewHTTPHandler creates a new instance of HTTPHandler.
func NewHTTPHandler(qm *queue.QueueManager) *HTTPHandler {
	return &HTTPHandler{
		QueueManager: qm,
	}
}

// ServeHTTP implements the http.Handler interface for HTTPHandler.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, callingRequest *http.Request) {
	// Create a context that cancels when the client disconnects.
	ctx, cancel := context.WithCancel(callingRequest.Context())
	defer cancel()

	// Channel to listen for client disconnection.
	notify := callingRequest.Context().Done()

	// Read the request body.
	body, err := io.ReadAll(callingRequest.Body)
	if err != nil {
		logAndReturnError(w, callingRequest, "Bad Request: unable to read body", http.StatusBadRequest)
		return
	}
	callingRequest.Body.Close()

	// Parse JSON to extract the model.
	var payload RequestPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		logAndReturnError(w, callingRequest, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}
	model := payload.Model

	// Create a channel to receive the response.
	responseChan := make(chan queue.RequestResult)

	// Create a request object to enqueue.
	req := &queue.Request{
		Method:       callingRequest.Method,
		URL:          callingRequest.URL.Path,
		Headers:      callingRequest.Header,
		Body:         body,
		ResponseChan: responseChan,
		Context:      ctx,
		EnqueuedAt:   time.Now(),
	}

	// Enqueue the request into the appropriate queue.
	h.QueueManager.Enqueue(model, req)

	// Channel to enforce the 99-second timeout.
	timeout := time.After(99 * time.Second)

	// Wait for the response, timeout, or client disconnection.
	select {
	case res := <-responseChan:
		if res.Error != nil {
			logAndReturnError(w, callingRequest, "Error processing request", res.StatusCode, fmt.Sprintf("Error processing request: %s", res.Error.Error()))
			return
		}
		// Write headers.
		for key, values := range res.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		// Write status code.
		w.WriteHeader(res.StatusCode)
		// Write body.
		w.Write(res.Body)
		logRequest(callingRequest)
	case <-notify:
		// Client has disconnected.
		log.Debugf("Client %s disconnected", callingRequest.RemoteAddr)
		// Note: The processing goroutine respects the request's context and will handle cancellation.
	case <-timeout:
		// Request has been in the queue for more than 99 seconds.
		logAndReturnError(w, callingRequest, "Request timed out", http.StatusRequestTimeout)
	}
}
