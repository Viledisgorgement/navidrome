package community

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/adapters/metalarchives"
	"github.com/navidrome/navidrome/adapters/musicbrainz"
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
	// Lets the UI fetch cover art from the Cover Art Archive
	MbzReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
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
	if s.ma != nil && s.needsRefresh(repo, artistID, model.ReleaseSourceMetalArchives, refresh) {
		s.refreshMetalArchives(ctx, artist, repo)
	}

	cached, err := repo.GetByArtist(artistID)
	if err != nil {
		return nil, err
	}
	// Metal Archives entries win the cross-source dedupe below: its release
	// types (Demo/Split/EP distinctions) are more reliable for metal
	sort.SliceStable(cached, func(i, j int) bool {
		return cached[i].Source == model.ReleaseSourceMetalArchives &&
			cached[j].Source != model.ReleaseSourceMetalArchives
	})

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
	seen := map[string]int{}
	for _, release := range cached {
		key := fmt.Sprintf("%s|%d", normalizeTitle(release.Title), release.Year)
		sources[release.Source] = true
		if idx, dup := seen[key]; dup {
			// Metal Archives entries win the dedupe (sorted first), but the
			// MusicBrainz duplicate still contributes its release-group id —
			// it drives cover art and exact ownership matching
			existing := &result.Releases[idx]
			if existing.MbzReleaseGroupID == "" && release.MbzReleaseGroupID != "" {
				existing.MbzReleaseGroupID = release.MbzReleaseGroupID
				if album := byReleaseGroup[release.MbzReleaseGroupID]; !existing.Owned && album != nil {
					existing.Owned = true
					existing.AlbumID = album.ID
				}
			}
			continue
		}
		seen[key] = len(result.Releases)

		entry := DiscographyRelease{
			Title:             release.Title,
			Year:              release.Year,
			ReleaseType:       release.ReleaseType,
			ExternalURL:       release.ExternalURL,
			Source:            release.Source,
			MbzReleaseGroupID: release.MbzReleaseGroupID,
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
	info, err := repo.GetArtistInfo(artist.ID, model.ReleaseSourceMusicBrainz)
	if err != nil {
		info = &model.ExternalArtistInfo{
			ArtistID: artist.ID,
			Source:   model.ReleaseSourceMusicBrainz,
		}
	}
	info.LastFetchedAt = time.Now()

	// Prefer the mbid from the files' tags; otherwise fall back to a
	// previously resolved id, then to a one-time name search
	mbid := artist.MbzArtistID
	if mbid == "" {
		mbid = info.ExternalArtistID
	}
	if mbid == "" {
		mbid, err = s.mbz.SearchArtist(ctx, artist.Name)
		if errors.Is(err, musicbrainz.ErrArtistNotFound) {
			info.FetchStatus = "not_found"
			_ = repo.PutArtistInfo(info)
			return
		}
		if err != nil {
			log.Warn(ctx, "MusicBrainz artist search failed", "artist", artist.Name, err)
			info.FetchStatus = "error"
			_ = repo.PutArtistInfo(info)
			return
		}
		log.Info(ctx, "Resolved artist on MusicBrainz by name", "artist", artist.Name, "mbid", mbid)
	}
	info.ExternalArtistID = mbid

	groups, err := s.mbz.ReleaseGroupsByArtist(ctx, mbid)
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

func (s *service) refreshMetalArchives(ctx context.Context, artist *model.Artist, repo model.ExternalReleaseRepository) {
	info, err := repo.GetArtistInfo(artist.ID, model.ReleaseSourceMetalArchives)
	if err != nil {
		info = &model.ExternalArtistInfo{
			ArtistID: artist.ID,
			Source:   model.ReleaseSourceMetalArchives,
		}
	}
	info.LastFetchedAt = time.Now()

	// The band id search only runs until it succeeds once; after that the
	// resolved id is reused from external_artist_info
	if info.ExternalArtistID == "" {
		bandID, err := s.ma.SearchBand(ctx, artist.Name)
		if errors.Is(err, metalarchives.ErrNotFound) {
			info.FetchStatus = "not_found"
			_ = repo.PutArtistInfo(info)
			return
		}
		if err != nil {
			log.Warn(ctx, "Metal Archives band search failed", "artist", artist.Name, err)
			info.FetchStatus = "error"
			_ = repo.PutArtistInfo(info)
			return
		}
		info.ExternalArtistID = bandID
	}

	maReleases, err := s.ma.Discography(ctx, info.ExternalArtistID)
	if err != nil {
		log.Warn(ctx, "Metal Archives discography fetch failed", "artist", artist.Name, "bandId", info.ExternalArtistID, err)
		info.FetchStatus = "error"
		_ = repo.PutArtistInfo(info)
		return
	}
	releases := make([]model.ExternalRelease, 0, len(maReleases))
	for _, r := range maReleases {
		releases = append(releases, model.ExternalRelease{
			ID:          model.ReleaseSourceMetalArchives + ":" + r.ID,
			ArtistID:    artist.ID,
			Source:      model.ReleaseSourceMetalArchives,
			ExternalID:  r.ID,
			Title:       r.Title,
			ReleaseType: r.Type,
			Year:        r.Year,
			ExternalURL: r.URL,
		})
	}
	if err := repo.ReplaceArtistReleases(artist.ID, model.ReleaseSourceMetalArchives, releases); err != nil {
		log.Error(ctx, "Error caching Metal Archives discography", "artist", artist.Name, err)
		info.FetchStatus = "error"
	} else {
		info.FetchStatus = "ok"
		log.Info(ctx, "Cached Metal Archives discography", "artist", artist.Name, "bandId", info.ExternalArtistID, "releases", len(releases))
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
