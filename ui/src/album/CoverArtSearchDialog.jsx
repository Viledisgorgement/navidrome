import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Card,
  CardActionArea,
  CardContent,
  CardMedia,
  CircularProgress,
  Dialog,
  DialogContent,
  DialogTitle,
  Grid,
  TextField,
  Typography,
  makeStyles,
} from '@material-ui/core'
import { useNotify, useRefresh, useTranslate } from 'react-admin'
import { REST_URL } from '../consts'
import { httpClient } from '../dataProvider'

const useStyles = makeStyles((theme) => ({
  form: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
    alignItems: 'center',
    marginBottom: theme.spacing(2),
  },
  field: {
    flex: '1 1 14em',
  },
  status: {
    display: 'flex',
    justifyContent: 'center',
    padding: theme.spacing(3),
  },
  card: {
    height: '100%',
  },
  media: {
    paddingTop: '100%',
    backgroundColor: theme.palette.action.hover,
  },
  caption: {
    padding: theme.spacing(1),
    '&:last-child': {
      paddingBottom: theme.spacing(1),
    },
  },
}))

const albumArtist = (record) =>
  record.albumArtist || record.participants?.albumartist?.[0]?.name || ''

const CoverCandidate = ({ candidate, onPick, disabled }) => {
  const classes = useStyles()
  const [broken, setBroken] = useState(false)
  // Not every Cover Art Archive match has art; hide those that fail to load
  if (broken) return null
  return (
    <Grid item xs={6} sm={4} md={3}>
      <Card className={classes.card}>
        <CardActionArea
          onClick={() => onPick(candidate.url)}
          disabled={disabled}
        >
          <CardMedia
            className={classes.media}
            image={candidate.thumbnail}
            title={`${candidate.artist} - ${candidate.album}`}
          />
          <img
            src={candidate.thumbnail}
            alt=""
            style={{ display: 'none' }}
            onError={() => setBroken(true)}
          />
          <CardContent className={classes.caption}>
            <Typography variant="body2" noWrap>
              {candidate.artist}
            </Typography>
            <Typography variant="body2" color="textSecondary" noWrap>
              {candidate.album}
            </Typography>
            <Typography variant="caption" color="textSecondary">
              {candidate.source}
            </Typography>
          </CardContent>
        </CardActionArea>
      </Card>
    </Grid>
  )
}

export const CoverArtSearchDialog = ({ record, open, onClose }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const [artist, setArtist] = useState('')
  const [album, setAlbum] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [results, setResults] = useState(null)
  const [searching, setSearching] = useState(false)
  const [saving, setSaving] = useState(false)

  const search = useCallback(
    (artistName, albumName) => {
      if (!albumName) return
      setSearching(true)
      setResults(null)
      const params = new URLSearchParams({
        artist: artistName,
        album: albumName,
      })
      httpClient(`${REST_URL}/album/${record.id}/cover/search?${params}`)
        .then(({ json }) => setResults(json))
        .catch(() => {
          setResults([])
          notify('message.coverSearchError', 'warning')
        })
        .finally(() => setSearching(false))
    },
    [record.id, notify],
  )

  // Pre-fill and search right away, like Airsonic
  useEffect(() => {
    if (!open) return
    const a = albumArtist(record)
    setArtist(a)
    setAlbum(record.name)
    setImageUrl('')
    search(a, record.name)
  }, [open, record, search])

  const save = useCallback(
    (url) => {
      if (!url) return
      setSaving(true)
      httpClient(`${REST_URL}/album/${record.id}/cover`, {
        method: 'POST',
        body: JSON.stringify({ url }),
      })
        .then(({ json }) => {
          notify(
            json.savedTo === 'override'
              ? 'message.coverSavedAsOverride'
              : 'message.coverSavedToFolder',
            json.savedTo === 'override' ? 'warning' : 'success',
          )
          refresh()
          onClose()
        })
        .catch(() => {
          notify('message.coverUploadError', 'warning')
        })
        .finally(() => setSaving(false))
    },
    [record.id, notify, refresh, onClose],
  )

  const onSearch = (e) => {
    e.preventDefault()
    search(artist, album)
  }
  const onUseUrl = (e) => {
    e.preventDefault()
    save(imageUrl.trim())
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{translate('message.changeCover')}</DialogTitle>
      <DialogContent>
        <form className={classes.form} onSubmit={onSearch}>
          <TextField
            className={classes.field}
            label={translate('resources.album.fields.albumArtist')}
            value={artist}
            onChange={(e) => setArtist(e.target.value)}
            size="small"
            variant="outlined"
          />
          <TextField
            className={classes.field}
            label={translate('resources.album.name', { smart_count: 1 })}
            value={album}
            onChange={(e) => setAlbum(e.target.value)}
            size="small"
            variant="outlined"
          />
          <Button
            type="submit"
            variant="contained"
            color="primary"
            disabled={searching || saving || !album}
          >
            {translate('message.coverSearch')}
          </Button>
        </form>
        <form className={classes.form} onSubmit={onUseUrl}>
          <TextField
            className={classes.field}
            label={translate('message.coverImageAddress')}
            placeholder="https://"
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
            size="small"
            variant="outlined"
          />
          <Button
            type="submit"
            variant="outlined"
            disabled={saving || !imageUrl.trim()}
          >
            {translate('ra.action.save')}
          </Button>
        </form>
        {(searching || saving) && (
          <div className={classes.status}>
            <CircularProgress />
          </div>
        )}
        {!searching && results && results.length === 0 && (
          <Typography className={classes.status} color="textSecondary">
            {translate('message.coverNoResults')}
          </Typography>
        )}
        {!searching && results && results.length > 0 && (
          <Grid container spacing={2}>
            {results.map((c) => (
              <CoverCandidate
                key={c.url}
                candidate={c}
                onPick={save}
                disabled={saving}
              />
            ))}
          </Grid>
        )}
      </DialogContent>
    </Dialog>
  )
}

export default CoverArtSearchDialog
