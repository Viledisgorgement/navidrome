import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-router-dom'
import { useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import albumLists from './albumLists'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      display: 'flex',
      flexWrap: 'wrap',
      alignItems: 'center',
      padding: theme.spacing(1.5, 2, 0, 2),
      fontSize: '0.875rem',
    },
    link: {
      color: theme.palette.text.secondary,
      textDecoration: 'none',
      padding: theme.spacing(0.25, 0.5),
      borderBottom: '2px solid transparent',
      whiteSpace: 'nowrap',
      '&:hover': {
        color: theme.palette.text.primary,
      },
    },
    active: {
      color: theme.palette.primary.main,
      borderBottomColor: theme.palette.primary.main,
    },
    separator: {
      color: theme.palette.divider,
      padding: theme.spacing(0, 0.75),
      userSelect: 'none',
    },
  }),
  { name: 'NDAlbumListNav' },
)

// Airsonic-style horizontal navigation between the album list types,
// replacing the sidebar submenu
const AlbumListNav = ({ current }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const types = Object.keys(albumLists)

  return (
    <nav className={classes.root}>
      {types.map((type, idx) => (
        <React.Fragment key={type}>
          {idx > 0 && <span className={classes.separator}>|</span>}
          <Link
            className={clsx(classes.link, current === type && classes.active)}
            to={`/album/${type}?${albumLists[type].params}`}
          >
            {translate(`resources.album.lists.${type}`, { smart_count: 2 })}
          </Link>
        </React.Fragment>
      ))}
    </nav>
  )
}

AlbumListNav.propTypes = {
  current: PropTypes.string,
}

export default AlbumListNav
