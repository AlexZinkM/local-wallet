package model

// ErrorResponse is the consistent JSON structure for all API error responses.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
