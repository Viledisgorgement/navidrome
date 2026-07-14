import React, { useState } from 'react'
import { useHistory } from 'react-router-dom'
import { useTranslate } from 'react-admin'
import { InputBase, makeStyles } from '@material-ui/core'
import { alpha } from '@material-ui/core/styles'
import SearchIcon from '@material-ui/icons/Search'

const useStyles = makeStyles(
  (theme) => ({
    search: {
      position: 'relative',
      borderRadius: theme.shape.borderRadius,
      backgroundColor: alpha(theme.palette.common.white, 0.15),
      '&:hover': {
        backgroundColor: alpha(theme.palette.common.white, 0.25),
      },
      marginLeft: theme.spacing(1),
      marginRight: theme.spacing(1),
    },
    searchIcon: {
      padding: theme.spacing(0, 1.5),
      height: '100%',
      position: 'absolute',
      pointerEvents: 'none',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
    inputRoot: {
      color: 'inherit',
    },
    inputInput: {
      padding: theme.spacing(1, 1, 1, 0),
      paddingLeft: `calc(1em + ${theme.spacing(3)}px)`,
      transition: theme.transitions.create('width'),
      width: '14ch',
      '&:focus': {
        width: '22ch',
      },
      [theme.breakpoints.down('xs')]: {
        width: '8ch',
      },
    },
  }),
  { name: 'NDAppBarSearch' },
)

// Global search in the app bar: searches ARTISTS (submit with Enter).
// The album list keeps its own album search box below.
const AppBarSearch = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const history = useHistory()
  const [query, setQuery] = useState('')

  const handleSubmit = (event) => {
    event.preventDefault()
    const q = query.trim()
    if (!q) return
    history.push(
      `/artist?filter=${encodeURIComponent(JSON.stringify({ name: q }))}`,
    )
  }

  return (
    <form className={classes.search} onSubmit={handleSubmit} role="search">
      <div className={classes.searchIcon}>
        <SearchIcon />
      </div>
      <InputBase
        placeholder={translate('search.artists')}
        classes={{ root: classes.inputRoot, input: classes.inputInput }}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        inputProps={{ 'aria-label': translate('search.artists') }}
      />
    </form>
  )
}

export default AppBarSearch
