package metalarchives

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Fixtures mirror the structure of metal-archives.com responses as of
// 2026-07. If the site changes, these tests keep passing but live scraping
// breaks — they only pin OUR parser against the last known format.

const searchFixture = `{
	"error": "",
	"iTotalRecords": 2,
	"iTotalDisplayRecords": 2,
	"sEcho": 0,
	"aaData": [
		[
			"<a href=\"https://www.metal-archives.com/bands/Bolt_Thrower/2717\" title=\"Bolt Thrower (GB)\">Bolt Thrower</a>  <!-- 14.797752 -->",
			"Death Metal",
			"United Kingdom"
		],
		[
			"<a href=\"https://www.metal-archives.com/bands/Bolt_Throwers/99999\">Bolt Throwers</a>  <!-- 9.03 -->",
			"Thrash Metal",
			"Germany"
		]
	]
}`

const discographyFixture = `<!DOCTYPE html>
<html><body>
<table class="display discog" cellpadding="0" cellspacing="0">
<thead><tr><th>Name</th><th>Type</th><th>Year</th><th>Reviews</th></tr></thead>
<tbody>
<tr>
	<td><a href="https://www.metal-archives.com/albums/Bolt_Thrower/In_Battle_There_Is_No_Law/2426" class="album">In Battle There Is No Law</a></td>
	<td>Full-length</td>
	<td>1988</td>
	<td><a href="#">8 (74%)</a></td>
</tr>
<tr>
	<td><a href="https://www.metal-archives.com/albums/Bolt_Thrower/Concession_of_Pain/68274" class="demo">Concession of Pain</a></td>
	<td>Demo</td>
	<td>1987</td>
	<td>&nbsp;</td>
</tr>
<tr>
	<td><a href="https://www.metal-archives.com/albums/Bolt_Thrower/Live_War/12345" class="other">Live War</a></td>
	<td>Live album</td>
	<td>1994</td>
	<td>&nbsp;</td>
</tr>
</tbody>
</table>
</body></html>`

func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/search/ajax-band-search/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(searchFixture))
	})
	mux.HandleFunc("/band/discography/id/2717/tab/all", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(discographyFixture))
	})
	mux.HandleFunc("/blocked/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestSearchBandExactMatch(t *testing.T) {
	server := fixtureServer(t)
	client := NewClient(server.URL)
	id, err := client.SearchBand(context.Background(), "Bolt Thrower")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "2717" {
		t.Errorf("expected band id 2717, got %q", id)
	}
}

func TestSearchBandNotFound(t *testing.T) {
	server := fixtureServer(t)
	client := NewClient(server.URL)
	_, err := client.SearchBand(context.Background(), "Nonexistent Band")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDiscographyParsing(t *testing.T) {
	server := fixtureServer(t)
	client := NewClient(server.URL)
	releases, err := client.Discography(context.Background(), "2717")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 3 {
		t.Fatalf("expected 3 releases, got %d", len(releases))
	}
	first := releases[0]
	if first.Title != "In Battle There Is No Law" || first.Type != "Full-length" ||
		first.Year != 1988 || first.ID != "2426" {
		t.Errorf("unexpected first release: %+v", first)
	}
	if releases[1].Type != "Demo" || releases[1].Year != 1987 {
		t.Errorf("unexpected second release: %+v", releases[1])
	}
	if releases[2].Type != "Live" {
		t.Errorf("expected Live album mapped to Live, got %q", releases[2].Type)
	}
}

func TestBlockedReturnsErrBlocked(t *testing.T) {
	server := fixtureServer(t)
	client := NewClient(server.URL + "/blocked")
	_, err := client.SearchBand(context.Background(), "Anything")
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}
