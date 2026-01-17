package models

import "time"

type GetAllSecretsResponse struct {
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Payload   []byte    `json:"payload"`
	Meta      *string   `json:"meta,omitempty"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}
