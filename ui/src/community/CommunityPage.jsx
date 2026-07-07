import React, { useState, useEffect, useCallback } from 'react'
import { useSelector } from 'react-redux'
import { useTranslate } from 'react-admin'
import { Grid, Tabs, Tab, makeStyles, Typography } from '@material-ui/core'
import { httpClient } from '../dataProvider'
import { Title } from '../common'
import UserFilter from './UserFilter'
import TopList from './TopList'
import PlayFeed from './PlayFeed'

const FEED_PAGE_SIZE = 30

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2, 1),
  },
  header: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(2),
    alignItems: 'center',
    marginBottom: theme.spacing(2),
  },
  pageTitle: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginRight: theme.spacing(2),
  },
}))

const RANGES = ['7d', '30d', '90d', 'all']

const CommunityPage = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const lastPlayEvent = useSelector((state) => state.activity.lastPlayEvent)

  const [range, setRange] = useState('30d')
  const [users, setUsers] = useState([])
  const [selectedUsers, setSelectedUsers] = useState([])
  const [tops, setTops] = useState({ song: [], album: [], artist: [] })
  const [feed, setFeed] = useState([])
  const [hasMore, setHasMore] = useState(false)

  const usersQuery = selectedUsers.length
    ? `&users=${selectedUsers.join(',')}`
    : ''

  useEffect(() => {
    httpClient('/api/community/users')
      .then((resp) => setUsers(resp.json || []))
      .catch(() => {})
  }, [])

  const fetchTops = useCallback(() => {
    Promise.all(
      ['song', 'album', 'artist'].map((kind) =>
        httpClient(
          `/api/community/top?type=${kind}&range=${range}&count=10${usersQuery}`,
        ).then((resp) => [kind, resp.json || []]),
      ),
    )
      .then((results) => setTops(Object.fromEntries(results)))
      .catch(() => {})
  }, [range, usersQuery])

  const fetchFeed = useCallback(
    (offset, append) => {
      httpClient(
        `/api/community/recent?count=${FEED_PAGE_SIZE}&offset=${offset}${usersQuery}`,
      )
        .then((resp) => {
          const page = resp.json || []
          setFeed((prev) => (append ? [...prev, ...page] : page))
          setHasMore(page.length === FEED_PAGE_SIZE)
        })
        .catch(() => {})
    },
    [usersQuery],
  )

  // Refresh everything when the range/user filter changes, and refresh
  // live when someone scrobbles (playEvent SSE)
  useEffect(() => {
    fetchTops()
  }, [fetchTops, lastPlayEvent])

  useEffect(() => {
    fetchFeed(0, false)
  }, [fetchFeed, lastPlayEvent])

  const loadMore = useCallback(
    () => fetchFeed(feed.length, true),
    [fetchFeed, feed.length],
  )

  return (
    <div className={classes.root}>
      <Title subTitle={'community.title'} />
      <div className={classes.header}>
        <Typography className={classes.pageTitle}>
          {translate('community.title')}
        </Typography>
        <Tabs
          value={range}
          onChange={(e, v) => setRange(v)}
          indicatorColor="primary"
          textColor="primary"
        >
          {RANGES.map((r) => (
            <Tab
              key={r}
              value={r}
              label={translate(`community.ranges.${r}`)}
              style={{ minWidth: 72 }}
            />
          ))}
        </Tabs>
        <UserFilter
          users={users}
          selected={selectedUsers}
          onChange={setSelectedUsers}
        />
      </div>
      <Grid container spacing={2}>
        <Grid item xs={12} md={4}>
          <TopList
            kind="album"
            entries={tops.album}
            titleKey="community.topAlbums"
          />
        </Grid>
        <Grid item xs={12} md={4}>
          <TopList
            kind="artist"
            entries={tops.artist}
            titleKey="community.topArtists"
          />
        </Grid>
        <Grid item xs={12} md={4}>
          <TopList
            kind="song"
            entries={tops.song}
            titleKey="community.topSongs"
          />
        </Grid>
        <Grid item xs={12}>
          <PlayFeed entries={feed} hasMore={hasMore} onLoadMore={loadMore} />
        </Grid>
      </Grid>
    </div>
  )
}

export default CommunityPage
