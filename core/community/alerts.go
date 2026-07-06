package community

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
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
)

// AlertChecker is the periodic job that looks for new/upcoming releases by
// library artists and records them as release alerts.
type AlertChecker struct {
	ds     model.DataStore
	broker events.Broker
	mbz    *musicbrainz.Client
	ma     *metalarchives.Client
}

func NewAlertChecker(ds model.DataStore, broker events.Broker) *AlertChecker {
	c := &AlertChecker{ds: ds, broker: broker}
	if conf.Server.MusicBrainz.Enabled {
		c.mbz = musicbrainz.NewClient(conf.Server.MusicBrainz.BaseURL)
	}
	if conf.Server.MetalArchives.Enabled {
		c.ma = metalarchives.NewClient(conf.Server.MetalArchives.BaseURL)
	}
	return c
}

// Check runs one pass over all library artists and returns how many new
// alerts were created.
func (c *AlertChecker) Check(ctx context.Context) (int, error) {
	newAlerts := 0
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
	if newAlerts > 0 {
		c.broker.SendBroadcastMessage(ctx, &events.ReleaseAlert{Count: newAlerts})
		log.Info(ctx, "New release alerts", "count", newAlerts)
	}
	return newAlerts, nil
}

func (c *AlertChecker) checkMusicBrainz(ctx context.Context) (int, error) {
	artists, err := c.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.NotEq{"artist.mbz_artist_id": ""},
			squirrel.NotEq{"artist.mbz_artist_id": nil},
		},
	})
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
		info, err := releaseRepo.GetArtistInfo(artist.ID, alertSource)
		if err == nil && time.Since(info.LastFetchedAt) < alertCheckInterval {
			continue
		}
		groups, err := c.mbz.ReleaseGroupsByArtist(ctx, artist.MbzArtistID)
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
			inserted, err := alertRepo.InsertIfMissing(model.ReleaseAlert{
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
			ExternalArtistID: artist.MbzArtistID,
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
		inserted, err := alertRepo.InsertIfMissing(model.ReleaseAlert{
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
