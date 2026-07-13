import React, { useState, useEffect, useCallback } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { useTranslate, Link } from 'react-admin'
import {
  makeStyles,
  Typography,
  List,
  ListItem,
  Avatar,
  Divider,
  Button,
  useTheme,
  useMediaQuery,
} from '@material-ui/core'
import subsonic from '../subsonic'
import { httpClient } from '../dataProvider'
import { useInterval } from '../common'
import { nowPlayingCountSync } from '../actions'
import { NowPlayingItem } from './NowPlayingPanel'
import config from '../config'

export const COMMUNITY_SIDEBAR_WIDTH = 280

// Entries newer than this show by default; older ones sit behind "Show more"
const RECENT_WINDOW_MS = 2 * 60 * 60 * 1000
const PAGE_SIZE = 20

const useStyles = makeStyles((theme) => ({
  sidebar: {
    width: COMMUNITY_SIDEBAR_WIDTH,
    flexShrink: 0,
    position: 'sticky',
    top: 0,
    alignSelf: 'flex-start',
    height: '100vh',
    overflowY: 'auto',
    boxSizing: 'border-box',
    borderLeft: `1px solid ${theme.palette.divider}`,
    backgroundColor: theme.palette.background.default,
    // The sidebar renders outside RALayout's CssBaseline, so text color
    // must be set explicitly or titles inherit the browser default (black)
    color: theme.palette.text.primary,
    // Clear the fixed AppBar and, when the queue is loaded, the player bar
    paddingTop: 48,
    paddingBottom: (props) => (props.addPadding ? 88 : theme.spacing(1)),
    paddingLeft: theme.spacing(1),
    paddingRight: theme.spacing(1),
  },
  sectionTitle: {
    fontWeight: 600,
    fontSize: '0.8rem',
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
    color: theme.palette.text.secondary,
    padding: theme.spacing(1, 1, 0.5, 1),
  },
  empty: {
    fontSize: '0.75rem',
    color: theme.palette.text.disabled,
    padding: theme.spacing(0.5, 1, 1, 1),
  },
  recentItem: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(0.75, 1),
  },
  recentAvatar: {
    width: theme.spacing(5),
    height: theme.spacing(5),
    borderRadius: theme.spacing(0.5),
  },
  recentContent: {
    flex: 1,
    minWidth: 0,
  },
  recentTitle: {
    fontSize: '0.8rem',
    fontWeight: 500,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  recentDetail: {
    fontSize: '0.7rem',
    color: theme.palette.text.secondary,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  recentUser: {
    fontSize: '0.65rem',
    color: theme.palette.text.disabled,
  },
  showMore: {
    margin: theme.spacing(0.5, 1),
    fontSize: '0.7rem',
  },
}))

const timeAgo = (date, translate) => {
  const minutes = Math.floor((Date.now() - new Date(date).getTime()) / 60000)
  if (minutes < 1) return translate('community.justNow')
  if (minutes < 60)
    return translate('nowPlaying.minutesAgo', { smart_count: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return translate('community.hoursAgo', { smart_count: hours })
  return new Date(date).toLocaleDateString()
}

const RecentEntry = ({ entry }) => {
  const classes = useStyles()
  const translate = useTranslate()
  return (
    <ListItem className={classes.recentItem} disableGutters>
      <Link to={`/album/${entry.albumId}/show`}>
        <Avatar
          className={classes.recentAvatar}
          variant="square"
          src={subsonic.getCoverArtUrl(
            { id: entry.mediaFileId, album: entry.album },
            80,
          )}
          alt={entry.album}
          loading="lazy"
        />
      </Link>
      <div className={classes.recentContent}>
        <Typography className={classes.recentTitle} title={entry.title}>
          {entry.title}
        </Typography>
        <Typography className={classes.recentDetail} title={entry.artist}>
          {entry.artist}
        </Typography>
        <Typography className={classes.recentUser}>
          {entry.userName} · {timeAgo(entry.submissionTime, translate)}
        </Typography>
      </div>
    </ListItem>
  )
}

const CommunitySidebar = () => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const theme = useTheme()
  const isSmallScreen = useMediaQuery(theme.breakpoints.down('sm'))
  const open = useSelector((state) => state.settings.communitySidebarOpen)
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue?.length > 0 })

  const serverUp = useSelector(
    (state) => !!state.activity.serverStart.startTime,
  )
  const nowPlayingLastUpdate = useSelector(
    (state) => state.activity.nowPlayingLastUpdate,
  )
  const lastPlayEvent = useSelector((state) => state.activity.lastPlayEvent)
  const streamReconnected = useSelector(
    (state) => state.activity.streamReconnected,
  )

  const [nowPlaying, setNowPlaying] = useState([])
  const [recent, setRecent] = useState([])
  // pages > 0 means the user asked for history beyond the 2h window
  const [pages, setPages] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [now, setNow] = useState(Date.now())

  const visible = open && !isSmallScreen && config.enableCommunity

  const fetchNowPlaying = useCallback(() => {
    subsonic
      .getNowPlaying()
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          const entries = data.nowPlaying?.entry || []
          const fetchTime = Date.now()
          setNowPlaying(entries.map((e) => ({ ...e, _fetchedAt: fetchTime })))
          dispatch(nowPlayingCountSync({ count: entries.length }))
        }
      })
      .catch(() => {})
  }, [dispatch])

  const fetchRecent = useCallback(() => {
    const count = PAGE_SIZE * (pages + 1)
    httpClient(`/api/community/recent?count=${count}`)
      .then((resp) => {
        const entries = resp.json || []
        setRecent(entries)
        setHasMore(entries.length === count)
      })
      .catch(() => {})
  }, [pages])

  // Refresh on SSE signals and reconnections
  useEffect(() => {
    if (visible && serverUp) fetchNowPlaying()
  }, [
    visible,
    serverUp,
    nowPlayingLastUpdate,
    streamReconnected,
    fetchNowPlaying,
  ])

  useEffect(() => {
    if (visible && serverUp) fetchRecent()
  }, [visible, serverUp, lastPlayEvent, streamReconnected, fetchRecent])

  // Animate progress bars and fall back to slow polling
  useInterval(() => setNow(Date.now()), visible ? 1000 : null)
  useInterval(
    () => {
      if (visible && serverUp) {
        fetchNowPlaying()
        fetchRecent()
      }
    },
    visible ? 60000 : null,
  )

  const getArtistLink = useCallback((artistId) => {
    if (!artistId) return null
    return config.devShowArtistPage && artistId !== config.variousArtistsId
      ? `/artist/${artistId}/show`
      : `/album?filter={"artist_id":"${artistId}"}&order=ASC&sort=max_year&displayedFilters={"compilation":true}&perPage=15`
  }, [])

  const noop = useCallback(() => {}, [])

  if (!visible) {
    return null
  }

  const cutoff = now - RECENT_WINDOW_MS
  const expanded = pages > 0
  const visibleRecent = expanded
    ? recent
    : recent.filter((e) => new Date(e.submissionTime).getTime() >= cutoff)
  const hiddenOlder = recent.length - visibleRecent.length
  const canShowMore = expanded ? hasMore : hiddenOlder > 0 || hasMore

  return (
    <aside
      className={classes.sidebar}
      aria-label={translate('community.sidebarTitle')}
    >
      <Typography className={classes.sectionTitle}>
        {translate('nowPlaying.title')}
      </Typography>
      {nowPlaying.length === 0 ? (
        <Typography className={classes.empty}>
          {translate('nowPlaying.empty')}
        </Typography>
      ) : (
        <List dense disablePadding>
          {nowPlaying.map((entry) => (
            <NowPlayingItem
              key={`${entry.username}-${entry.playerName}`}
              nowPlayingEntry={entry}
              onLinkClick={noop}
              getArtistLink={getArtistLink}
              now={now}
            />
          ))}
        </List>
      )}
      <Divider />
      <Typography className={classes.sectionTitle}>
        {translate('community.recentlyPlayed')}
      </Typography>
      {visibleRecent.length === 0 ? (
        <Typography className={classes.empty}>
          {translate('community.noRecentPlays')}
        </Typography>
      ) : (
        <List dense disablePadding>
          {visibleRecent.map((entry, idx) => (
            <RecentEntry
              key={`${entry.userId}-${entry.mediaFileId}-${entry.submissionTime}-${idx}`}
              entry={entry}
            />
          ))}
        </List>
      )}
      {canShowMore && (
        <Button
          className={classes.showMore}
          size="small"
          fullWidth
          onClick={() => setPages(pages + 1)}
        >
          {translate('community.showMore')}
        </Button>
      )}
      {expanded && (
        <Button
          className={classes.showMore}
          size="small"
          fullWidth
          onClick={() => setPages(0)}
        >
          {translate('community.showRecentOnly')}
        </Button>
      )}
    </aside>
  )
}

export default CommunitySidebar
