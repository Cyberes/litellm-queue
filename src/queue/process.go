package queue

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// process continuously processes requests from the queue.
func (rq *RequestQueue) process() {
	for req := range rq.queue {
		rq.incrementActive()

		// Ensure each request respects the 99-second timeout
		ctx, cancel := context.WithTimeout(req.Context, 99*time.Second)
		defer cancel()

		// Forward the request to the backend
		resp, err := rq.backend.Forward(ctx, req.Method, req.URL, req.Headers, bytes.NewReader(req.Body))
		if err != nil {
			req.ResponseChan <- RequestResult{
				StatusCode: http.StatusBadGateway,
				Headers:    nil,
				Body:       []byte("Bad Gateway: failed to reach backend"),
				Error:      err,
			}
			rq.decrementActive()
			continue
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			req.ResponseChan <- RequestResult{
				StatusCode: http.StatusInternalServerError,
				Headers:    nil,
				Body:       []byte("Internal Server Error: failed to read backend response"),
				Error:      err,
			}
			resp.Body.Close()
			rq.decrementActive()
			continue
		}
		resp.Body.Close()

		// Prepare the result
		result := RequestResult{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       body,
			Error:      nil,
		}

		req.ResponseChan <- result
		rq.decrementActive()
	}
}
