package model

import "time"

type Scrobble struct {
	MediaFileID    string
	UserID         string
	SubmissionTime time.Time
}

// PlayEntry is a single entry in the server-wide play history feed.
type PlayEntry struct {
	UserID         string    `json:"userId"`
	UserName       string    `json:"userName"`
	MediaFileID    string    `json:"mediaFileId"`
	Title          string    `json:"title"`
	Artist         string    `json:"artist"`
	ArtistID       string    `json:"artistId"`
	Album          string    `json:"album"`
	AlbumID        string    `json:"albumId"`
	SubmissionTime time.Time `json:"submissionTime"`
}

// TopEntry is an aggregated play-count ranking entry (song, album or artist).
type TopEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist,omitempty"`
	ArtistID  string `json:"artistId,omitempty"`
	AlbumID   string `json:"albumId,omitempty"`
	PlayCount int64  `json:"playCount"`
}

// ScrobbleUser is a user that has at least one recorded play.
type ScrobbleUser struct {
	ID       string `json:"id"`
	UserName string `json:"userName"`
}

const (
	TopKindSong   = "song"
	TopKindAlbum  = "album"
	TopKindArtist = "artist"
)

type ScrobbleRepository interface {
	RecordScrobble(mediaFileID string, submissionTime time.Time) error

	// Recent returns the server-wide play feed, newest first, optionally
	// filtered to the given user ids.
	Recent(limit, offset int, userIDs []string) ([]PlayEntry, error)

	// Top returns play-count rankings grouped by kind (song/album/artist),
	// counting plays since the given time (zero time = all time),
	// optionally filtered to the given user ids.
	Top(kind string, since time.Time, limit int, userIDs []string) ([]TopEntry, error)

	// ActiveUsers returns the users that have at least one recorded play.
	ActiveUsers() ([]ScrobbleUser, error)
}
