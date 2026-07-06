package metalarchives

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestLive hits the real metal-archives.com. It is skipped unless
// ND_MA_LIVE_TEST=1, so CI and normal test runs never touch the site.
// Run manually to check whether the scraper still works:
//
//	ND_MA_LIVE_TEST=1 go test ./adapters/metalarchives/ -run TestLive -v
func TestLive(t *testing.T) {
	if os.Getenv("ND_MA_LIVE_TEST") != "1" {
		t.Skip("set ND_MA_LIVE_TEST=1 to run against the live site")
	}
	client := NewClient("")
	ctx := context.Background()

	bandID, err := client.SearchBand(ctx, "Bolt Thrower")
	if err != nil {
		t.Fatalf("SearchBand failed: %v", err)
	}
	t.Logf("Bolt Thrower band id: %s", bandID)
	if bandID != "234" {
		t.Errorf("expected Bolt Thrower id 234, got %s", bandID)
	}

	releases, err := client.Discography(ctx, bandID)
	if err != nil {
		t.Fatalf("Discography failed: %v", err)
	}
	t.Logf("got %d releases", len(releases))
	if len(releases) < 10 {
		t.Errorf("expected at least 10 releases, got %d", len(releases))
	}
	found := false
	for _, release := range releases {
		t.Logf("  [%s] %s (%d) id=%s", release.Type, release.Title, release.Year, release.ID)
		if strings.Contains(release.Title, "Realm of Chaos") && release.Type == "Full-length" && release.Year == 1989 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Realm of Chaos (1989, Full-length)")
	}
}
