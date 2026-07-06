// Package musicbrainz provides a minimal MusicBrainz API client used to
// fetch artist discographies (release groups) for the missing-albums and
// release-alerts features. It is not an agents.Interface implementation:
// the agents API has no notion of full discographies.
package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

const pageSize = 100

// rate limiter shared by all requests: MusicBrainz allows 1 req/s
var (
	rateMu   sync.Mutex
	lastCall time.Time
)

func rateLimit(interval time.Duration) {
	rateMu.Lock()
	defer rateMu.Unlock()
	if wait := interval - time.Since(lastCall); wait > 0 {
		time.Sleep(wait)
	}
	lastCall = time.Now()
}

type ReleaseGroup struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	PrimaryType      string   `json:"primary-type"`
	SecondaryTypes   []string `json:"secondary-types"`
	FirstReleaseDate string   `json:"first-release-date"`
}

type releaseGroupsResponse struct {
	Count         int            `json:"release-group-count"`
	ReleaseGroups []ReleaseGroup `json:"release-groups"`
}

type Client struct {
	baseURL string
	hc      *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://musicbrainz.org"
	}
	return &Client{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: 15 * time.Second},
	}
}

// ReleaseGroupsByArtist returns all release groups credited to the given
// MusicBrainz artist id, paging through the browse API.
func (c *Client) ReleaseGroupsByArtist(ctx context.Context, mbid string) ([]ReleaseGroup, error) {
	var all []ReleaseGroup
	for offset := 0; ; offset += pageSize {
		page, total, err := c.releaseGroupsPage(ctx, mbid, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(all) >= total || len(page) == 0 {
			break
		}
	}
	return all, nil
}

func (c *Client) releaseGroupsPage(ctx context.Context, mbid string, offset int) ([]ReleaseGroup, int, error) {
	rateLimit(time.Second)
	params := url.Values{}
	params.Set("artist", mbid)
	params.Set("limit", fmt.Sprint(pageSize))
	params.Set("offset", fmt.Sprint(offset))
	params.Set("fmt", "json")
	reqURL := fmt.Sprintf("%s/ws/2/release-group?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("NavidromeMetalEdition/%s (https://github.com/navidrome/navidrome)", consts.Version))
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Warn(ctx, "MusicBrainz request failed", "url", reqURL, "status", resp.StatusCode)
		return nil, 0, fmt.Errorf("musicbrainz: unexpected status %d", resp.StatusCode)
	}
	var parsed releaseGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, 0, err
	}
	return parsed.ReleaseGroups, parsed.Count, nil
}

// ReleaseType maps a release group's types to a single normalized label.
// Secondary types win over the primary type because e.g. a live album has
// primary type "Album" + secondary "Live".
func (rg ReleaseGroup) ReleaseType() string {
	for _, secondary := range rg.SecondaryTypes {
		switch secondary {
		case "Live":
			return "Live"
		case "Compilation":
			return "Compilation"
		case "Demo":
			return "Demo"
		case "Remix":
			return "Remix"
		case "Soundtrack":
			return "Soundtrack"
		}
	}
	switch rg.PrimaryType {
	case "Album":
		return "Full-length"
	case "EP":
		return "EP"
	case "Single":
		return "Single"
	case "Broadcast":
		return "Broadcast"
	case "Other", "":
		return "Other"
	}
	return rg.PrimaryType
}

// Year extracts the release year from the first release date ("2006-03-27").
func (rg ReleaseGroup) Year() int {
	if len(rg.FirstReleaseDate) < 4 {
		return 0
	}
	var year int
	_, _ = fmt.Sscanf(rg.FirstReleaseDate[:4], "%d", &year)
	return year
}
