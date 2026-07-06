package model

import "time"

// ReleaseAlert is a new or upcoming release by a library artist, found by
// the periodic release checker.
type ReleaseAlert struct {
	ID          string    `json:"id"` // "<source>:<external_id>"
	ArtistID    string    `json:"artistId"`
	ArtistName  string    `json:"artistName"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	ReleaseType string    `json:"releaseType"`
	ReleaseDate string    `json:"releaseDate"`
	ExternalURL string    `json:"externalUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ReleaseAlertRepository interface {
	// InsertIfMissing adds the alert unless its id already exists; returns
	// whether a row was inserted.
	InsertIfMissing(alert ReleaseAlert) (bool, error)
	GetAll(limit int) ([]ReleaseAlert, error)
	CountSince(t time.Time) (int64, error)
}
