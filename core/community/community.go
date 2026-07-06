package community

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
)

const (
	maxFeedCount    = 200
	defaultFeedSize = 50
	defaultTopSize  = 10
)

// Service exposes server-wide listening activity: the shared play feed,
// top-played rankings and the list of users with recorded plays.
type Service interface {
	Recent(ctx context.Context, limit, offset int, userIDs []string) ([]model.PlayEntry, error)
	Top(ctx context.Context, kind, timeRange string, limit int, userIDs []string) ([]model.TopEntry, error)
	Users(ctx context.Context) ([]model.ScrobbleUser, error)
}

func NewService(ds model.DataStore) Service {
	return &service{ds: ds}
}

type service struct {
	ds model.DataStore
}

func (s *service) Recent(ctx context.Context, limit, offset int, userIDs []string) ([]model.PlayEntry, error) {
	if limit <= 0 {
		limit = defaultFeedSize
	}
	limit = min(limit, maxFeedCount)
	offset = max(offset, 0)
	return s.ds.Scrobble(ctx).Recent(limit, offset, userIDs)
}

func (s *service) Top(ctx context.Context, kind, timeRange string, limit int, userIDs []string) ([]model.TopEntry, error) {
	switch kind {
	case model.TopKindSong, model.TopKindAlbum, model.TopKindArtist:
	default:
		kind = model.TopKindSong
	}
	if limit <= 0 {
		limit = defaultTopSize
	}
	limit = min(limit, maxFeedCount)
	return s.ds.Scrobble(ctx).Top(kind, sinceFromRange(timeRange), limit, userIDs)
}

func (s *service) Users(ctx context.Context) ([]model.ScrobbleUser, error) {
	return s.ds.Scrobble(ctx).ActiveUsers()
}

// sinceFromRange converts a UI time-range token into a cutoff time.
// Unknown tokens (including "all") mean no cutoff.
func sinceFromRange(timeRange string) time.Time {
	var days int
	switch timeRange {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		return time.Time{}
	}
	return time.Now().AddDate(0, 0, -days)
}
