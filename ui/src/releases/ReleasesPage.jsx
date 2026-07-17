import React, { useState, useEffect, useCallback } from 'react'
import { useDataProvider, useTranslate, Link } from 'react-admin'
import {
  Card,
  CardContent,
  Typography,
  IconButton,
  Button,
  Select,
  MenuItem,
  Avatar,
  Tooltip,
  makeStyles,
} from '@material-ui/core'
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft'
import ChevronRightIcon from '@material-ui/icons/ChevronRight'
import ArrowUpwardIcon from '@material-ui/icons/ArrowUpward'
import ArrowDownwardIcon from '@material-ui/icons/ArrowDownward'
import { Title } from '../common'
import { ShuffleAllButton } from '../common/ShuffleAllButton'
import subsonic from '../subsonic'

const PAGE_SIZE = 100

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2, 1),
  },
  header: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
    alignItems: 'center',
    marginBottom: theme.spacing(2),
  },
  pageTitle: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginRight: theme.spacing(1),
  },
  yearSelect: {
    minWidth: 90,
  },
  count: {
    fontSize: '0.8rem',
    color: theme.palette.text.secondary,
    marginLeft: theme.spacing(1),
  },
  row: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1.5),
    padding: theme.spacing(0.75, 0),
    borderBottom: `1px solid ${theme.palette.divider}`,
    '&:last-child': {
      borderBottom: 'none',
    },
  },
  cover: {
    width: theme.spacing(6),
    height: theme.spacing(6),
    borderRadius: theme.spacing(0.5),
  },
  info: { flex: 1, minWidth: 0 },
  albumName: {
    fontSize: '0.9rem',
    fontWeight: 500,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  artist: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  date: {
    fontSize: '0.8rem',
    color: theme.palette.text.secondary,
    flexShrink: 0,
    minWidth: 90,
    textAlign: 'right',
  },
  empty: {
    fontSize: '0.85rem',
    color: theme.palette.text.disabled,
    padding: theme.spacing(2, 0),
  },
  loadMore: {
    marginTop: theme.spacing(1),
  },
}))

// Best-known release date for an album: full tag date first, then original
// and release dates, then the bare year.
const albumDate = (album) =>
  album.date || album.originalDate || album.releaseDate || `${album.maxYear}`

const MONTHS = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
]

// "2026-03-14" / "2026-03" -> "Mar 2026"; "2026" stays as-is
const formatDate = (dateStr) => {
  const parts = String(dateStr).split('-')
  if (parts.length >= 2) {
    const month = MONTHS[parseInt(parts[1], 10) - 1]
    if (month) return `${month} ${parts[0]}`
  }
  return parts[0]
}

const ReleasesPage = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()

  const currentYear = new Date().getFullYear()
  const [year, setYear] = useState(currentYear)
  const [minYear, setMinYear] = useState(currentYear - 30)
  const [order, setOrder] = useState('ASC')
  const [albums, setAlbums] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)

  // Find the oldest album year once, to bound the year selector. Fetches a
  // small page because albums without any year (max_year 0) sort first.
  useEffect(() => {
    dataProvider
      .getList('album', {
        pagination: { page: 1, perPage: 20 },
        sort: { field: 'max_year', order: 'ASC' },
        filter: {},
      })
      .then((res) => {
        const first = res.data.find((a) => (a.minYear || a.maxYear) > 0)
        if (first) setMinYear(first.minYear || first.maxYear)
      })
      .catch(() => {})
  }, [dataProvider])

  const fetchPage = useCallback(
    (pageToLoad, append) => {
      dataProvider
        .getList('album', {
          pagination: { page: pageToLoad, perPage: PAGE_SIZE },
          // Both mappings keep undated (year-only) albums at the end; the
          // direction is baked into the mapping, so always request asc
          sort: {
            field: order === 'ASC' ? 'release_date' : 'release_date_desc',
            order: 'ASC',
          },
          filter: { release_year: year },
        })
        .then((res) => {
          setAlbums((prev) => (append ? [...prev, ...res.data] : res.data))
          setTotal(res.total)
          setPage(pageToLoad)
        })
        .catch(() => {})
    },
    [dataProvider, year, order],
  )

  useEffect(() => {
    fetchPage(1, false)
  }, [fetchPage])

  const years = []
  for (let y = currentYear; y >= minYear; y--) years.push(y)

  return (
    <div className={classes.root}>
      <Title subTitle={'releases.title'} />
      <Card>
        <CardContent>
          <div className={classes.header}>
            <Typography className={classes.pageTitle}>
              {translate('releases.title')}
            </Typography>
            <IconButton
              size="small"
              disabled={year <= minYear}
              onClick={() => setYear(year - 1)}
            >
              <ChevronLeftIcon />
            </IconButton>
            <Select
              className={classes.yearSelect}
              value={year}
              onChange={(e) => setYear(e.target.value)}
            >
              {years.map((y) => (
                <MenuItem key={y} value={y}>
                  {y}
                </MenuItem>
              ))}
            </Select>
            <IconButton
              size="small"
              disabled={year >= currentYear}
              onClick={() => setYear(year + 1)}
            >
              <ChevronRightIcon />
            </IconButton>
            <Tooltip title={translate('releases.toggleOrder')}>
              <IconButton
                size="small"
                onClick={() => setOrder(order === 'ASC' ? 'DESC' : 'ASC')}
              >
                {order === 'ASC' ? <ArrowUpwardIcon /> : <ArrowDownwardIcon />}
              </IconButton>
            </Tooltip>
            <ShuffleAllButton
              filters={{ year }}
              label={translate('releases.shuffleYear', { year })}
            />
            <span className={classes.count}>
              {translate('releases.albumCount', {
                smart_count: total,
              })}
            </span>
          </div>
          {albums.length === 0 ? (
            <Typography className={classes.empty}>
              {translate('releases.empty', { year })}
            </Typography>
          ) : (
            albums.map((album) => (
              <div key={album.id} className={classes.row}>
                <Link to={`/album/${album.id}/show`}>
                  <Avatar
                    className={classes.cover}
                    variant="square"
                    src={subsonic.getCoverArtUrl(album, 96)}
                    alt={album.name}
                    loading="lazy"
                  />
                </Link>
                <div className={classes.info}>
                  <Typography className={classes.albumName} title={album.name}>
                    <Link to={`/album/${album.id}/show`}>{album.name}</Link>
                  </Typography>
                  <Typography className={classes.artist}>
                    {album.albumArtist}
                  </Typography>
                </div>
                <span className={classes.date}>
                  {formatDate(albumDate(album))}
                </span>
              </div>
            ))
          )}
          {albums.length < total && (
            <Button
              className={classes.loadMore}
              size="small"
              fullWidth
              onClick={() => fetchPage(page + 1, true)}
            >
              {translate('releases.loadMore')}
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default ReleasesPage
