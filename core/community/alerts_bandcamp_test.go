package community

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/navidrome/navidrome/adapters/bandcamp"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
)

func bandcampServer(t *testing.T, opens *int32) *httptest.Server {
	t.Helper()
	recent := time.Now().Add(-24 * time.Hour).UTC().Format("02 Jan 2006 15:04:05 MST")
	page := func(id, kind, title, date string) string {
		return fmt.Sprintf(`<html><head><meta name="bc-page-properties" content='{"item_type":"%s","item_id":%s}'>
<script type="application/ld+json">{"@type":"MusicAlbum","name":"%s","datePublished":"%s","numTracks":10}</script></head></html>`, kind, id, title, date)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/music", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body><ol id="music-grid"
data-client-items="[{&quot;art_id&quot;:5,&quot;id&quot;:5,&quot;page_url&quot;:&quot;/album/old-5&quot;,&quot;title&quot;:&quot;Old 5&quot;,&quot;type&quot;:&quot;album&quot;}]">
<li data-item-id="album-1"><a href="/album/midgar"><img src="https://f4.bcbits.com/img/a77_2.jpg"><p class="title">it’s raining in midgar</p></a></li>
<li data-item-id="album-2"><a href="/album/old-2"><p class="title">Old 2</p></a></li>
<li data-item-id="album-3"><a href="/album/old-3"><p class="title">Old 3</p></a></li>
<li data-item-id="album-4"><a href="/album/old-4"><p class="title">Old 4</p></a></li>
</ol></body></html>`)
	})
	serve := func(path, body string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(opens, 1)
			_, _ = fmt.Fprint(w, body)
		})
	}
	serve("/album/midgar", page("1", "a", "it’s raining in midgar", recent))
	serve("/album/old-2", page("2", "a", "Old 2", "01 Jan 2020 00:00:00 GMT"))
	serve("/album/old-3", page("3", "a", "Old 3", "01 Jan 2019 00:00:00 GMT"))
	serve("/album/old-4", page("4", "a", "Old 4", "01 Jan 2018 00:00:00 GMT"))
	serve("/album/old-5", page("5", "a", "Old 5", "01 Jan 2017 00:00:00 GMT"))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newBandcampChecker(srvURL string, alerts []model.ReleaseAlert) (*AlertChecker, *tests.MockReleaseAlertRepo, *tests.MockExternalReleaseRepo) {
	artistRepo := tests.CreateMockArtistRepo()
	artistRepo.SetData(model.Artists{{ID: "ar1", Name: "TPR"}})
	alertRepo := &tests.MockReleaseAlertRepo{Alerts: alerts}
	releaseRepo := &tests.MockExternalReleaseRepo{}
	// Bandcamp link already resolved from MusicBrainz
	_ = releaseRepo.PutArtistInfo(&model.ExternalArtistInfo{ArtistID: "ar1", Source: model.ReleaseSourceBandcamp, ExternalArtistID: srvURL})
	ds := &tests.MockDataStore{MockedArtist: artistRepo, MockedReleaseAlert: alertRepo, MockedExternalRelease: releaseRepo}
	return &AlertChecker{ds: ds, bc: bandcamp.NewClient()}, alertRepo, releaseRepo
}

func TestBandcampAlertsNewRelease(t *testing.T) {
	var opens int32
	srv := bandcampServer(t, &opens)
	c, alertRepo, releaseRepo := newBandcampChecker(srv.URL, nil)
	ctx := context.Background()

	if err := c.loadExisting(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := c.checkBandcamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(alertRepo.Alerts) != 1 {
		t.Fatalf("want 1 alert, got %d: %+v", n, alertRepo.Alerts)
	}
	a := alertRepo.Alerts[0]
	if a.ID != "bandcamp:album-1@77" || a.Source != model.ReleaseSourceBandcamp || a.ExternalURL != srv.URL+"/album/midgar" {
		t.Errorf("unexpected alert: %+v", a)
	}
	// first run opens only the first few grid items; the rest are cached as seen
	if opens != bandcampFirstRunOpens {
		t.Errorf("first run opened %d pages, want %d", opens, bandcampFirstRunOpens)
	}
	if got := len(releaseRepo.Releases["ar1"]); got != 5 {
		t.Errorf("cached %d items, want 5", got)
	}

	// next run: nothing new in the grid, so no release pages are opened
	releaseRepo.ArtistInfos["ar1:"+model.ReleaseSourceBandcamp].LastFetchedAt = time.Time{}
	opens = 0
	n, err = c.checkBandcamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || opens != 0 {
		t.Errorf("second run: %d new alerts, %d pages opened; want 0, 0", n, opens)
	}
}

func TestBandcampSkipsReleaseAlreadyAlertedByOtherSource(t *testing.T) {
	var opens int32
	srv := bandcampServer(t, &opens)
	existing := []model.ReleaseAlert{{
		ID: "musicbrainz:rg1", ArtistID: "ar1", ArtistName: "TPR", Source: model.ReleaseSourceMusicBrainz,
		Title: "It's Raining in Midgar", ReleaseDate: time.Now().Format("2006-01-02"),
	}}
	c, alertRepo, _ := newBandcampChecker(srv.URL, existing)
	ctx := context.Background()
	if err := c.loadExisting(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := c.checkBandcamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || len(alertRepo.Alerts) != 1 {
		t.Fatalf("want no duplicate, got %d new: %+v", n, alertRepo.Alerts)
	}

	// and the reverse: a MusicBrainz alert for a title Bandcamp already listed
	alertRepo.Alerts = nil
	c.existing = nil
	c.remember(model.ReleaseAlert{ArtistID: "ar1", Title: "it’s raining in midgar", Source: model.ReleaseSourceBandcamp})
	inserted, err := c.insert(alertRepo, existing[0])
	if err != nil || inserted {
		t.Errorf("MusicBrainz duplicate inserted=%v err=%v", inserted, err)
	}
}
