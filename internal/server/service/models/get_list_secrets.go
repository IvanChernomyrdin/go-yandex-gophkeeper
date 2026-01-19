package models

type UpdateSecretRequest struct {
	Type    *string `json:"type,omitempty"`
	Title   *string `json:"title,omitempty"`
	Payload *string `json:"payload,omitempty"`
	Meta    *string `json:"meta,omitempty"`
	Version int     `json:"version"`
}
