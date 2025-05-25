// handler/http_handler.go

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"server/manager"
	"time"
)

type contextKey string

const errorChannelKey = contextKey("errorChannel")

// HTTPHandler handles incoming HTTP requests and proxies them to the backend.
type HTTPHandler struct {
	ConcurrencyManager *manager.ConcurrencyManager
	ReverseProxy       *httputil.ReverseProxy
	Timeout            time.Duration
}

// NewHTTPHandler creates a new instance of HTTPHandler.
func NewHTTPHandler(cm *manager.ConcurrencyManager, backendURL string) *HTTPHandler {
	parsedURL, err := url.Parse(backendURL)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
	}

	proxy.Transport = &http.Transport{}

	// Send any errors back to the caller.
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		// Retrieve the error channel from the context
		if ch, ok := req.Context().Value(errorChannelKey).(chan error); ok {
			// Non-blocking send to the channel
			select {
			case ch <- err:
			default:
			}
		}
	}

	return &HTTPHandler{
		ConcurrencyManager: cm,
		ReverseProxy:       proxy,
		Timeout:            99 * time.Second,
	}
}

// ServeHTTP implements the http.Handler interface for HTTPHandler.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only handle specific HTTP methods if necessary
	//if r.Method != http.MethodPost {
	//	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	//	return
	//}

	// Read and buffer the request body to extract the model
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logAndReturnError(w, r, "Bad Request: unable to read body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Parse JSON to extract the model
	var payload RequestPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		logAndReturnError(w, r, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	model := payload.Model
	if model == "" {
		model = "default"
	}

	// Attempt to acquire concurrency slot
	release, ok := h.ConcurrencyManager.Acquire(model)
	if !ok {
		logAndReturnError(w, r, "Service Unavailable: too many requests in queue", http.StatusServiceUnavailable)
		return
	}
	defer release()

	// Restore the request body for ReverseProxy
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	r.ContentLength = int64(len(bodyBytes))
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	// Replace the request's context with the new context
	r = r.WithContext(ctx)

	// Create an error channel for this request
	errChan := make(chan error, 1)

	// Inject the error channel into the request's context
	ctx = context.WithValue(ctx, errorChannelKey, errChan)
	r = r.WithContext(ctx)

	// Handle client disconnect
	notify := r.Context().Done()
	done := make(chan struct{})

	go func() {
		select {
		case <-notify:
			log.Debugf("Client %s disconnected", r.RemoteAddr)
			cancel()
		case <-done:
			// Proxying completed
		}
	}()

	// Serve the request using ReverseProxy
	h.ReverseProxy.ServeHTTP(w, r)

	// Signal that proxying is done
	close(done)

	// Check if an error was reported by the ErrorHandler
	select {
	case err := <-errChan:
		// Respond to the client based on the error type
		if errors.Is(err, context.Canceled) {
			// Client canceled the request
			logAndReturnError(w, r, "Client canceled the request", http.StatusBadRequest)
		} else {
			logAndReturnError(w, r, "Bad Gateway: failed to reach backend", http.StatusBadGateway)
		}
	default:
		// No error occurred; proceed to log the successful request
		logRequest(r)
	}
}

// responseRecorder is a custom ResponseWriter to capture the status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
