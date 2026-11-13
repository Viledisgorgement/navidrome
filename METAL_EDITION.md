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
list, sorted by actual release day (ascending or descending). A
"Shuffle <year>" button queues up to 500 random songs from that year. Uses
tag dates (`DATE`, then `ORIGINALDATE`, `RELEASEDATE`), falling back to the
bare year when no full date is tagged; albums appear only under the year
their displayed date belongs to.

### Server chat
A chat box on the `/community` page shared by everyone on the server. Supports text
and images — paste an image straight into the input (or attach one with the photo
button), preview it, and send. Messages appear live for all connected users via SSE.
Users can delete their own messages; admins can delete any. Images are stored under
the data folder (`artwork/chat/`) and are removed when their message is deleted.
Gated by the same `ND_ENABLECOMMUNITY` flag.

### Change cover art
Hover an album's cover and click the image-search icon (admins, or everyone when
`ND_ENABLEARTWORKUPLOAD` is on). The dialog is pre-filled with the album artist and
title and searches Deezer, iTunes, and MusicBrainz/Cover Art Archive (plus Last.fm
when `ND_LASTFM_APIKEY` is set; get a free key at https://www.last.fm/api/account/create). Bracketed title parts like a translated title are dropped on retry when nothing matches. Click a result to use it, or paste an image address.
The camera icon uploads a file from your computer instead.

Like Airsonic, the chosen image is saved as `cover.<ext>` in the album's folder (the
parent folder for albums split into disc subfolders), and any existing `cover.*` is
kept as `cover.*.bak`. The folder is rescanned right away. This needs the music
mounted read-write and the default `ND_COVERARTPRIORITY` (`cover.*` first). If the
folder is not writable, the cover is stored under the data folder
(`artwork/album/`) instead and takes priority over everything else; the trash icon
removes such a cover. Either way it survives rescans.

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
same music as prod but keeps its own database.

## Sonos

Navidrome has no Sonos support of its own. `deploy/portainer-stack-bonob.yml`
runs [bonob](https://github.com/simojenki/bonob) beside it, which presents the
library to Sonos as a music service. On Sonos S2 the Sonos cloud talks to bonob
over the internet and the speaker pulls audio from bonob's public address, so
the speaker can be on a different network from the server (an office, say).

1. Deploy the bonob stack with `BONOB_URL`, `BONOB_SECRET` and `NAVIDROME_URL`
   set (see the comments in the stack file).
2. Publish `http://<host>:4534` at `BONOB_URL` over HTTPS on port 443, for
   example a `bonob.` subdomain in Caddy (`reverse_proxy <host>:4534`) or a
   cloudflared public hostname. Opening `BONOB_URL` in a browser should show
   the bonob page.
3. Sign in at https://developer.sonos.com with the same Sonos account the
   speaker uses, create a **Content Integration (Sandbox)**, and fill it in as
   described in bonob's
   [S2 setup guide](https://github.com/simojenki/bonob/blob/master/docs/sonos-s2-setup.md)
   (SMAPI endpoint `BONOB_URL/ws/sonos`, OAuth with redirect `BONOB_URL/login`,
   your Sonos ID under Service Deployment Settings). Press Refresh, then Send.
4. After about 10 minutes, restart the Sonos app, search for the service name
   under music services, add it, and sign in with a Navidrome user.

Sonos plays are scrobbled, so they appear in the community sidebar and history.

Caveats: this uses Sonos's developer sandbox, which Sonos does not officially
support for self-hosting (they only support Plex), and there are reports of
sandbox services vanishing from the app and needing to be recreated. Older S1
systems register differently; see bonob's S1 guide.

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
