package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"server/manager"
	"strings"
	"time"
)

type contextKey string

const errorChannelKey = contextKey("errorChannel")

// HTTPHandler handles incoming HTTP requests and proxies them to the backend
type HTTPHandler struct {
	ConcurrencyManager *manager.ConcurrencyManager
	ReverseProxy       *httputil.ReverseProxy
	Timeout            time.Duration
}

// NewHTTPHandler creates a new instance of HTTPHandler
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

	// Avoid any sort of buffering
	proxy.Transport = &http.Transport{
		DisableCompression:  true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ExpectContinueTimeout: 1 * time.Second,
	}
	proxy.FlushInterval = 100 * time.Millisecond

	// Send any errors back to the caller
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		if ch, ok := req.Context().Value(errorChannelKey).(chan error); ok {
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
	// Capture just enough data to parse the JSON
	buf := &bytes.Buffer{}
	teeReader := io.TeeReader(r.Body, buf)
	decoder := json.NewDecoder(teeReader)
	var payload RequestPayload
	if err := decoder.Decode(&payload); err != nil {
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
	r.Body = io.NopCloser(io.MultiReader(buf, r.Body))

	// Add client's IP to X-Forwarded-For
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior, ok := r.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		r.Header.Set("X-Forwarded-For", clientIP)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	// Inject the error channel into the request's context
	r = r.WithContext(ctx)
	errChan := make(chan error, 1)
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
	close(done)

	// Check if an error was reported by the ErrorHandler
	select {
	case err := <-errChan:
		if errors.Is(err, context.Canceled) {
			logAndReturnError(w, r, "Client canceled the request", http.StatusBadRequest)
		} else {
			logAndReturnError(w, r, "Bad Gateway: failed to reach backend", http.StatusBadGateway)
		}
	default:
		// No error occurred
		logRequest(r, model)
	}
}
