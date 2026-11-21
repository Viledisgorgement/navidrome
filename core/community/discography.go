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
	"github.com/navidrome/navidrome/persistence"
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

	// Same album set the artist page shows: albums where the artist is an
	// album artist OR a track artist. album.album_artist_id alone only holds
	// the first album artist, so albums credited to e.g. "TPR & Roxane Genot"
	// or tagged with a different primary album artist were never considered
	albums, err := s.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Or{
				squirrel.Eq{"album.album_artist_id": artistID},
				persistence.Exists("json_tree(participants, '$.albumartist')", squirrel.Eq{"value": artistID}),
				persistence.Exists("json_tree(participants, '$.artist')", squirrel.Eq{"value": artistID}),
			},
			squirrel.Eq{"album.missing": false},
		},
	})
	if err != nil {
		return nil, err
	}
	matcher := newAlbumMatcher(albums)

	result := &Discography{ArtistID: artistID, ArtistName: artist.Name}
	sources := map[string]bool{}
	seen := map[string]int{}
	for _, release := range cached {
		// Bandcamp entries are only a new-release bookkeeping cache for the
		// alert checker (most have no date), not part of the discography
		if release.Source == model.ReleaseSourceBandcamp {
			continue
		}
		key := fmt.Sprintf("%s|%d", normalizeTitle(release.Title), release.Year)
		sources[release.Source] = true
		if idx, dup := seen[key]; dup {
			// Metal Archives entries win the dedupe (sorted first), but the
			// MusicBrainz duplicate still contributes its release-group id —
			// it drives cover art and exact ownership matching
			existing := &result.Releases[idx]
			if existing.MbzReleaseGroupID == "" && release.MbzReleaseGroupID != "" {
				existing.MbzReleaseGroupID = release.MbzReleaseGroupID
				if album := matcher.byReleaseGroup[release.MbzReleaseGroupID]; !existing.Owned && album != nil {
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
		if album := matcher.match(release.MbzReleaseGroupID, release.Title, release.Year); album != nil {
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

var (
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	// Trailing "(Remastered)", "[Deluxe Edition]", "(2014 Reissue)" etc.
	editionSuffix = regexp.MustCompile(`\s*[(\[][^)\]]*\b(remaster(ed)?|deluxe|edition|reissue|bonus tracks?|expanded|anniversary)\b[^)\]]*[)\]]\s*$`)
	// Separator between a release's main title and its subtitle:
	// "Buried in the Snow: A Melancholy Tribute to…", "Farewell Hyrule King - Melancholy music from…"
	subtitleSep = regexp.MustCompile(`\s*(:|\s[-–—]\s)\s*`)
)

// normalizeTitle lowercases and strips punctuation/whitespace so that
// "Slaughter of the Soul (Remastered)" matches "slaughter of the soul"
// and "VII, VIII & IX" matches "VII, VIII and IX" across sources.
func normalizeTitle(title string) string {
	normalized := strings.ToLower(title)
	normalized = strings.NewReplacer("&", " and ", "+", " and ", "’", "'", "‘", "'").Replace(normalized)
	for {
		trimmed := editionSuffix.ReplaceAllString(normalized, "")
		if trimmed == normalized {
			break
		}
		normalized = trimmed
	}
	normalized = strings.ReplaceAll(normalized, "'", "")
	return strings.Trim(nonAlphaNum.ReplaceAllString(normalized, " "), " ")
}

// mainTitle returns the normalized part of a title before its subtitle
// ("Buried in the Snow: A Melancholy Tribute…" -> "buried in the snow"), or
// "" when the title has no subtitle.
func mainTitle(title string) string {
	loc := subtitleSep.FindStringIndex(title)
	if loc == nil || loc[0] == 0 {
		return ""
	}
	return normalizeTitle(title[:loc[0]])
}

// albumMatcher decides whether an external release exists in the library.
type albumMatcher struct {
	byReleaseGroup map[string]*model.Album
	byTitle        map[string]*model.Album
	byMainTitle    map[string][]*model.Album
}

func newAlbumMatcher(albums model.Albums) *albumMatcher {
	m := &albumMatcher{
		byReleaseGroup: map[string]*model.Album{},
		byTitle:        map[string]*model.Album{},
		byMainTitle:    map[string][]*model.Album{},
	}
	for i := range albums {
		album := &albums[i]
		if album.MbzReleaseGroupID != "" {
			m.byReleaseGroup[album.MbzReleaseGroupID] = album
		}
		m.byTitle[normalizeTitle(album.Name)] = album
		main := mainTitle(album.Name)
		if main == "" {
			main = normalizeTitle(album.Name)
		}
		m.byMainTitle[main] = append(m.byMainTitle[main], album)
	}
	return m
}

// match tries, in order: the MusicBrainz release-group id, the full
// normalized title, then the main title (text before ":" or " - ") when the
// release years agree. The last step covers sources that word the subtitle
// differently ("…Final Fantasy VII, VIII & IX" vs "…VII VIII and IX").
// The year guard keeps different releases sharing a main title apart, e.g.
// "Fragments of Memories" (2013) vs its 2017 Definitive Edition.
func (m *albumMatcher) match(releaseGroupID, title string, year int) *model.Album {
	if releaseGroupID != "" {
		if album := m.byReleaseGroup[releaseGroupID]; album != nil {
			return album
		}
	}
	if album := m.byTitle[normalizeTitle(title)]; album != nil {
		return album
	}
	main := mainTitle(title)
	if main == "" {
		main = normalizeTitle(title)
	}
	// One- or two-word main titles ("Memoria", "Reductions") are too generic
	// to match on without the subtitle
	if len(strings.Fields(main)) < 2 || year == 0 {
		return nil
	}
	for _, album := range m.byMainTitle[main] {
		if albumYearMatches(album, year) {
			return album
		}
	}
	return nil
}

func albumYearMatches(album *model.Album, year int) bool {
	for _, y := range []int{album.MinOriginalYear, album.MaxOriginalYear, album.MinYear, album.MaxYear} {
		if y != 0 && y >= year-1 && y <= year+1 {
			return true
		}
	}
	return false
}
