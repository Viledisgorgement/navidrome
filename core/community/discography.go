package community

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const discographyCacheTTL = 30 * 24 * time.Hour

type DiscographyRelease struct {
	Title       string `json:"title"`
	Year        int    `json:"year"`
	ReleaseType string `json:"releaseType"`
	Owned       bool   `json:"owned"`
	AlbumID     string `json:"albumId,omitempty"`
	ExternalURL string `json:"externalUrl,omitempty"`
	Source      string `json:"source"`
}

type Discography struct {
	ArtistID   string               `json:"artistId"`
	ArtistName string               `json:"artistName"`
	Sources    []string             `json:"sources"`
	Releases   []DiscographyRelease `json:"releases"`
}

// Discography returns the artist's full discography from external sources,
// each release flagged with whether it exists in the library. Results are
// cached in the external_release table for 30 days.
func (s *service) Discography(ctx context.Context, artistID string, refresh bool) (*Discography, error) {
	artist, err := s.ds.Artist(ctx).Get(artistID)
	if err != nil {
		return nil, err
	}
	repo := s.ds.ExternalRelease(ctx)

	if s.mbz != nil && s.needsRefresh(repo, artistID, model.ReleaseSourceMusicBrainz, refresh) {
		s.refreshMusicBrainz(ctx, artist, repo)
	}

	cached, err := repo.GetByArtist(artistID)
	if err != nil {
		return nil, err
	}

	albums, err := s.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"album.album_artist_id": artistID},
	})
	if err != nil {
		return nil, err
	}
	byReleaseGroup := map[string]*model.Album{}
	byTitle := map[string]*model.Album{}
	for i := range albums {
		album := &albums[i]
		if album.MbzReleaseGroupID != "" {
			byReleaseGroup[album.MbzReleaseGroupID] = album
		}
		byTitle[normalizeTitle(album.Name)] = album
	}

	result := &Discography{ArtistID: artistID, ArtistName: artist.Name}
	sources := map[string]bool{}
	seen := map[string]bool{}
	for _, release := range cached {
		key := fmt.Sprintf("%s|%d", normalizeTitle(release.Title), release.Year)
		if seen[key] {
			continue
		}
		seen[key] = true
		sources[release.Source] = true

		entry := DiscographyRelease{
			Title:       release.Title,
			Year:        release.Year,
			ReleaseType: release.ReleaseType,
			ExternalURL: release.ExternalURL,
			Source:      release.Source,
		}
		if album := byReleaseGroup[release.MbzReleaseGroupID]; release.MbzReleaseGroupID != "" && album != nil {
			entry.Owned = true
			entry.AlbumID = album.ID
		} else if album := byTitle[normalizeTitle(release.Title)]; album != nil {
			entry.Owned = true
			entry.AlbumID = album.ID
		}
		result.Releases = append(result.Releases, entry)
	}
	for source := range sources {
		result.Sources = append(result.Sources, source)
	}
	return result, nil
}

func (s *service) needsRefresh(repo model.ExternalReleaseRepository, artistID, source string, force bool) bool {
	if force {
		return true
	}
	info, err := repo.GetArtistInfo(artistID, source)
	if errors.Is(err, model.ErrNotFound) {
		return true
	}
	if err != nil {
		return false
	}
	// Retry no-mbid/error states more often than successful fetches
	ttl := discographyCacheTTL
	if info.FetchStatus != "ok" {
		ttl = 24 * time.Hour
	}
	return time.Since(info.LastFetchedAt) > ttl
}

func (s *service) refreshMusicBrainz(ctx context.Context, artist *model.Artist, repo model.ExternalReleaseRepository) {
	info := &model.ExternalArtistInfo{
		ArtistID:         artist.ID,
		Source:           model.ReleaseSourceMusicBrainz,
		ExternalArtistID: artist.MbzArtistID,
		LastFetchedAt:    time.Now(),
	}
	if artist.MbzArtistID == "" {
		info.FetchStatus = "no_mbid"
		_ = repo.PutArtistInfo(info)
		return
	}
	groups, err := s.mbz.ReleaseGroupsByArtist(ctx, artist.MbzArtistID)
	if err != nil {
		log.Warn(ctx, "Error fetching MusicBrainz discography", "artist", artist.Name, "mbid", artist.MbzArtistID, err)
		info.FetchStatus = "error"
		_ = repo.PutArtistInfo(info)
		return
	}
	releases := make([]model.ExternalRelease, 0, len(groups))
	for _, rg := range groups {
		releases = append(releases, model.ExternalRelease{
			ID:                model.ReleaseSourceMusicBrainz + ":" + rg.ID,
			ArtistID:          artist.ID,
			Source:            model.ReleaseSourceMusicBrainz,
			ExternalID:        rg.ID,
			Title:             rg.Title,
			ReleaseType:       rg.ReleaseType(),
			ReleaseDate:       rg.FirstReleaseDate,
			Year:              rg.Year(),
			MbzReleaseGroupID: rg.ID,
			ExternalURL:       "https://musicbrainz.org/release-group/" + rg.ID,
		})
	}
	if err := repo.ReplaceArtistReleases(artist.ID, model.ReleaseSourceMusicBrainz, releases); err != nil {
		log.Error(ctx, "Error caching MusicBrainz discography", "artist", artist.Name, err)
		info.FetchStatus = "error"
	} else {
		info.FetchStatus = "ok"
		log.Info(ctx, "Cached MusicBrainz discography", "artist", artist.Name, "releases", len(releases))
	}
	_ = repo.PutArtistInfo(info)
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// normalizeTitle lowercases and strips punctuation/whitespace so that
// "Slaughter of the Soul (Remastered)" matches "slaughter of the soul
// remastered"-ish variants from different sources.
func normalizeTitle(title string) string {
	normalized := strings.ToLower(title)
	for _, suffix := range []string{"(remaster)", "(remastered)", "(deluxe edition)", "(reissue)", "(bonus track edition)"} {
		normalized = strings.TrimSpace(strings.TrimSuffix(normalized, suffix))
	}
	return strings.Trim(nonAlphaNum.ReplaceAllString(normalized, " "), " ")
}
