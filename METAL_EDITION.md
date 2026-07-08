# Navidrome — Metal Edition

A personal fork of [Navidrome](https://github.com/navidrome/navidrome) that adds
community listening features and metal-focused collection tools. All features are
additive and gated behind config flags, so this branch rebases cleanly onto upstream.

## Features

### Community sidebar
Persistent right-hand panel showing what every user is playing right now (with live
progress) and the server-wide recently-played feed. Updates are pushed over the
existing SSE event stream the moment anyone scrobbles. Toggled with the people icon
in the app bar; the open/closed state is remembered per browser.

### Community page
`/community` in the menu: top albums, artists, and songs across all users over a
selectable range (7/30/90 days or all time), per-user filter chips, and a paged feed
of everyone's plays. Backed by upstream's `scrobbles` play-history table, so history
recorded before this fork was installed is included.

### Missing albums
Every artist page gets a **Show missing albums** toggle that compares the library
against the artist's full external discography and renders what's absent as dimmed
ghost cards (year, release type, link to the source). Owned releases are matched by
MusicBrainz release-group id when files are tagged (use Picard!), falling back to
normalized title matching. Defaults to Full-length + EP + Split, with chips to reveal
demos, live albums, compilations, singles, etc.

Discographies come from:
- **MusicBrainz** (default on) — reliable API, requires artists to have a
  MusicBrainz artist id in their tags.
- **Metal Archives** (opt-in) — scraped politely (browser UA, 1 request/3s, 30-day
  cache). Catches the demos/splits/EPs MusicBrainz misses. Band names are resolved
  once via MA's search and cached; ambiguous names (multiple bands, hi "Nirvana")
  are skipped rather than guessed. If MA blocks or changes, the feature silently
  degrades to MusicBrainz-only.

### New release alerts
A bell in the app bar with a per-user unread badge. A scheduled job (default 6 AM
daily) checks each library artist for release groups dated in the last 14 days or
announced for the future, and scans Metal Archives' upcoming-releases page (when
enabled) for library band names. New alerts are broadcast live via SSE.

## Configuration

| Flag | Default | Purpose |
|------|---------|---------|
| `ND_ENABLECOMMUNITY` | `true` | Community sidebar, page, and play feed endpoints |
| `ND_MUSICBRAINZ_ENABLED` | `true` | MusicBrainz discography + release checks |
| `ND_MUSICBRAINZ_BASEURL` | `https://musicbrainz.org` | Override for mirrors |
| `ND_METALARCHIVES_ENABLED` | `false` | Metal Archives scraping (opt-in) |
| `ND_METALARCHIVES_BASEURL` | `https://www.metal-archives.com` | |
| `ND_ENABLERELEASEALERTS` | `true` | Release alert job + bell (needs MBZ or MA on) |
| `ND_RELEASEALERTSSCHEDULE` | `0 6 * * *` | Cron schedule for the release check |

Upstream's `ND_ENABLESCROBBLEHISTORY` (default `true`) must stay on — the community
features read from the `scrobbles` table it populates.

## Deployment

Pushing the `metal-edition` branch to GitHub triggers
`.github/workflows/docker-metal.yml`, which builds the standard Navidrome
Docker image (Alpine + ffmpeg) from this branch and publishes it to
`ghcr.io/<owner>/navidrome-metal:latest`. Deploy it with
`deploy/portainer-stack.yml` (edit the image owner, host port, and music
path first). It coexists with a stock Navidrome instance — separate image,
container name, data volume, and port.

## Maintenance notes

- New code lives in new files: `core/community/`, `adapters/musicbrainz/`,
  `adapters/metalarchives/`, `server/nativeapi/community.go` + `discography.go`,
  `ui/src/community/`, `ui/src/layout/CommunitySidebar.jsx` + `ReleaseAlertsPanel.jsx`,
  `ui/src/artist/DiscographySection.jsx`.
- Upstream files carry only small flag-gated insertions (route mounts, one SSE
  broadcast in `play_tracker.go`, AppBar entries, config fields). Rebase hotspots:
  `server/nativeapi/native_api.go`, `core/scrobbler/play_tracker.go`,
  `ui/src/layout/AppBar.jsx`.
- Metal Archives parser is pinned by fixture tests; check the scraper still works
  against the live site with `ND_MA_LIVE_TEST=1 go test ./adapters/metalarchives/ -run TestLive -v`.
