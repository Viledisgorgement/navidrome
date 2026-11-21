package community

import (
	"testing"

	"github.com/navidrome/navidrome/model"
)

func TestNormalizeTitle(t *testing.T) {
	cases := map[string]string{
		"Buried in the Snow: A Melancholy Tribute to Final Fantasy VII, VIII & IX":   "buried in the snow a melancholy tribute to final fantasy vii viii and ix",
		"Buried in the Snow - A Melancholy Tribute to Final Fantasy VII, VIII and IX": "buried in the snow a melancholy tribute to final fantasy vii viii and ix",
		"Slaughter of the Soul (Remastered)":                                          "slaughter of the soul",
		"Slaughter of the Soul [2014 Reissue]":                                        "slaughter of the soul",
		"Midnight Mementos (Fool’s Edition)":                                          "midnight mementos",
		"Don’t Stop":                                                                  "dont stop",
		"Don't Stop":                                                                  "dont stop",
	}
	for in, want := range cases {
		if got := normalizeTitle(in); got != want {
			t.Errorf("normalizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAlbumMatcher(t *testing.T) {
	library := model.Albums{
		{ID: "buried", Name: "Buried in the Snow: A Melancholy Tribute to Final Fantasy VII VIII and IX", MaxYear: 2013},
		{ID: "fleeting", Name: "A Fleeting Dream: A Melancholy Tribute To Final Fantasy X", MaxYear: 2014},
		{ID: "frag2013", Name: "Fragments of Memories: A Melancholy Tribute to Final Fantasy VIII", MaxYear: 2013},
		{ID: "eight", Name: "Pieces of Eight: Melancholy Music from Octopath Traveler", MaxYear: 2018},
		{ID: "ff9", Name: "A Melancholy Tribute to Final Fantasy IX", MaxYear: 2012},
		{ID: "rg", Name: "Something Else Entirely", MbzReleaseGroupID: "rg-1"},
	}
	m := newAlbumMatcher(library)

	tests := []struct {
		rg, title string
		year      int
		want      string
	}{
		{"", "Buried in the Snow: A Melancholy Tribute to Final Fantasy VII, VIII & IX", 2013, "buried"},
		{"", "A Fleeting Dream: A Melancholy Tribute to Final Fantasy X", 2014, "fleeting"},
		{"", "Fragments of Memories: A Melancholy Tribute to Final Fantasy VIII", 2013, "frag2013"},
		{"rg-1", "Whatever MusicBrainz calls it", 2020, "rg"},
		// Different releases that must stay missing
		{"", "Fragments of Memories: Melancholy Music From Final Fantasy VIII (Definitive Edition)", 2017, ""},
		{"", "Pieces Of Night: Melancholy Music From Octopath Traveler II", 2023, ""},
		{"", "Memoria: A Melancholy Tribute to Final Fantasy IX", 2014, ""},
		{"", "Buried in the Snow: Something", 2019, ""},
	}
	for _, tt := range tests {
		got := ""
		if album := m.match(tt.rg, tt.title, tt.year); album != nil {
			got = album.ID
		}
		if got != tt.want {
			t.Errorf("match(%q, %q, %d) = %q, want %q", tt.rg, tt.title, tt.year, got, tt.want)
		}
	}
}
