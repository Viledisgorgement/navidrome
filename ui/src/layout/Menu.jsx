import React, { useState } from 'react'
import { useSelector } from 'react-redux'
import { Divider, makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import { useTranslate, MenuItemLink, getResources } from 'react-admin'
import ViewListIcon from '@material-ui/icons/ViewList'
import AlbumIcon from '@material-ui/icons/Album'
import PeopleIcon from '@material-ui/icons/People'
import { humanize, pluralize } from 'inflection'
import { defaultAlbumList } from '../album/albumLists'
import PlaylistsSubMenu from './PlaylistsSubMenu'
import LibrarySelector from '../common/LibrarySelector'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
    paddingBottom: (props) => (props.addPadding ? '80px' : '20px'),
  },
  open: {
    width: 240,
  },
  closed: {
    width: 55,
  },
  active: {
    color: theme.palette.text.primary,
    fontWeight: 'bold',
  },
}))

const translatedResourceName = (resource, translate) =>
  translate(`resources.${resource.name}.name`, {
    smart_count: 2,
    _:
      resource.options && resource.options.label
        ? translate(resource.options.label, {
            smart_count: 2,
            _: resource.options.label,
          })
        : humanize(pluralize(resource.name)),
  })

const Menu = ({ dense = false }) => {
  const open = useSelector((state) => state.admin.ui.sidebarOpen)
  const translate = useTranslate()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const resources = useSelector(getResources)

  // TODO State is not persisted in mobile when you close the sidebar menu. Move to redux?
  const [state, setState] = useState({
    menuPlaylists: true,
    menuSharedPlaylists: true,
  })

  const renderResourceMenuItemLink = (resource) => (
    <MenuItemLink
      key={resource.name}
      to={`/${resource.name}`}
      activeClassName={classes.active}
      primaryText={translatedResourceName(resource, translate)}
      leftIcon={resource.icon || <ViewListIcon />}
      sidebarIsOpen={open}
      dense={dense}
    />
  )

  const subItems = (subMenu) => (resource) =>
    resource.hasList && resource.options && resource.options.subMenu === subMenu

  return (
    <div
      className={clsx(classes.root, {
        [classes.open]: open,
        [classes.closed]: !open,
      })}
    >
      {open && <LibrarySelector />}
      {/* Album list types moved to the Airsonic-style nav at the top of
          the album page; the sidebar keeps a single entry point */}
      <MenuItemLink
        to={`/album/${defaultAlbumList}`}
        activeClassName={classes.active}
        primaryText={translate('menu.albumList')}
        leftIcon={<AlbumIcon />}
        sidebarIsOpen={open}
        dense={dense}
      />
      {resources.filter(subItems(undefined)).map(renderResourceMenuItemLink)}
      {config.enableCommunity && (
        <MenuItemLink
          to="/community"
          activeClassName={classes.active}
          primaryText={translate('community.title', { _: 'Community' })}
          leftIcon={<PeopleIcon />}
          sidebarIsOpen={open}
          dense={dense}
          exact
        />
      )}
      {config.devSidebarPlaylists && open ? (
        <>
          <Divider />
          <PlaylistsSubMenu
            state={state}
            setState={setState}
            sidebarIsOpen={open}
            dense={dense}
          />
        </>
      ) : (
        resources.filter(subItems('playlist')).map(renderResourceMenuItemLink)
      )}
    </div>
  )
}

export default Menu
