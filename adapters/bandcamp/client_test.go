package bandcamp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Fixtures mirror the structure of bandcamp.com pages as of 2026-11.

const musicFixture = `<!DOCTYPE html><html><body>
<ol id="music-grid" class="editable-grid music-grid columns-4 public"
    data-client-items="[{&quot;art_id&quot;:1377529713,&quot;band_id&quot;:1,&quot;id&quot;:3794688303,&quot;page_url&quot;:&quot;/album/pieces-of-eight&quot;,&quot;title&quot;:&quot;Pieces Of Eight&quot;,&quot;type&quot;:&quot;album&quot;},{&quot;art_id&quot;:42,&quot;band_id&quot;:1,&quot;id&quot;:777,&quot;page_url&quot;:&quot;/track/a-single&quot;,&quot;title&quot;:&quot;A Single&quot;,&quot;type&quot;:&quot;track&quot;}]">
<li data-item-id="album-2832672141" data-band-id="1" class="music-grid-item square first-four">
    <a href="/album/it-s-raining-in-midgar">
        <div class="art"><img src="https://f4.bcbits.com/img/a1114338617_2.jpg" alt=""></div>
        <p class="title">
            it’s raining in midgar
        </p>
    </a>
</li>
<li data-item-id="album-111" data-band-id="1" class="music-grid-item square">
    <a href="https://label.bandcamp.com/album/tranquil-days?from=x">
        <div class="art"><img class="lazy" src="/img/0.gif" data-original="https://f4.bcbits.com/img/a999_2.jpg" alt=""></div>
        <p class="title">
            Tranquil Days
            <span class="artist-override">TPR &amp; Roxane Genot</span>
        </p>
    </a>
</li>
</ol>
</body></html>`

const albumFixture = `<!DOCTYPE html><html><head>
<meta name="bc-page-properties" content="{&quot;item_type&quot;:&quot;a&quot;,&quot;item_id&quot;:2832672141,&quot;tralbum_page_version&quot;:0}">
<script type="application/ld+json">
{"@type":"MusicAlbum","@id":"https://tprpiano.com/album/it-s-raining-in-midgar","name":"it’s raining in midgar","datePublished":"20 Nov 2026 00:00:00 GMT","numTracks":25,"image":"https://f4.bcbits.com/img/a1114338617_10.jpg","byArtist":{"@type":"MusicGroup","name":"TPR"}}
</script></head><body></body></html>`

const trackFixture = `<!DOCTYPE html><html><head>
<meta name="bc-page-properties" content="{&quot;item_type&quot;:&quot;t&quot;,&quot;item_id&quot;:777,&quot;tralbum_page_version&quot;:0}">
<script type="application/ld+json">
{"@type":"MusicRecording","name":"A Single","datePublished":"05 Jan 2026 00:00:00 GMT","image":"https://f4.bcbits.com/img/a42_10.jpg"}
</script></head><body></body></html>`

func server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/music", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(musicFixture)) })
	mux.HandleFunc("/album/it-s-raining-in-midgar", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(albumFixture)) })
	mux.HandleFunc("/track/a-single", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(trackFixture)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestDiscography(t *testing.T) {
	srv := server(t)
	items, err := NewClient().Discography(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	want := []Item{
		{Key: "album-2832672141", Title: "it’s raining in midgar", URL: srv.URL + "/album/it-s-raining-in-midgar", ArtID: "1114338617"},
		{Key: "album-111", Title: "Tranquil Days", URL: "https://label.bandcamp.com/album/tranquil-days", ArtID: "999"},
		{Key: "album-3794688303", Title: "Pieces Of Eight", URL: srv.URL + "/album/pieces-of-eight", ArtID: "1377529713"},
		{Key: "track-777", Title: "A Single", URL: srv.URL + "/track/a-single", ArtID: "42"},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items: %+v", len(items), items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, items[i], want[i])
		}
	}
}

func TestDiscographySingleReleaseRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/music", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/album/it-s-raining-in-midgar", http.StatusFound)
	})
	mux.HandleFunc("/album/it-s-raining-in-midgar", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(albumFixture)) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	items, err := NewClient().Discography(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Key != "album-2832672141" || items[0].ArtID != "1114338617" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestRelease(t *testing.T) {
	srv := server(t)
	c := NewClient()
	album, err := c.Release(context.Background(), srv.URL+"/album/it-s-raining-in-midgar")
	if err != nil {
		t.Fatal(err)
	}
	if album.ReleaseDate != "2026-11-20" || album.ReleaseType != "Full-length" || album.Title != "it’s raining in midgar" || album.Key != "album-2832672141" {
		t.Errorf("unexpected album: %+v", album)
	}
	track, err := c.Release(context.Background(), srv.URL+"/track/a-single")
	if err != nil {
		t.Fatal(err)
	}
	if track.ReleaseDate != "2026-01-05" || track.ReleaseType != "Single" || track.Key != "track-777" || track.ArtID != "42" {
		t.Errorf("unexpected track: %+v", track)
	}
}
