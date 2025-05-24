package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"server/queue"
	"time"
)

// HTTPHandler handles incoming HTTP requests and interacts with the request queue
type HTTPHandler struct {
	Queue *queue.RequestQueue
}

// NewHTTPHandler creates a new instance of HTTPHandler
func NewHTTPHandler(q *queue.RequestQueue) *HTTPHandler {
	return &HTTPHandler{
		Queue: q,
	}
}

// ServeHTTP implements the http.Handler interface for HTTPHandler
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, callingRequest *http.Request) {
	// callingRequest is the HTTP request from the client that initiated this function.

	// Create a context that cancels when the client disconnects
	ctx, cancel := context.WithCancel(callingRequest.Context())
	defer cancel()

	// Channel to listen for client disconnection
	notify := callingRequest.Context().Done()

	// Read the request body
	body, err := io.ReadAll(callingRequest.Body)
	if err != nil {
		logAndReturnError(w, "Bad Request: unable to read body", http.StatusBadRequest)
		return
	}
	callingRequest.Body.Close()

	// Create a channel to receive the response
	responseChan := make(chan queue.RequestResult)

	// Create a request object to enqueue
	req := &queue.Request{
		Method:       callingRequest.Method,
		URL:          callingRequest.URL.Path,
		Headers:      callingRequest.Header,
		Body:         body,
		ResponseChan: responseChan,
		Context:      ctx,
	}

	// Enqueue the request
	h.Queue.Enqueue(req)

	// Channel to enforce the 99-second timeout
	timeout := time.After(99 * time.Second)

	// Wait for the response, timeout, or client disconnection
	select {
	case res := <-responseChan:
		if res.Error != nil {
			logAndReturnError(w, "Error processing request", res.StatusCode, fmt.Sprintf("Error processing request: %s", res.Error.Error()))
			return
		}
		// Write headers
		for key, values := range res.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		// Write status code
		w.WriteHeader(res.StatusCode)
		// Write body
		w.Write(res.Body)
		logRequest(callingRequest)
	case <-notify:
		// Client has disconnected, remove the request from the queue
		h.Queue.Remove(req)
		log.Debugf("Client %s disconnected, request removed from queue", callingRequest.RemoteAddr)
	case <-timeout:
		// Request has been in the queue for more than 99 seconds
		logAndReturnError(w, "Request timed out", http.StatusRequestTimeout)
	}
}
