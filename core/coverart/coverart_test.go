package coverart

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

var _ = Describe("coverart", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
		DeferCleanup(configtest.SetupConfig())
	})

	Describe("WriteFolderCover", func() {
		var dir string
		BeforeEach(func() {
			dir = GinkgoT().TempDir()
		})

		It("writes cover<ext> into the folder", func() {
			path, err := WriteFolderCover(dir, []byte("new"), ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(dir, "cover.jpg")))
			Expect(os.ReadFile(path)).To(Equal([]byte("new")))
		})

		It("backs up existing cover images, leaving other files alone", func() {
			Expect(os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("old"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "Cover.PNG"), []byte("old png"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "folder.jpg"), []byte("folder"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "01 song.mp3"), []byte("mp3"), 0600)).To(Succeed())

			_, err := WriteFolderCover(dir, []byte("new"), ".jpg")
			Expect(err).ToNot(HaveOccurred())

			Expect(os.ReadFile(filepath.Join(dir, "cover.jpg"))).To(Equal([]byte("new")))
			Expect(os.ReadFile(filepath.Join(dir, "cover.jpg.bak"))).To(Equal([]byte("old")))
			Expect(os.ReadFile(filepath.Join(dir, "Cover.PNG.bak"))).To(Equal([]byte("old png")))
			Expect(filepath.Join(dir, "folder.jpg")).To(BeAnExistingFile())
			Expect(filepath.Join(dir, "01 song.mp3")).To(BeAnExistingFile())
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				Expect(e.Name()).ToNot(HavePrefix(".cover-"))
			}
		})

		It("fails without touching anything when the folder is not writable", func() {
			if os.Geteuid() == 0 {
				Skip("root ignores directory permissions")
			}
			Expect(os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("old"), 0600)).To(Succeed())
			Expect(os.Chmod(dir, 0500)).To(Succeed())
			DeferCleanup(func() { _ = os.Chmod(dir, 0700) })

			_, err := WriteFolderCover(dir, []byte("new"), ".jpg")
			Expect(err).To(HaveOccurred())
			Expect(os.ReadFile(filepath.Join(dir, "cover.jpg"))).To(Equal([]byte("old")))
		})
	})

	Describe("Download", func() {
		var srv *httptest.Server
		BeforeEach(func() {
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/img.png":
					_, _ = w.Write(pngBytes())
				case "/page.html":
					_, _ = w.Write([]byte("<html></html>"))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			DeferCleanup(srv.Close)
		})

		It("returns the image and its extension", func() {
			data, ext, err := Download(ctx, srv.URL+"/img.png", 1<<20)
			Expect(err).ToNot(HaveOccurred())
			Expect(ext).To(Equal(".png"))
			Expect(data).To(Equal(pngBytes()))
		})
		It("rejects non-images", func() {
			_, _, err := Download(ctx, srv.URL+"/page.html", 1<<20)
			Expect(err).To(MatchError(ErrNotImage))
		})
		It("rejects images over the size limit", func() {
			_, _, err := Download(ctx, srv.URL+"/img.png", 10)
			Expect(err).To(MatchError(ContainSubstring("larger than")))
		})
		It("rejects non-http URLs", func() {
			_, _, err := Download(ctx, "file:///etc/passwd", 1<<20)
			Expect(err).To(MatchError(ContainSubstring("invalid image URL")))
		})
		It("fails on HTTP errors", func() {
			_, _, err := Download(ctx, srv.URL+"/missing.jpg", 1<<20)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Search", func() {
		BeforeEach(func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/deezer/search/album":
					Expect(r.URL.Query().Get("q")).To(Equal(`artist:"Zero 7" album:"In The Waiting Line"`))
					_, _ = w.Write([]byte(`{"data":[{"title":"In the Waiting Line","cover_medium":"https://dz/250.jpg","cover_xl":"https://dz/1000.jpg","artist":{"name":"Zero 7"}}]}`))
				case r.URL.Path == "/itunes/search":
					_, _ = w.Write([]byte(`{"results":[{"artistName":"Zero 7","collectionName":"In the Waiting Line - EP","artworkUrl100":"https://mz/thumb/100x100bb.jpg"},{"artistName":"x","collectionName":"no art"}]}`))
				case r.URL.Path == "/mb/ws/2/release-group":
					Expect(r.URL.Query().Get("query")).To(Equal(`releasegroup:"In The Waiting Line" AND artist:"Zero 7"`))
					_, _ = w.Write([]byte(`{"release-groups":[{"id":"rg1","title":"In the Waiting Line","score":100,"artist-credit":[{"name":"Zero 7"}]},{"id":"rg2","title":"Unrelated","score":20}]}`))
				case r.URL.Path == "/lastfm/":
					Expect(r.URL.Query().Get("api_key")).To(Equal("key"))
					_, _ = w.Write([]byte(`{"results":{"albummatches":{"album":[{"name":"In the Waiting Line","artist":"Zero 7","image":[{"#text":"https://lf/i/u/34s/a.png","size":"small"},{"#text":"https://lf/i/u/300x300/a.png","size":"extralarge"}]},{"name":"No image","artist":"Zero 7","image":[{"#text":"","size":"small"}]}]}}}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			DeferCleanup(srv.Close)
			oldDeezer, oldITunes, oldMB, oldLastFM := deezerURL, itunesURL, musicBrainzURL, lastFMURL
			DeferCleanup(func() { deezerURL, itunesURL, musicBrainzURL, lastFMURL = oldDeezer, oldITunes, oldMB, oldLastFM })
			deezerURL, itunesURL, musicBrainzURL, lastFMURL = srv.URL+"/deezer", srv.URL+"/itunes", srv.URL+"/mb", srv.URL+"/lastfm/"
		})

		It("merges results from all sources in order", func() {
			conf.Server.LastFM.ApiKey = "key"
			res := Search(ctx, "Zero 7", "In The Waiting Line")
			Expect(res).To(Equal([]Candidate{
				{Source: "Deezer", Artist: "Zero 7", Album: "In the Waiting Line", Thumbnail: "https://dz/250.jpg", URL: "https://dz/1000.jpg"},
				{Source: "iTunes", Artist: "Zero 7", Album: "In the Waiting Line - EP", Thumbnail: "https://mz/thumb/300x300bb.jpg", URL: "https://mz/thumb/1200x1200bb.jpg"},
				{Source: "Cover Art Archive", Artist: "Zero 7", Album: "In the Waiting Line", Thumbnail: "https://coverartarchive.org/release-group/rg1/front-250", URL: "https://coverartarchive.org/release-group/rg1/front-1200"},
				{Source: "Last.fm", Artist: "Zero 7", Album: "In the Waiting Line", Thumbnail: "https://lf/i/u/300x300/a.png", URL: "https://lf/i/u/a.png"},
			}))
		})

		It("retries Deezer and iTunes without bracketed title parts", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/deezer/search/album":
					if r.URL.Query().Get("q") == `artist:"Astrofaes" album:"Ancestors' Shadows"` {
						_, _ = w.Write([]byte(`{"data":[{"title":"Ancestors' Shadows","cover_medium":"m","cover_xl":"xl","artist":{"name":"Astrofaes"}}]}`))
						return
					}
					_, _ = w.Write([]byte(`{"data":[]}`))
				case "/itunes/search":
					if r.URL.Query().Get("term") == "Astrofaes Ancestors' Shadows" {
						_, _ = w.Write([]byte(`{"results":[{"artistName":"Astrofaes","collectionName":"Ancestors' Shadows","artworkUrl100":"https://mz/100x100bb.jpg"}]}`))
						return
					}
					_, _ = w.Write([]byte(`{"results":[]}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			DeferCleanup(srv.Close)
			deezerURL, itunesURL, musicBrainzURL = srv.URL+"/deezer", srv.URL+"/itunes", srv.URL+"/mb"
			conf.Server.LastFM.ApiKey = ""
			res := Search(ctx, "Astrofaes", "Ancestors' Shadows (Тени Предков)")
			Expect(res).To(HaveLen(2))
			Expect(res[0].URL).To(Equal("xl"))
			Expect(res[1].Source).To(Equal("iTunes"))
		})

		It("skips Last.fm without an API key and tolerates failing sources", func() {
			conf.Server.LastFM.ApiKey = ""
			deezerURL = "http://127.0.0.1:1"
			res := Search(ctx, "Zero 7", "In The Waiting Line")
			var sources []string
			for _, c := range res {
				sources = append(sources, c.Source)
			}
			Expect(strings.Join(sources, ",")).To(Equal("iTunes,Cover Art Archive"))
		})
	})
})
