package handler

// RequestPayload represents the expected JSON structure in the request body.
type RequestPayload struct {
	Model string `json:"model"`
	Input string `json:"input"`
}
