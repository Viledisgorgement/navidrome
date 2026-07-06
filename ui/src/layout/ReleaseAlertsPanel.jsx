import React, { useState, useEffect, useCallback } from 'react'
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
import config from '../config'

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
    httpClient('/api/community/alerts?count=50')
      .then((resp) => {
        setAlerts(resp.json?.alerts || [])
        setUnread(resp.json?.unreadCount || 0)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    fetchAlerts()
  }, [fetchAlerts, alertBump])

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
            <Typography className={classes.header}>
              {translate('releaseAlerts.title')}
            </Typography>
            {alerts.length === 0 ? (
              <Typography className={classes.empty}>
                {translate('releaseAlerts.empty')}
              </Typography>
            ) : (
              <List className={classes.list} dense>
                {alerts.map((alert) => (
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
                      {alert.releaseType} · {alert.releaseDate} · {alert.source}
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
              </List>
            )}
          </CardContent>
        </Card>
      </Popover>
    </div>
  )
}

export default ReleaseAlertsPanel
