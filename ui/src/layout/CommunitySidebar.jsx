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

const useStyles = makeStyles((theme) => ({
  sidebar: {
    position: 'fixed',
    top: 48,
    right: 0,
    bottom: (props) => (props.addPadding ? 80 : 0),
    width: COMMUNITY_SIDEBAR_WIDTH,
    overflowY: 'auto',
    borderLeft: `1px solid ${theme.palette.divider}`,
    backgroundColor: theme.palette.background.default,
    zIndex: theme.zIndex.appBar - 1,
    padding: theme.spacing(1),
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
  const isSmallScreen = useMediaQuery(theme.breakpoints.down('md'))
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
    httpClient('/api/community/recent?count=20')
      .then((resp) => setRecent(resp.json || []))
      .catch(() => {})
  }, [])

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
      {recent.length === 0 ? (
        <Typography className={classes.empty}>
          {translate('community.noPlays')}
        </Typography>
      ) : (
        <List dense disablePadding>
          {recent.map((entry, idx) => (
            <RecentEntry
              key={`${entry.userId}-${entry.mediaFileId}-${entry.submissionTime}-${idx}`}
              entry={entry}
            />
          ))}
        </List>
      )}
    </aside>
  )
}

export default CommunitySidebar
