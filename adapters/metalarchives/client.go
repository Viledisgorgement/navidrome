// Package metalarchives scrapes Encyclopaedia Metallum (metal-archives.com),
// which has no official API. It is opt-in (ND_METALARCHIVES_ENABLED), uses a
// browser-like User-Agent, is rate-limited to one request per 3 seconds, and
// callers must tolerate failure at any time: the site blocks generic clients
// with 403s and its HTML can change without notice.
package metalarchives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
	"golang.org/x/net/html"
)

var (
	ErrBlocked  = errors.New("metalarchives: request blocked (403)")
	ErrNotFound = errors.New("metalarchives: band not found")
)

const requestInterval = 3 * time.Second

// Browser-like UA: metal-archives 403s obvious non-browser clients
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

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

type Release struct {
	ID    string
	Title string
	Type  string
	Year  int
	URL   string
}

type Client struct {
	baseURL string
	hc      *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://www.metal-archives.com"
	}
	return &Client{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: 20 * time.Second},
	}
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
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrBlocked
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("metalarchives: unexpected status %d", resp.StatusCode)
	}
	return resp, nil
}

type searchResponse struct {
	Error        string     `json:"error"`
	TotalRecords int        `json:"iTotalRecords"`
	Data         [][]string `json:"aaData"`
}

var bandLinkRe = regexp.MustCompile(`href="([^"]*/bands/[^/"]+/(\d+))"[^>]*>([^<]+)<`)

// SearchBand resolves a band name to a Metal Archives band id using the
// site's internal JSON search endpoint. Returns the id of the band whose
// name matches exactly (case-insensitive); ErrNotFound when there is no
// unambiguous match.
func (c *Client) SearchBand(ctx context.Context, name string) (string, error) {
	params := url.Values{}
	params.Set("field", "name")
	params.Set("query", name)
	resp, err := c.get(ctx, c.baseURL+"/search/ajax-band-search/?"+params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("metalarchives: parsing search response: %w", err)
	}
	var matches []string
	for _, row := range parsed.Data {
		if len(row) == 0 {
			continue
		}
		m := bandLinkRe.FindStringSubmatch(row[0])
		if m == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(m[3]), strings.TrimSpace(name)) {
			matches = append(matches, m[2])
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		// Ambiguous (multiple bands share the name). Picking one at random
		// risks caching a wrong discography, so give up.
		log.Debug(ctx, "Ambiguous Metal Archives band name", "name", name, "matches", len(matches))
		return "", ErrNotFound
	}
	return "", ErrNotFound
}

// Discography fetches and parses the band's "Complete discography" table.
func (c *Client) Discography(ctx context.Context, bandID string) ([]Release, error) {
	resp, err := c.get(ctx, fmt.Sprintf("%s/band/discography/id/%s/tab/all", c.baseURL, bandID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("metalarchives: parsing discography html: %w", err)
	}
	return parseDiscographyTable(doc), nil
}

var albumLinkRe = regexp.MustCompile(`/albums/[^/]+/[^/]+/(\d+)$`)

// parseDiscographyTable walks the discography HTML. Each row is expected to
// be: [album link, type, year, reviews]. Rows that don't fit are skipped.
func parseDiscographyTable(doc *html.Node) []Release {
	var releases []Release
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if release, ok := parseRow(n); ok {
				releases = append(releases, release)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return releases
}

func parseRow(tr *html.Node) (Release, bool) {
	var cells []*html.Node
	for cell := tr.FirstChild; cell != nil; cell = cell.NextSibling {
		if cell.Type == html.ElementNode && cell.Data == "td" {
			cells = append(cells, cell)
		}
	}
	if len(cells) < 3 {
		return Release{}, false
	}
	link := findLink(cells[0])
	title := strings.TrimSpace(textContent(cells[0]))
	relType := strings.TrimSpace(textContent(cells[1]))
	year, _ := strconv.Atoi(strings.TrimSpace(textContent(cells[2])))
	if title == "" || relType == "" {
		return Release{}, false
	}
	release := Release{
		Title: title,
		Type:  normalizeType(relType),
		Year:  year,
		URL:   link,
	}
	if m := albumLinkRe.FindStringSubmatch(link); m != nil {
		release.ID = m[1]
	} else {
		release.ID = fmt.Sprintf("%s-%d", strings.ToLower(strings.ReplaceAll(title, " ", "-")), year)
	}
	return release, true
}

func normalizeType(t string) string {
	if strings.EqualFold(t, "Live album") {
		return "Live"
	}
	return t
}

func findLink(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				return attr.Val
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if link := findLink(child); link != "" {
			return link
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		sb.WriteString(textContent(child))
	}
	return sb.String()
}
