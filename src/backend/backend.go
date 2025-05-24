package backend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a client to communicate with the backend API server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewBackendClient creates a new BackendClient with the specified base URL.
func NewBackendClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Backend request timeout
		},
	}
}

// Forward sends the HTTP request to the backend server and returns the response.
func (c *Client) Forward(ctx context.Context, method, path string, headers http.Header, body io.Reader) (*http.Response, error) {
	// Construct the full URL.
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	// Create a new HTTP request with context.
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// Copy headers.
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Send the request to the backend.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
