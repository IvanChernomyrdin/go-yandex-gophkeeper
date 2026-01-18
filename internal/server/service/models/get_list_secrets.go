package models

import "time"

type SecretResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Payload   []byte    `json:"payload"`
	Meta      *string   `json:"meta,omitempty"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}
type GetAllSecretsResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

type UpdateSecretRequest struct {
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Payload []byte  `json:"payload"`
	Meta    *string `json:"meta"`
	Version int     `json:"version"`
}
