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

### Releases (year browser)
`/releases` in the menu: pick a year and see every album from it as a dated
list, sorted by actual release day (ascending or descending), with covers,
artists and song counts. A "Shuffle <year>" button queues up to 500 random
songs from that year. Uses tag dates (`DATE`, then `ORIGINALDATE`,
`RELEASEDATE`), falling back to the bare year when no full date is tagged.

### Server chat
A chat box on the `/community` page shared by everyone on the server. Supports text
and images — paste an image straight into the input (or attach one with the photo
button), preview it, and send. Messages appear live for all connected users via SSE.
Users can delete their own messages; admins can delete any. Images are stored under
the data folder (`artwork/chat/`) and are removed when their message is deleted.
Gated by the same `ND_ENABLECOMMUNITY` flag.

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

`.github/workflows/docker-metal.yml` builds the standard Navidrome Docker
image (Alpine + ffmpeg) on every push, in two channels:

| Branch | Image tag | Stack file | Purpose |
|--------|-----------|------------|---------|
| `metal-edition` | `:latest` | `deploy/portainer-stack-staging.yml` (port 4672) | staging — every change lands here first |
| `metal-stable` | `:stable` | `deploy/portainer-stack.yml` (port 4671) | production |

Promotion: after testing on staging, fast-forward the stable branch and
re-pull the prod stack:

```
git push origin metal-edition:metal-stable
```

Both stacks coexist with a stock Navidrome instance — separate images,
container names, data volumes, and ports. The staging stack mounts the
same music read-only but keeps its own database.

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
