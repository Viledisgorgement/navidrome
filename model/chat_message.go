package model

import "time"

// ChatMessage is a message posted to the server-wide community chat. It may
// carry an uploaded image, stored on disk and referenced by filename.
type ChatMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Message   string    `json:"message"`
	Image     string    `json:"-"` // image filename on disk, not exposed
	HasImage  bool      `json:"hasImage"`
	CreatedAt time.Time `json:"createdAt"`
}

type ChatMessageRepository interface {
	Add(msg ChatMessage) error
	// GetRecent returns messages newest-first. If before is non-zero, only
	// messages created strictly before it are returned.
	GetRecent(limit int, before time.Time) ([]ChatMessage, error)
	Get(id string) (*ChatMessage, error)
	Delete(id string) error
}
