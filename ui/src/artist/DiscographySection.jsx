import React, { useState, useCallback } from 'react'
import PropTypes from 'prop-types'
import { useTranslate } from 'react-admin'
import {
  Button,
  Chip,
  Typography,
  Card,
  CardContent,
  CircularProgress,
  IconButton,
  Tooltip,
  makeStyles,
} from '@material-ui/core'
import AlbumIcon from '@material-ui/icons/Album'
import RefreshIcon from '@material-ui/icons/Refresh'
import LaunchIcon from '@material-ui/icons/Launch'
import { httpClient } from '../dataProvider'
import config from '../config'

// Release types shown by default; the rest (Demo, Live, Compilation,
// Single...) are revealed via the type chips
const DEFAULT_TYPES = ['Full-length', 'EP', 'Split']

const useStyles = makeStyles((theme) => ({
  root: {
    padding: theme.spacing(1, 2, 3, 2),
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
  title: {
    fontWeight: 600,
    fontSize: '0.9rem',
    marginRight: theme.spacing(1),
  },
  grid: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1.5),
  },
  // Matches the dimmed "missingAlbum" look of AlbumGridView, but for
  // releases that were never in the library
  ghostCard: {
    width: 150,
    opacity: 0.55,
    border: `1px dashed ${theme.palette.divider}`,
    backgroundColor: 'transparent',
    boxShadow: 'none',
    '&:hover': {
      opacity: 0.85,
    },
  },
  ghostContent: {
    padding: `${theme.spacing(1)}px !important`,
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(0.5),
  },
  ghostCover: {
    width: '100%',
    height: 134,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.palette.action.hover,
    borderRadius: theme.spacing(0.5),
    color: theme.palette.text.disabled,
  },
  ghostTitle: {
    fontSize: '0.8rem',
    fontWeight: 500,
    lineHeight: 1.3,
    overflow: 'hidden',
    display: '-webkit-box',
    '-webkit-line-clamp': 2,
    '-webkit-box-orient': 'vertical',
  },
  ghostDetail: {
    fontSize: '0.7rem',
    color: theme.palette.text.secondary,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  empty: {
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
  },
  count: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
  },
}))

const DiscographySection = ({ record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [show, setShow] = useState(false)
  const [loading, setLoading] = useState(false)
  const [discography, setDiscography] = useState(null)
  const [activeTypes, setActiveTypes] = useState(DEFAULT_TYPES)

  const fetchDiscography = useCallback(
    (refresh) => {
      setLoading(true)
      httpClient(
        `/api/artist/${record.id}/discography${refresh ? '?refresh=true' : ''}`,
      )
        .then((resp) => setDiscography(resp.json))
        .catch(() => setDiscography({ releases: [] }))
        .finally(() => setLoading(false))
    },
    [record.id],
  )

  const toggle = () => {
    const next = !show
    setShow(next)
    if (next && !discography) {
      fetchDiscography(false)
    }
  }

  if (!config.enableDiscography) {
    return null
  }

  const releases = discography?.releases || []
  const allTypes = [...new Set(releases.map((r) => r.releaseType))].sort()
  const missing = releases.filter(
    (r) => !r.owned && activeTypes.includes(r.releaseType),
  )
  const ownedCount = releases.filter((r) => r.owned).length

  const toggleType = (type) =>
    setActiveTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type],
    )

  return (
    <div className={classes.root}>
      <div className={classes.header}>
        <Button variant="outlined" size="small" onClick={toggle}>
          {translate(
            show ? 'discography.hideMissing' : 'discography.showMissing',
          )}
        </Button>
        {show && !loading && discography && (
          <>
            <Typography className={classes.count}>
              {translate('discography.summary', {
                owned: ownedCount,
                total: releases.length,
              })}
            </Typography>
            {allTypes.map((type) => (
              <Chip
                key={type}
                label={type}
                size="small"
                color={activeTypes.includes(type) ? 'primary' : 'default'}
                onClick={() => toggleType(type)}
              />
            ))}
            <Tooltip title={translate('discography.refresh')}>
              <IconButton size="small" onClick={() => fetchDiscography(true)}>
                <RefreshIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </>
        )}
        {show && loading && <CircularProgress size={18} />}
      </div>
      {show && !loading && (
        <div className={classes.grid}>
          {missing.length === 0 ? (
            <Typography className={classes.empty}>
              {translate(
                releases.length === 0
                  ? 'discography.noData'
                  : 'discography.complete',
              )}
            </Typography>
          ) : (
            missing.map((release) => (
              <Card
                key={`${release.source}-${release.title}-${release.year}`}
                className={classes.ghostCard}
              >
                <CardContent className={classes.ghostContent}>
                  <div className={classes.ghostCover}>
                    <AlbumIcon fontSize="large" />
                  </div>
                  <Typography
                    className={classes.ghostTitle}
                    title={release.title}
                  >
                    {release.title}
                  </Typography>
                  <div className={classes.ghostDetail}>
                    <span>
                      {release.year || '—'} · {release.releaseType}
                    </span>
                    {release.externalUrl && (
                      <a
                        href={release.externalUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <LaunchIcon style={{ fontSize: 14 }} />
                      </a>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      )}
    </div>
  )
}

DiscographySection.propTypes = {
  record: PropTypes.object.isRequired,
}

export default DiscographySection
