import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { useTranslate, Link } from 'react-admin'
import {
  Card,
  CardContent,
  Typography,
  Chip,
  Tooltip,
  makeStyles,
} from '@material-ui/core'
import AlbumIcon from '@material-ui/icons/Album'
import LaunchIcon from '@material-ui/icons/Launch'
import PropTypes from 'prop-types'
import { Title } from '../common'
import { httpClient } from '../dataProvider'
import subsonic from '../subsonic'
import { GROUPS, groupAlerts, parseReleaseDate } from './releaseGroups'

const MAX_ALERTS = 500

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(2, 1),
  },
  header: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
    alignItems: 'center',
    marginBottom: theme.spacing(1),
  },
  pageTitle: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginRight: theme.spacing(1),
  },
  filters: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.5),
  },
  sectionHeading: {
    fontSize: '0.75rem',
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    color: theme.palette.text.secondary,
    margin: theme.spacing(3, 0, 0.5),
  },
  section: {
    margin: theme.spacing(1.5, 0, 0.5),
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
  },
  sectionTitle: {
    fontSize: '1rem',
    fontWeight: 600,
  },
  count: {
    fontSize: '0.8rem',
    color: theme.palette.text.secondary,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
    gap: theme.spacing(2),
  },
  tile: {
    minWidth: 0,
  },
  cover: {
    position: 'relative',
    width: '100%',
    paddingTop: '100%',
    borderRadius: theme.spacing(0.5),
    overflow: 'hidden',
    backgroundColor: theme.palette.action.hover,
  },
  coverImg: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
    objectFit: 'cover',
  },
  coverIcon: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    fontSize: 48,
    color: theme.palette.text.disabled,
  },
  typeBadge: {
    position: 'absolute',
    top: theme.spacing(0.5),
    left: theme.spacing(0.5),
    padding: '1px 6px',
    borderRadius: 3,
    fontSize: '0.65rem',
    fontWeight: 600,
    color: '#fff',
    backgroundColor: 'rgba(0, 0, 0, 0.65)',
  },
  title: {
    fontSize: '0.85rem',
    fontWeight: 600,
    marginTop: theme.spacing(0.75),
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  artist: {
    fontSize: '0.8rem',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  detail: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
  },
  empty: {
    fontSize: '0.85rem',
    color: theme.palette.text.disabled,
    padding: theme.spacing(2, 0),
  },
}))

const formatReleaseDate = (date) => {
  const parsed = parseReleaseDate(date)
  if (!parsed) return date
  return parsed.day.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    ...(!parsed.monthOnly && { day: 'numeric' }),
  })
}

// Candidate cover images, best first: the Cover Art Archive for MusicBrainz
// release groups, Metal Archives' album image, then the artist's own image
// from the library.
const coverUrls = (alert) => {
  const urls = []
  const [source, externalId = ''] = alert.id.split(/:(.*)/)
  if (source === 'musicbrainz' && externalId) {
    urls.push(
      `https://coverartarchive.org/release-group/${externalId}/front-250`,
    )
  }
  if (source === 'metalarchives' && /^\d{4,}$/.test(externalId)) {
    const d = externalId.split('')
    urls.push(
      `https://www.metal-archives.com/images/${d[0]}/${d[1]}/${d[2]}/${d[3]}/${externalId}.jpg`,
    )
  }
  urls.push(subsonic.getCoverArtUrl({ id: alert.artistId }, 300, true))
  return urls
}

const ReleaseCover = ({ alert }) => {
  const classes = useStyles()
  const urls = useMemo(() => coverUrls(alert), [alert])
  const [index, setIndex] = useState(0)
  return (
    <div className={classes.cover}>
      {index < urls.length ? (
        <img
          className={classes.coverImg}
          src={urls[index]}
          alt={alert.title}
          loading="lazy"
          referrerPolicy="no-referrer"
          onError={() => setIndex(index + 1)}
        />
      ) : (
        <AlbumIcon className={classes.coverIcon} />
      )}
      {alert.releaseType && (
        <span className={classes.typeBadge}>{alert.releaseType}</span>
      )}
    </div>
  )
}

ReleaseCover.propTypes = {
  alert: PropTypes.object.isRequired,
}

const ReleaseTile = ({ alert }) => {
  const classes = useStyles()
  return (
    <div className={classes.tile}>
      <ReleaseCover alert={alert} />
      <Tooltip title={alert.title}>
        <Typography className={classes.title}>{alert.title}</Typography>
      </Tooltip>
      <Typography className={classes.artist}>
        <Link to={`/artist/${alert.artistId}/show`}>{alert.artistName}</Link>
      </Typography>
      <Typography className={classes.detail}>
        {formatReleaseDate(alert.releaseDate)}
        {alert.externalUrl && (
          <a
            href={alert.externalUrl}
            target="_blank"
            rel="noopener noreferrer"
            title={alert.source}
          >
            <LaunchIcon style={{ fontSize: 13 }} />
          </a>
        )}
      </Typography>
    </div>
  )
}

ReleaseTile.propTypes = {
  alert: PropTypes.object.isRequired,
}

const NewReleasesPage = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const [alerts, setAlerts] = useState(null)
  const [type, setType] = useState(null)
  // Bumped by SSE releaseAlert events
  const alertBump = useSelector((state) => state.activity.releaseAlerts)

  const fetchAlerts = useCallback(() => {
    httpClient(`/api/community/alerts?count=${MAX_ALERTS}`)
      .then((resp) => setAlerts(resp.json?.alerts || []))
      .catch(() => setAlerts([]))
  }, [])

  useEffect(() => {
    fetchAlerts()
  }, [fetchAlerts, alertBump])

  const types = useMemo(
    () =>
      [
        ...new Set((alerts || []).map((a) => a.releaseType).filter(Boolean)),
      ].sort(),
    [alerts],
  )

  const groups = useMemo(
    () =>
      groupAlerts(
        (alerts || []).filter((a) => !type || a.releaseType === type),
      ),
    [alerts, type],
  )

  const renderGroups = (upcoming) =>
    GROUPS.filter(
      (g) => g.upcoming === upcoming && groups[g.key].length > 0,
    ).map((g) => (
      <div key={g.key}>
        <div className={classes.section}>
          <Typography className={classes.sectionTitle}>
            {translate(`newReleases.groups.${g.key}`)}
          </Typography>
          <Typography className={classes.count}>
            {groups[g.key].length}
          </Typography>
        </div>
        <div className={classes.grid}>
          {groups[g.key].map((alert) => (
            <ReleaseTile key={alert.id} alert={alert} />
          ))}
        </div>
      </div>
    ))

  const hasAny = (upcoming) =>
    GROUPS.some((g) => g.upcoming === upcoming && groups[g.key].length > 0)

  return (
    <div className={classes.root}>
      <Title subTitle={'newReleases.title'} />
      <Card>
        <CardContent>
          <div className={classes.header}>
            <Typography className={classes.pageTitle}>
              {translate('newReleases.title')}
            </Typography>
            {types.length > 1 && (
              <div className={classes.filters}>
                <Chip
                  size="small"
                  label={translate('newReleases.allTypes')}
                  color={type ? 'default' : 'primary'}
                  onClick={() => setType(null)}
                />
                {types.map((t) => (
                  <Chip
                    key={t}
                    size="small"
                    label={t}
                    color={type === t ? 'primary' : 'default'}
                    onClick={() => setType(type === t ? null : t)}
                  />
                ))}
              </div>
            )}
          </div>
          {alerts !== null && !hasAny(false) && !hasAny(true) && (
            <Typography className={classes.empty}>
              {translate('releaseAlerts.empty')}
            </Typography>
          )}
          {hasAny(false) && (
            <Typography className={classes.sectionHeading}>
              {translate('newReleases.outNow')}
            </Typography>
          )}
          {renderGroups(false)}
          {hasAny(true) && (
            <Typography className={classes.sectionHeading}>
              {translate('newReleases.upcoming')}
            </Typography>
          )}
          {renderGroups(true)}
        </CardContent>
      </Card>
    </div>
  )
}

export default NewReleasesPage
