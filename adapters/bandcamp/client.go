// Package bandcamp scrapes an artist's Bandcamp page for new releases.
// Bandcamp has no public API, so this reads the /music discography grid and
// the schema.org JSON-LD block of album/track pages. Requests are throttled
// to one every two seconds and callers must tolerate failure at any time:
// the HTML can change without notice.
package bandcamp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const requestInterval = 2 * time.Second

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

var ErrNoReleaseDate = errors.New("bandcamp: no release date on page")

var (
	rateMu   sync.Mutex
	lastCall time.Time
)

func rateLimit() {
	rateMu.Lock()
	defer rateMu.Unlock()
	if wait := requestInterval - time.Since(lastCall); wait > 0 {
		time.Sleep(wait)
	}
	lastCall = time.Now()
}

// Item is one entry of an artist's discography grid.
type Item struct {
	Key   string // "album-123" / "track-456", stable Bandcamp item id
	Title string
	URL   string // absolute album/track page URL
	ArtID string // cover art id, for https://f4.bcbits.com/img/a<ArtID>_2.jpg
}

// Release is the detail read from an album or track page.
type Release struct {
	Key         string
	Title       string
	ReleaseDate string // YYYY-MM-DD
	ReleaseType string
	ArtID       string
	URL         string
}

type Client struct {
	hc *http.Client
}

func NewClient() *Client {
	return &Client{hc: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) get(ctx context.Context, reqURL string) (*http.Response, error) {
	rateLimit()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bandcamp: %s returned status %d", reqURL, resp.StatusCode)
	}
	return resp, nil
}

// Discography lists the items on an artist's /music page, in the order the
// artist arranged them (usually newest first). An artist with a single
// release gets redirected straight to that release's page; that case is
// returned as a one-item list.
func (c *Client) Discography(ctx context.Context, artistURL string) ([]Item, error) {
	base, err := url.Parse(strings.TrimRight(artistURL, "/") + "/music")
	if err != nil {
		return nil, err
	}
	resp, err := c.get(ctx, base.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	final := resp.Request.URL
	if strings.HasPrefix(final.Path, "/album/") || strings.HasPrefix(final.Path, "/track/") {
		release, err := parseReleasePage(body, final.String())
		if err != nil {
			return nil, err
		}
		return []Item{{Key: release.Key, Title: release.Title, URL: release.URL, ArtID: release.ArtID}}, nil
	}
	return parseMusicPage(body, final)
}

// Release opens an album or track page and reads its release date.
func (c *Client) Release(ctx context.Context, pageURL string) (*Release, error) {
	resp, err := c.get(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return parseReleasePage(body, resp.Request.URL.String())
}

var artIDPattern = regexp.MustCompile(`/img/a(\d+)_`)

type clientItem struct {
	ArtID   json.Number `json:"art_id"`
	ID      json.Number `json:"id"`
	PageURL string      `json:"page_url"`
	Title   string      `json:"title"`
	Type    string      `json:"type"`
}

func parseMusicPage(body []byte, base *url.URL) ([]Item, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	grid := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "ol" && attr(n, "id") == "music-grid"
	})
	if grid == nil {
		return nil, errors.New("bandcamp: music grid not found")
	}

	var items []Item
	seen := map[string]bool{}
	add := func(item Item) {
		if item.Key == "" || item.URL == "" || seen[item.Key] {
			return
		}
		seen[item.Key] = true
		items = append(items, item)
	}

	// Items rendered as HTML (the first page of the grid)
	for li := grid.FirstChild; li != nil; li = li.NextSibling {
		if li.Type != html.ElementNode || li.Data != "li" {
			continue
		}
		item := Item{Key: attr(li, "data-item-id")}
		if a := findNode(li, isElement("a")); a != nil {
			item.URL = resolve(base, attr(a, "href"))
		}
		if p := findNode(li, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "p" && hasClass(n, "title")
		}); p != nil {
			item.Title = ownText(p)
		}
		if img := findNode(li, isElement("img")); img != nil {
			src := attr(img, "data-original")
			if src == "" {
				src = attr(img, "src")
			}
			if m := artIDPattern.FindStringSubmatch(src); m != nil {
				item.ArtID = m[1]
			}
		}
		add(item)
	}

	// The rest of the grid is lazy-loaded from a JSON attribute
	if raw := attr(grid, "data-client-items"); raw != "" {
		var extra []clientItem
		if err := json.Unmarshal([]byte(raw), &extra); err == nil {
			for _, ci := range extra {
				add(Item{
					Key:   ci.Type + "-" + ci.ID.String(),
					Title: strings.TrimSpace(ci.Title),
					URL:   resolve(base, ci.PageURL),
					ArtID: ci.ArtID.String(),
				})
			}
		}
	}
	return items, nil
}

type jsonLD struct {
	Type          string `json:"@type"`
	Name          string `json:"name"`
	DatePublished string `json:"datePublished"`
	NumTracks     int    `json:"numTracks"`
	Image         any    `json:"image"`
}

var pageItemPattern = regexp.MustCompile(`"item_type"\s*:\s*"([at])"\s*,\s*"item_id"\s*:\s*(\d+)`)

func parseReleasePage(body []byte, pageURL string) (*Release, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	script := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "script" && attr(n, "type") == "application/ld+json"
	})
	if script == nil || script.FirstChild == nil {
		return nil, ErrNoReleaseDate
	}
	var ld jsonLD
	if err := json.Unmarshal([]byte(script.FirstChild.Data), &ld); err != nil {
		return nil, fmt.Errorf("bandcamp: bad JSON-LD: %w", err)
	}
	date, err := parseDate(ld.DatePublished)
	if err != nil {
		return nil, ErrNoReleaseDate
	}
	release := &Release{
		Title:       strings.TrimSpace(ld.Name),
		ReleaseDate: date,
		URL:         pageURL,
	}
	switch {
	case ld.Type == "MusicRecording", ld.NumTracks == 1:
		release.ReleaseType = "Single"
	case ld.NumTracks > 1 && ld.NumTracks <= 6:
		release.ReleaseType = "EP"
	default:
		release.ReleaseType = "Full-length"
	}
	if img, ok := ld.Image.(string); ok {
		if m := artIDPattern.FindStringSubmatch(img); m != nil {
			release.ArtID = m[1]
		}
	}
	// bc-page-properties carries the numeric item id used by the grid
	if meta := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && attr(n, "name") == "bc-page-properties"
	}); meta != nil {
		if m := pageItemPattern.FindStringSubmatch(attr(meta, "content")); m != nil {
			kind := "album"
			if m[1] == "t" {
				kind = "track"
			}
			release.Key = kind + "-" + m[2]
		}
	}
	return release, nil
}

// parseDate converts Bandcamp's "20 Nov 2026 00:00:00 GMT" to YYYY-MM-DD.
func parseDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"02 Jan 2006 15:04:05 MST", "2 Jan 2006 15:04:05 MST", time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("bandcamp: unknown date format %q", s)
}

func resolve(base *url.URL, ref string) string {
	if ref == "" {
		return ""
	}
	u, err := base.Parse(ref)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func isElement(tag string) func(*html.Node) bool {
	return func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == tag }
}

func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, match); found != nil {
			return found
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

// ownText returns the direct text of a node, skipping child elements such as
// the "artist-override" span on split/featured releases.
func ownText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}
