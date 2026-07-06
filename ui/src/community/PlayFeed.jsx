import React from 'react'
import PropTypes from 'prop-types'
import { useTranslate, Link } from 'react-admin'
import {
  Card,
  CardContent,
  Typography,
  List,
  ListItem,
  Avatar,
  Button,
  makeStyles,
} from '@material-ui/core'
import subsonic from '../subsonic'

const useStyles = makeStyles((theme) => ({
  title: {
    fontWeight: 600,
    fontSize: '0.9rem',
    marginBottom: theme.spacing(1),
  },
  item: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(0.75, 0),
  },
  avatar: {
    width: theme.spacing(5),
    height: theme.spacing(5),
    borderRadius: theme.spacing(0.5),
  },
  content: { flex: 1, minWidth: 0 },
  trackTitle: {
    fontSize: '0.85rem',
    fontWeight: 500,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  detail: {
    fontSize: '0.7rem',
    color: theme.palette.text.secondary,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  meta: {
    fontSize: '0.7rem',
    color: theme.palette.text.disabled,
    flexShrink: 0,
    textAlign: 'right',
  },
  empty: {
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
  },
  loadMore: {
    marginTop: theme.spacing(1),
  },
}))

const formatTime = (date) => {
  const d = new Date(date)
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  return sameDay
    ? d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

const PlayFeed = ({ entries, hasMore, onLoadMore }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <Card>
      <CardContent>
        <Typography className={classes.title}>
          {translate('community.feedTitle')}
        </Typography>
        {entries.length === 0 ? (
          <Typography className={classes.empty}>
            {translate('community.noPlays')}
          </Typography>
        ) : (
          <List dense disablePadding>
            {entries.map((entry, idx) => (
              <ListItem
                key={`${entry.userId}-${entry.mediaFileId}-${entry.submissionTime}-${idx}`}
                className={classes.item}
                disableGutters
              >
                <Link to={`/album/${entry.albumId}/show`}>
                  <Avatar
                    className={classes.avatar}
                    variant="square"
                    src={subsonic.getCoverArtUrl(
                      { id: entry.mediaFileId, album: entry.album },
                      80,
                    )}
                    alt={entry.album}
                    loading="lazy"
                  />
                </Link>
                <div className={classes.content}>
                  <Typography
                    className={classes.trackTitle}
                    title={entry.title}
                  >
                    {entry.title}
                  </Typography>
                  <Typography className={classes.detail}>
                    {entry.artist} · {entry.album}
                  </Typography>
                </div>
                <div className={classes.meta}>
                  <div>{entry.userName}</div>
                  <div>{formatTime(entry.submissionTime)}</div>
                </div>
              </ListItem>
            ))}
          </List>
        )}
        {hasMore && (
          <Button
            className={classes.loadMore}
            size="small"
            onClick={onLoadMore}
            fullWidth
          >
            {translate('community.loadMore')}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

PlayFeed.propTypes = {
  entries: PropTypes.array.isRequired,
  hasMore: PropTypes.bool.isRequired,
  onLoadMore: PropTypes.func.isRequired,
}

export default PlayFeed
