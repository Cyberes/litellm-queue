package queue

import (
	"context"
	"net/http"
	"time"
)

// Request represents an HTTP request to be processed.
type Request struct {
	Method       string
	URL          string
	Headers      http.Header
	Body         []byte
	ResponseChan chan RequestResult
	Context      context.Context
	EnqueuedAt   time.Time
}

// RequestResult represents the outcome of processing a request.
type RequestResult struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Error      error
}
