package model

// GenerateResponse represents response for POST .../generate
type GenerateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Address string `json:"address,omitempty"`
}
