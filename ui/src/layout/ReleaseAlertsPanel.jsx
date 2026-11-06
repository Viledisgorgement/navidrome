import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { useTranslate, Link } from 'react-admin'
import {
  Popover,
  IconButton,
  Badge,
  Tooltip,
  Card,
  CardContent,
  List,
  ListItem,
  Typography,
  makeStyles,
} from '@material-ui/core'
import NotificationsIcon from '@material-ui/icons/Notifications'
import LaunchIcon from '@material-ui/icons/Launch'
import { httpClient } from '../dataProvider'
import { GROUPS, groupAlerts } from '../newreleases/releaseGroups'

// The bell shows the same date groups as the New Releases page, minus
// releases from earlier months and undated ones ("See all" has those).
const BELL_GROUPS = GROUPS.filter((g) => !['earlier', 'tba'].includes(g.key))
const MAX_FETCH = 500
const MAX_SHOWN = 50

// Alerts store ISO dates (YYYY-MM-DD or YYYY-MM) so they sort correctly;
// show them in the user's locale. Anything else is shown as-is.
const formatReleaseDate = (date) => {
  const m = /^(\d{4})-(\d{2})(?:-(\d{2}))?$/.exec(date || '')
  if (!m) return date
  const [, year, month, day] = m
  const d = new Date(Number(year), Number(month) - 1, Number(day || 1))
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    ...(day && { day: 'numeric' }),
  })
}

const useStyles = makeStyles((theme) => ({
  button: { color: 'inherit' },
  list: {
    width: '24em',
    maxHeight: '24em',
    overflowY: 'auto',
    padding: 0,
  },
  item: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    gap: 2,
    padding: theme.spacing(1),
    borderBottom: `1px solid ${theme.palette.divider}`,
  },
  title: {
    fontSize: '0.85rem',
    fontWeight: 600,
  },
  detail: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
  },
  empty: {
    padding: theme.spacing(1.5),
    fontSize: '0.85rem',
    color: theme.palette.text.disabled,
  },
  headerRow: {
    display: 'flex',
    alignItems: 'baseline',
    justifyContent: 'space-between',
  },
  seeAll: {
    padding: theme.spacing(1, 1, 0.5, 1),
    fontSize: '0.8rem',
  },
  groupHeader: {
    padding: theme.spacing(0.75, 1, 0.25, 1),
    fontSize: '0.7rem',
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: '0.04em',
    color: theme.palette.text.secondary,
    borderBottom: `1px solid ${theme.palette.divider}`,
  },
  header: {
    padding: theme.spacing(1, 1, 0.5, 1),
    fontWeight: 600,
    fontSize: '0.85rem',
  },
}))

const ReleaseAlertsPanel = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const [alerts, setAlerts] = useState([])
  const [unread, setUnread] = useState(0)
  // Bumped by SSE releaseAlert events
  const alertBump = useSelector((state) => state.activity.releaseAlerts)
  const open = Boolean(anchorEl)

  const fetchAlerts = useCallback(() => {
    httpClient(`/api/community/alerts?count=${MAX_FETCH}`)
      .then((resp) => {
        setAlerts(resp.json?.alerts || [])
        setUnread(resp.json?.unreadCount || 0)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    fetchAlerts()
  }, [fetchAlerts, alertBump])

  // Grouped against today's date on every render, so the list follows the
  // calendar instead of the server's fixed 14-day ordering.
  const sections = useMemo(() => {
    const groups = groupAlerts(alerts)
    let remaining = MAX_SHOWN
    return BELL_GROUPS.map((g) => {
      const items = groups[g.key].slice(0, Math.max(remaining, 0))
      remaining -= items.length
      return { key: g.key, items }
    }).filter((g) => g.items.length > 0)
  }, [alerts, anchorEl]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleOpen = (event) => {
    setAnchorEl(event.currentTarget)
    fetchAlerts()
    httpClient('/api/community/alerts/seen', { method: 'POST' })
      .then(() => setUnread(0))
      .catch(() => {})
  }

  return (
    <div>
      <Tooltip title={translate('releaseAlerts.title')}>
        <IconButton
          className={classes.button}
          onClick={handleOpen}
          aria-label={translate('releaseAlerts.title')}
        >
          <Badge badgeContent={unread} color="primary" overlap="rectangular">
            <NotificationsIcon style={{ fontSize: 20 }} />
          </Badge>
        </IconButton>
      </Tooltip>
      <Popover
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={open}
        onClose={() => setAnchorEl(null)}
      >
        <Card>
          <CardContent style={{ padding: 0 }}>
            <div className={classes.headerRow}>
              <Typography className={classes.header}>
                {translate('releaseAlerts.title')}
              </Typography>
              <Link
                to="/new-releases"
                className={classes.seeAll}
                onClick={() => setAnchorEl(null)}
              >
                {translate('releaseAlerts.seeAll')}
              </Link>
            </div>
            {sections.length === 0 ? (
              <Typography className={classes.empty}>
                {translate('releaseAlerts.empty')}
              </Typography>
            ) : (
              <List className={classes.list} dense>
                {sections.map((section) => (
                  <React.Fragment key={section.key}>
                    <li className={classes.groupHeader}>
                      {translate(`newReleases.groups.${section.key}`)}
                    </li>
                    {section.items.map((alert) => (
                      <ListItem key={alert.id} className={classes.item}>
                        <Typography className={classes.title}>
                          <Link
                            to={`/artist/${alert.artistId}/show`}
                            onClick={() => setAnchorEl(null)}
                          >
                            {alert.artistName}
                          </Link>
                          {' — '}
                          {alert.title}
                        </Typography>
                        <Typography className={classes.detail}>
                          {alert.releaseType} ·{' '}
                          {formatReleaseDate(alert.releaseDate)} ·{' '}
                          {alert.source}
                          {alert.externalUrl && (
                            <a
                              href={alert.externalUrl}
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              <LaunchIcon style={{ fontSize: 13 }} />
                            </a>
                          )}
                        </Typography>
                      </ListItem>
                    ))}
                  </React.Fragment>
                ))}
              </List>
            )}
          </CardContent>
        </Card>
      </Popover>
    </div>
  )
}

export default ReleaseAlertsPanel
