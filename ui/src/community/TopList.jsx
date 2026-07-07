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
  makeStyles,
} from '@material-ui/core'
import subsonic from '../subsonic'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  card: { height: '100%' },
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
  rank: {
    width: '1.5em',
    textAlign: 'right',
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
    fontVariantNumeric: 'tabular-nums',
    flexShrink: 0,
  },
  avatar: {
    width: theme.spacing(5),
    height: theme.spacing(5),
    borderRadius: theme.spacing(0.5),
  },
  content: { flex: 1, minWidth: 0 },
  name: {
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
  plays: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    flexShrink: 0,
    fontVariantNumeric: 'tabular-nums',
  },
  empty: {
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
  },
}))

const entryLink = (kind, entry) => {
  if (kind === 'artist') {
    if (!entry.id) return null
    return config.devShowArtistPage && entry.id !== config.variousArtistsId
      ? `/artist/${entry.id}/show`
      : null
  }
  const albumId = kind === 'album' ? entry.id : entry.albumId
  return albumId ? `/album/${albumId}/show` : null
}

const coverUrl = (kind, entry) => {
  const albumId = kind === 'album' ? entry.id : entry.albumId
  if (!albumId) return null
  // Force the "al-" cover art prefix used for albums
  return subsonic.getCoverArtUrl({ id: albumId, albumArtist: ' ' }, 80)
}

const TopList = ({ kind, entries, titleKey }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <Card className={classes.card}>
      <CardContent>
        <Typography className={classes.title}>{translate(titleKey)}</Typography>
        {entries.length === 0 ? (
          <Typography className={classes.empty}>
            {translate('community.noPlays')}
          </Typography>
        ) : (
          <List dense disablePadding>
            {entries.map((entry, i) => {
              const link = entryLink(kind, entry)
              const cover = coverUrl(kind, entry)
              const name = (
                <Typography className={classes.name} title={entry.name}>
                  {entry.name}
                </Typography>
              )
              return (
                <ListItem
                  key={entry.id || i}
                  className={classes.item}
                  disableGutters
                >
                  <span className={classes.rank}>{i + 1}</span>
                  {cover && (
                    <Avatar
                      className={classes.avatar}
                      variant="square"
                      src={cover}
                      alt={entry.name}
                      loading="lazy"
                    />
                  )}
                  <div className={classes.content}>
                    {link ? <Link to={link}>{name}</Link> : name}
                    {entry.artist && (
                      <Typography
                        className={classes.detail}
                        title={entry.artist}
                      >
                        {entry.artist}
                      </Typography>
                    )}
                  </div>
                  <span className={classes.plays}>
                    {translate('community.plays', {
                      smart_count: entry.playCount,
                    })}
                  </span>
                </ListItem>
              )
            })}
          </List>
        )}
      </CardContent>
    </Card>
  )
}

TopList.propTypes = {
  kind: PropTypes.oneOf(['song', 'album', 'artist']).isRequired,
  entries: PropTypes.array.isRequired,
  titleKey: PropTypes.string.isRequired,
}

export default TopList
