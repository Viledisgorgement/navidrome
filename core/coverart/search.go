// Package coverart finds candidate album covers on public services and
// downloads a chosen one, for the "Change cover art" feature.
package coverart

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/adapters/musicbrainz"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	_ "golang.org/x/image/webp"
)

// Candidate is one cover found by a search
type Candidate struct {
	Source    string `json:"source"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Thumbnail string `json:"thumbnail"` // small image for the results grid
	URL       string `json:"url"`       // full-size image to save
}

// Base URLs, overridable in tests
var (
	deezerURL       = "https://api.deezer.com"
	itunesURL       = "https://itunes.apple.com"
	musicBrainzURL  = "https://musicbrainz.org"
	coverArtArchURL = "https://coverartarchive.org"
	lastFMURL       = "https://ws.audioscrobbler.com/2.0/"
)

const (
	perSourceLimit = 8
	searchTimeout  = 15 * time.Second
)

var httpClient = &http.Client{Timeout: searchTimeout}

type source struct {
	name   string
	search func(ctx context.Context, artist, album string) ([]Candidate, error)
}

func sources() []source {
	ss := []source{
		{"Deezer", searchDeezer},
		{"iTunes", searchITunes},
		{"MusicBrainz", searchMusicBrainz},
	}
	if conf.Server.LastFM.ApiKey != "" {
		ss = append(ss, source{"Last.fm", searchLastFM})
	}
	return ss
}

// Search queries all sources concurrently and returns their results grouped
// by source, in a fixed source order. Failing sources are logged and skipped.
func Search(ctx context.Context, artist, album string) []Candidate {
	ss := sources()
	results := make([][]Candidate, len(ss))
	var wg sync.WaitGroup
	for i, s := range ss {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(ctx, searchTimeout)
			defer cancel()
			res, err := s.search(ctx, artist, album)
			if err != nil {
				log.Warn(ctx, "Cover art search failed", "source", s.name, "artist", artist, "album", album, err)
				return
			}
			if len(res) > perSourceLimit {
				res = res[:perSourceLimit]
			}
			results[i] = res
		})
	}
	wg.Wait()
	var all []Candidate
	seen := map[string]bool{}
	for _, res := range results {
		for _, c := range res {
			if c.URL == "" || seen[c.URL] {
				continue
			}
			seen[c.URL] = true
			all = append(all, c)
		}
	}
	return all
}

func getJSON(ctx context.Context, reqURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", consts.HTTPUserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var parenRe = regexp.MustCompile(`\s*[(\[][^)\]]*[)\]]`)

// simplifyTitle drops bracketed parts like "(Remastered)" or a translated
// title, which often keep services from matching
func simplifyTitle(s string) string {
	return strings.TrimSpace(parenRe.ReplaceAllString(s, ""))
}

// searchDeezer tries a strict artist/album query first, then looser ones
func searchDeezer(ctx context.Context, artist, album string) ([]Candidate, error) {
	short := simplifyTitle(album)
	queries := []string{fmt.Sprintf(`album:"%s"`, album)}
	if artist != "" {
		queries[0] = fmt.Sprintf(`artist:"%s" %s`, artist, queries[0])
		if short != "" && short != album {
			queries = append(queries, fmt.Sprintf(`artist:"%s" album:"%s"`, artist, short))
		}
	}
	queries = append(queries, strings.TrimSpace(artist+" "+cmp.Or(short, album)))
	var err error
	for _, q := range queries {
		var res []Candidate
		if res, err = deezerQuery(ctx, q); err == nil && len(res) > 0 {
			return res, nil
		}
	}
	return nil, err
}

func deezerQuery(ctx context.Context, q string) ([]Candidate, error) {
	var res struct {
		Data []struct {
			Title   string `json:"title"`
			CoverMd string `json:"cover_medium"`
			CoverXL string `json:"cover_xl"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"data"`
	}
	params := url.Values{"q": {q}, "limit": {fmt.Sprint(perSourceLimit)}}
	if err := getJSON(ctx, deezerURL+"/search/album?"+params.Encode(), &res); err != nil {
		return nil, err
	}
	var out []Candidate
	for _, d := range res.Data {
		out = append(out, Candidate{Source: "Deezer", Artist: d.Artist.Name, Album: d.Title, Thumbnail: d.CoverMd, URL: d.CoverXL})
	}
	return out, nil
}

var itunesSizeRe = regexp.MustCompile(`/\d+x\d+bb\.`)

func searchITunes(ctx context.Context, artist, album string) ([]Candidate, error) {
	res, err := itunesQuery(ctx, artist+" "+album)
	if err == nil && len(res) == 0 {
		if short := simplifyTitle(album); short != "" && short != album {
			return itunesQuery(ctx, artist+" "+short)
		}
	}
	return res, err
}

func itunesQuery(ctx context.Context, term string) ([]Candidate, error) {
	var res struct {
		Results []struct {
			ArtistName     string `json:"artistName"`
			CollectionName string `json:"collectionName"`
			Artwork        string `json:"artworkUrl100"`
		} `json:"results"`
	}
	params := url.Values{
		"term":   {strings.TrimSpace(term)},
		"entity": {"album"},
		"limit":  {fmt.Sprint(perSourceLimit)},
	}
	if err := getJSON(ctx, itunesURL+"/search?"+params.Encode(), &res); err != nil {
		return nil, err
	}
	var out []Candidate
	for _, r := range res.Results {
		if r.Artwork == "" {
			continue
		}
		out = append(out, Candidate{
			Source:    "iTunes",
			Artist:    r.ArtistName,
			Album:     r.CollectionName,
			Thumbnail: itunesSizeRe.ReplaceAllString(r.Artwork, "/300x300bb."),
			URL:       itunesSizeRe.ReplaceAllString(r.Artwork, "/1200x1200bb."),
		})
	}
	return out, nil
}

// searchMusicBrainz matches release groups and points at their Cover Art
// Archive front image. Not every release group has art; the UI drops
// candidates whose thumbnail fails to load.
func searchMusicBrainz(ctx context.Context, artist, album string) ([]Candidate, error) {
	matches, err := musicbrainz.NewClient(musicBrainzURL).SearchReleaseGroups(ctx, artist, album, perSourceLimit)
	if err != nil {
		return nil, err
	}
	var out []Candidate
	for _, m := range matches {
		if m.Score < 70 {
			continue
		}
		base := fmt.Sprintf("%s/release-group/%s/front", coverArtArchURL, m.ID)
		out = append(out, Candidate{Source: "Cover Art Archive", Artist: m.ArtistName(), Album: m.Title, Thumbnail: base + "-250", URL: base + "-1200"})
	}
	return out, nil
}

var lastFMSizeRe = regexp.MustCompile(`/i/u/\d+x\d+/`)

func searchLastFM(ctx context.Context, artist, album string) ([]Candidate, error) {
	var res struct {
		Results struct {
			AlbumMatches struct {
				Album []struct {
					Name   string `json:"name"`
					Artist string `json:"artist"`
					Image  []struct {
						URL  string `json:"#text"`
						Size string `json:"size"`
					} `json:"image"`
				} `json:"album"`
			} `json:"albummatches"`
		} `json:"results"`
	}
	params := url.Values{
		"method":  {"album.search"},
		"album":   {strings.TrimSpace(artist + " " + album)},
		"api_key": {conf.Server.LastFM.ApiKey},
		"format":  {"json"},
		"limit":   {fmt.Sprint(perSourceLimit)},
	}
	if err := getJSON(ctx, lastFMURL+"?"+params.Encode(), &res); err != nil {
		return nil, err
	}
	var out []Candidate
	for _, a := range res.Results.AlbumMatches.Album {
		var img string
		for _, i := range a.Image {
			if i.URL != "" {
				img = i.URL // sizes are listed smallest to largest
			}
		}
		if img == "" {
			continue
		}
		out = append(out, Candidate{
			Source:    "Last.fm",
			Artist:    a.Artist,
			Album:     a.Name,
			Thumbnail: lastFMSizeRe.ReplaceAllString(img, "/i/u/300x300/"),
			URL:       lastFMSizeRe.ReplaceAllString(img, "/i/u/"), // original size
		})
	}
	return out, nil
}

// ErrNotImage is returned by Download when the URL does not serve an image
var ErrNotImage = errors.New("not a supported image")

// Download fetches an image from an http(s) URL, rejecting anything larger
// than maxSize or that does not decode as an image. Returns the bytes and the
// file extension for the image format (e.g. ".jpg").
func Download(ctx context.Context, imageURL string, maxSize int64) ([]byte, string, error) {
	u, err := url.Parse(imageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, "", fmt.Errorf("invalid image URL %q", imageURL)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", consts.HTTPUserAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("downloading image: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxSize {
		return nil, "", fmt.Errorf("image is larger than %d bytes", maxSize)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrNotImage
	}
	return data, ImageExt(format), nil
}

// ImageExt maps an image.DecodeConfig format name to a file extension
func ImageExt(format string) string {
	if format == "jpeg" {
		return ".jpg"
	}
	return "." + format
}
