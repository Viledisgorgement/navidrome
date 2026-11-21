package community

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/adapters/bandcamp"
	"github.com/navidrome/navidrome/adapters/metalarchives"
	"github.com/navidrome/navidrome/adapters/musicbrainz"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

const (
	// releases whose first release date is within this window (past) or in
	// the future count as "new"
	newReleaseWindow = 14 * 24 * time.Hour
	// don't re-check an artist against MusicBrainz more often than this
	alertCheckInterval = 20 * time.Hour
	alertSource        = "musicbrainz_alerts"

	// Bandcamp: the first time an artist is checked, only the first few
	// grid items (normally the newest) are opened for release dates and the
	// rest are recorded as already seen; after that only items that appear
	// in the grid are opened, at most bandcampMaxOpens per artist per run
	bandcampFirstRunOpens = 3
	bandcampMaxOpens      = 5
	// artists with no Bandcamp link on MusicBrainz are re-checked weekly
	bandcampNoLinkRetry = 7 * 24 * time.Hour
)

// AlertChecker is the periodic job that looks for new/upcoming releases by
// library artists and records them as release alerts.
type AlertChecker struct {
	ds     model.DataStore
	broker events.Broker
	mbz    *musicbrainz.Client
	ma     *metalarchives.Client
	bc     *bandcamp.Client

	// normalized "artistID|title" -> sources of existing alerts, so the same
	// release found by two sources is only listed once
	existing map[string]map[string]bool
}

func NewAlertChecker(ds model.DataStore, broker events.Broker) *AlertChecker {
	c := &AlertChecker{ds: ds, broker: broker}
	if conf.Server.MusicBrainz.Enabled {
		c.mbz = musicbrainz.NewClient(conf.Server.MusicBrainz.BaseURL)
	}
	if conf.Server.MetalArchives.Enabled {
		c.ma = metalarchives.NewClient(conf.Server.MetalArchives.BaseURL)
	}
	// Bandcamp pages are found through MusicBrainz artist links
	if conf.Server.Bandcamp.Enabled && c.mbz != nil {
		c.bc = bandcamp.NewClient()
	}
	return c
}

// Check runs one pass over all library artists and returns how many new
// alerts were created.
func (c *AlertChecker) Check(ctx context.Context) (int, error) {
	newAlerts := 0
	if err := c.loadExisting(ctx); err != nil {
		return 0, err
	}
	if c.mbz != nil {
		n, err := c.checkMusicBrainz(ctx)
		if err != nil {
			log.Warn(ctx, "MusicBrainz release check failed", err)
		}
		newAlerts += n
	}
	if c.ma != nil {
		n, err := c.checkMetalArchivesUpcoming(ctx)
		if err != nil {
			log.Warn(ctx, "Metal Archives upcoming releases check failed", err)
		}
		newAlerts += n
	}
	if c.bc != nil {
		n, err := c.checkBandcamp(ctx)
		if err != nil {
			log.Warn(ctx, "Bandcamp release check failed", err)
		}
		newAlerts += n
	}
	if newAlerts > 0 {
		c.broker.SendBroadcastMessage(ctx, &events.ReleaseAlert{Count: newAlerts})
		log.Info(ctx, "New release alerts", "count", newAlerts)
	}
	return newAlerts, nil
}

func (c *AlertChecker) checkMusicBrainz(ctx context.Context) (int, error) {
	artists, err := c.ds.Artist(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	releaseRepo := c.ds.ExternalRelease(ctx)
	alertRepo := c.ds.ReleaseAlert(ctx)
	cutoff := time.Now().Add(-newReleaseWindow)
	newAlerts := 0

	for _, artist := range artists {
		if ctx.Err() != nil {
			return newAlerts, ctx.Err()
		}
		// mbid from file tags, or one resolved earlier by the discography
		// name search; artists with neither are skipped (viewing an artist's
		// missing albums once is enough to enroll them)
		mbid := artistMbid(&artist, releaseRepo)
		if mbid == "" {
			continue
		}
		info, err := releaseRepo.GetArtistInfo(artist.ID, alertSource)
		if err == nil && time.Since(info.LastFetchedAt) < alertCheckInterval {
			continue
		}
		groups, err := c.mbz.ReleaseGroupsByArtist(ctx, mbid)
		status := "ok"
		if err != nil {
			log.Debug(ctx, "Release check: MusicBrainz fetch failed", "artist", artist.Name, err)
			status = "error"
		}
		for _, rg := range groups {
			releaseDate, err := time.Parse("2006-01-02", rg.FirstReleaseDate)
			if err != nil || releaseDate.Before(cutoff) {
				continue
			}
			inserted, err := c.insert(alertRepo, model.ReleaseAlert{
				ID:          model.ReleaseSourceMusicBrainz + ":" + rg.ID,
				ArtistID:    artist.ID,
				ArtistName:  artist.Name,
				Source:      model.ReleaseSourceMusicBrainz,
				Title:       rg.Title,
				ReleaseType: rg.ReleaseType(),
				ReleaseDate: rg.FirstReleaseDate,
				ExternalURL: "https://musicbrainz.org/release-group/" + rg.ID,
			})
			if err != nil {
				log.Error(ctx, "Error saving release alert", "artist", artist.Name, "title", rg.Title, err)
				continue
			}
			if inserted {
				newAlerts++
			}
		}
		_ = releaseRepo.PutArtistInfo(&model.ExternalArtistInfo{
			ArtistID:         artist.ID,
			Source:           alertSource,
			ExternalArtistID: mbid,
			LastFetchedAt:    time.Now(),
			FetchStatus:      status,
		})
	}
	return newAlerts, nil
}

// checkMetalArchivesUpcoming scrapes MA's upcoming releases page (one page
// per run, cheap) and matches band names against library artists.
func (c *AlertChecker) checkMetalArchivesUpcoming(ctx context.Context) (int, error) {
	upcoming, err := c.ma.UpcomingReleases(ctx)
	if err != nil {
		if errors.Is(err, metalarchives.ErrBlocked) {
			log.Warn(ctx, "Metal Archives blocked the upcoming releases request")
		}
		return 0, err
	}
	artists, err := c.ds.Artist(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	byName := map[string]*model.Artist{}
	for i := range artists {
		byName[normalizeTitle(artists[i].Name)] = &artists[i]
	}

	alertRepo := c.ds.ReleaseAlert(ctx)
	newAlerts := 0
	for _, release := range upcoming {
		artist := byName[normalizeTitle(release.Band)]
		if artist == nil {
			continue
		}
		inserted, err := c.insert(alertRepo, model.ReleaseAlert{
			ID:          model.ReleaseSourceMetalArchives + ":" + release.ID,
			ArtistID:    artist.ID,
			ArtistName:  artist.Name,
			Source:      model.ReleaseSourceMetalArchives,
			Title:       release.Title,
			ReleaseType: release.Type,
			ReleaseDate: release.Date,
			ExternalURL: release.URL,
		})
		if err != nil {
			log.Error(ctx, "Error saving release alert", "artist", artist.Name, "title", release.Title, err)
			continue
		}
		if inserted {
			newAlerts++
		}
	}
	return newAlerts, nil
}

// artistMbid returns the artist's MusicBrainz id from file tags, or one
// resolved earlier by the discography name search.
func artistMbid(artist *model.Artist, releaseRepo model.ExternalReleaseRepository) string {
	if artist.MbzArtistID != "" {
		return artist.MbzArtistID
	}
	if resolved, err := releaseRepo.GetArtistInfo(artist.ID, model.ReleaseSourceMusicBrainz); err == nil {
		return resolved.ExternalArtistID
	}
	return ""
}

func alertKey(artistID, title string) string {
	return artistID + "|" + normalizeTitle(title)
}

// loadExisting indexes the alerts already stored, by artist and title.
func (c *AlertChecker) loadExisting(ctx context.Context) error {
	alerts, err := c.ds.ReleaseAlert(ctx).GetAll(math.MaxInt32)
	if err != nil {
		return err
	}
	c.existing = map[string]map[string]bool{}
	for _, alert := range alerts {
		c.remember(alert)
	}
	return nil
}

func (c *AlertChecker) remember(alert model.ReleaseAlert) {
	if c.existing == nil {
		c.existing = map[string]map[string]bool{}
	}
	key := alertKey(alert.ArtistID, alert.Title)
	if c.existing[key] == nil {
		c.existing[key] = map[string]bool{}
	}
	c.existing[key][alert.Source] = true
}

// insert stores an alert unless the same artist already has an alert with
// the same (normalized) title from a different source. Same-source
// duplicates are told apart by their ids as before.
func (c *AlertChecker) insert(repo model.ReleaseAlertRepository, alert model.ReleaseAlert) (bool, error) {
	for source := range c.existing[alertKey(alert.ArtistID, alert.Title)] {
		if source != alert.Source {
			return false, nil
		}
	}
	inserted, err := repo.InsertIfMissing(alert)
	if err != nil {
		return false, err
	}
	c.remember(alert)
	return inserted, nil
}

// checkBandcamp reads the Bandcamp page linked from each artist's
// MusicBrainz entry, and alerts on grid items released within the window.
// Items already seen are cached in external_release (source "bandcamp"), so
// each artist costs one request per run plus one per newly listed item.
func (c *AlertChecker) checkBandcamp(ctx context.Context) (int, error) {
	artists, err := c.ds.Artist(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	releaseRepo := c.ds.ExternalRelease(ctx)
	alertRepo := c.ds.ReleaseAlert(ctx)
	cutoff := time.Now().Add(-newReleaseWindow).Format("2006-01-02")
	newAlerts := 0

	for i := range artists {
		if ctx.Err() != nil {
			return newAlerts, ctx.Err()
		}
		artist := &artists[i]
		info, err := releaseRepo.GetArtistInfo(artist.ID, model.ReleaseSourceBandcamp)
		if err != nil {
			info = &model.ExternalArtistInfo{ArtistID: artist.ID, Source: model.ReleaseSourceBandcamp}
		} else {
			wait := alertCheckInterval
			if info.FetchStatus == "no_link" {
				wait = bandcampNoLinkRetry
			}
			if time.Since(info.LastFetchedAt) < wait {
				continue
			}
		}

		if info.ExternalArtistID == "" {
			mbid := artistMbid(artist, releaseRepo)
			if mbid == "" {
				continue
			}
			link, err := c.mbz.ArtistBandcampURL(ctx, mbid)
			if err != nil {
				log.Debug(ctx, "Release check: MusicBrainz link lookup failed", "artist", artist.Name, err)
				continue
			}
			info.LastFetchedAt = time.Now()
			if link == "" {
				info.FetchStatus = "no_link"
				_ = releaseRepo.PutArtistInfo(info)
				continue
			}
			info.ExternalArtistID = link
		}

		n, err := c.checkBandcampArtist(ctx, artist, info.ExternalArtistID, releaseRepo, alertRepo, cutoff)
		info.LastFetchedAt = time.Now()
		info.FetchStatus = "ok"
		if err != nil {
			log.Debug(ctx, "Release check: Bandcamp fetch failed", "artist", artist.Name, "url", info.ExternalArtistID, err)
			info.FetchStatus = "error"
		}
		_ = releaseRepo.PutArtistInfo(info)
		newAlerts += n
	}
	return newAlerts, nil
}

func (c *AlertChecker) checkBandcampArtist(ctx context.Context, artist *model.Artist, artistURL string,
	releaseRepo model.ExternalReleaseRepository, alertRepo model.ReleaseAlertRepository, cutoff string) (int, error) {
	items, err := c.bc.Discography(ctx, artistURL)
	if err != nil {
		return 0, err
	}
	cached, err := releaseRepo.GetByArtist(artist.ID)
	if err != nil {
		return 0, err
	}
	known := map[string]model.ExternalRelease{}
	for _, r := range cached {
		if r.Source == model.ReleaseSourceBandcamp {
			known[r.ExternalID] = r
		}
	}
	firstRun := len(known) == 0

	newAlerts, opened := 0, 0
	keep := make([]model.ExternalRelease, 0, len(items))
	for i, item := range items {
		if r, ok := known[item.Key]; ok {
			keep = append(keep, r)
			continue
		}
		entry := model.ExternalRelease{
			ID:          model.ReleaseSourceBandcamp + ":" + item.Key,
			ArtistID:    artist.ID,
			Source:      model.ReleaseSourceBandcamp,
			ExternalID:  item.Key,
			Title:       item.Title,
			ExternalURL: item.URL,
		}
		if firstRun && i >= bandcampFirstRunOpens {
			// back catalog: remember it without opening the page
			keep = append(keep, entry)
			continue
		}
		if opened >= bandcampMaxOpens {
			// not cached, so it is picked up on a later run
			continue
		}
		opened++
		release, err := c.bc.Release(ctx, item.URL)
		if err != nil {
			log.Debug(ctx, "Release check: Bandcamp release page failed", "artist", artist.Name, "url", item.URL, err)
			continue
		}
		if release.Title != "" {
			entry.Title = release.Title
		}
		entry.ReleaseDate = release.ReleaseDate
		entry.ReleaseType = release.ReleaseType
		if len(release.ReleaseDate) >= 4 {
			entry.Year, _ = strconv.Atoi(release.ReleaseDate[:4])
		}
		keep = append(keep, entry)

		if release.ReleaseDate < cutoff {
			continue
		}
		artID := item.ArtID
		if artID == "" {
			artID = release.ArtID
		}
		alertID := entry.ID
		if artID != "" {
			// the UI builds the cover image URL from the art id
			alertID += "@" + artID
		}
		inserted, err := c.insert(alertRepo, model.ReleaseAlert{
			ID:          alertID,
			ArtistID:    artist.ID,
			ArtistName:  artist.Name,
			Source:      model.ReleaseSourceBandcamp,
			Title:       entry.Title,
			ReleaseType: entry.ReleaseType,
			ReleaseDate: entry.ReleaseDate,
			ExternalURL: item.URL,
		})
		if err != nil {
			log.Error(ctx, "Error saving release alert", "artist", artist.Name, "title", entry.Title, err)
			continue
		}
		if inserted {
			newAlerts++
		}
	}
	if err := releaseRepo.ReplaceArtistReleases(artist.ID, model.ReleaseSourceBandcamp, keep); err != nil {
		return newAlerts, err
	}
	return newAlerts, nil
}
