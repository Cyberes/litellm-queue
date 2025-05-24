package queue

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// process continuously processes requests from the queue
func (q *RequestQueue) process() {
	for {
		q.mutex.Lock()
		// Wait until there is a request in the queue or shutdown is signaled
		for len(q.requests) == 0 {
			q.cond.Wait()
			select {
			case <-q.shutdownCh:
				q.mutex.Unlock()
				return
			default:
				// Continue processing
			}
		}

		// Dequeue the first request
		req := q.requests[0]
		q.requests = q.requests[1:]
		q.mutex.Unlock()

		// Calculate the remaining time before timeout
		elapsed := time.Since(req.EnqueuedAt)
		remainingTime := 99*time.Second - elapsed
		if remainingTime <= 0 {
			// Timeout has already been exceeded
			req.ResponseChan <- RequestResult{
				StatusCode: http.StatusServiceUnavailable,
				Headers:    nil,
				Body:       []byte("Service Unavailable: request timed out in queue"),
				Error:      nil,
			}
			continue
		}

		// Create a context with the remaining timeout
		ctx, cancel := context.WithTimeout(req.Context, remainingTime)
		defer cancel()

		// Channel to receive the processing result
		doneChan := make(chan RequestResult)

		// Process the request in a separate goroutine
		go func(r *Request) {
			// Forward the request to the backend
			resp, err := q.backend.Forward(r.Method, r.URL, r.Headers, bytes.NewReader(r.Body))
			if err != nil {
				doneChan <- RequestResult{
					StatusCode: http.StatusBadGateway,
					Headers:    nil,
					Body:       []byte("Bad Gateway: failed to reach backend"),
					Error:      err,
				}
				return
			}
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				doneChan <- RequestResult{
					StatusCode: http.StatusInternalServerError,
					Headers:    nil,
					Body:       []byte("Internal Server Error: failed to read backend response"),
					Error:      err,
				}
				return
			}

			// Prepare the result
			result := RequestResult{
				StatusCode: resp.StatusCode,
				Headers:    resp.Header,
				Body:       body,
				Error:      nil,
			}

			doneChan <- result
		}(req)

		// Wait for either the processing to complete, context timeout, or client cancellation
		select {
		case res := <-doneChan:
			req.ResponseChan <- res
		case <-ctx.Done():
			// Timeout occurred
			req.ResponseChan <- RequestResult{
				StatusCode: http.StatusServiceUnavailable,
				Headers:    nil,
				Body:       []byte("Service Unavailable: request timed out in queue"),
				Error:      nil,
			}
		}
	}
}
