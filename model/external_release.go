package model

import "time"

// ExternalRelease is a release from an external discography source
// (MusicBrainz, Metal Archives), cached locally for comparison against
// the library.
type ExternalRelease struct {
	ID                string // "<source>:<external_id>"
	ArtistID          string
	Source            string
	ExternalID        string
	Title             string
	ReleaseType       string
	ReleaseDate       string
	Year              int
	MbzReleaseGroupID string
	ExternalURL       string
	FetchedAt         time.Time
}

// ExternalArtistInfo tracks per-source fetch bookkeeping for an artist,
// including the resolved external artist/band id.
type ExternalArtistInfo struct {
	ArtistID         string
	Source           string
	ExternalArtistID string
	LastFetchedAt    time.Time
	FetchStatus      string // "ok", "error", "no_mbid"
}

const (
	ReleaseSourceMusicBrainz   = "musicbrainz"
	ReleaseSourceMetalArchives = "metalarchives"
)

type ExternalReleaseRepository interface {
	// ReplaceArtistReleases atomically replaces the cached releases of an
	// artist for one source.
	ReplaceArtistReleases(artistID, source string, releases []ExternalRelease) error
	GetByArtist(artistID string) ([]ExternalRelease, error)
	GetArtistInfo(artistID, source string) (*ExternalArtistInfo, error)
	PutArtistInfo(info *ExternalArtistInfo) error
}
