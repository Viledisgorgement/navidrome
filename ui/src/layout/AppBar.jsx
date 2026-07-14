import React, { createElement, forwardRef, Fragment } from 'react'
import {
  AppBar as RAAppBar,
  MenuItemLink,
  useTranslate,
  usePermissions,
  getResources,
} from 'react-admin'
import { MdInfo, MdPerson, MdSupervisorAccount, MdGroups } from 'react-icons/md'
import { useSelector, useDispatch } from 'react-redux'
import { Link } from 'react-router-dom'
import {
  makeStyles,
  MenuItem,
  ListItemIcon,
  Divider,
  IconButton,
  Tooltip,
  Typography,
} from '@material-ui/core'
import ViewListIcon from '@material-ui/icons/ViewList'
import HomeIcon from '@material-ui/icons/Home'
import { Dialogs } from '../dialogs/Dialogs'
import { AboutDialog } from '../dialogs'
import PersonalMenu from './PersonalMenu'
import ActivityPanel from './ActivityPanel'
import NowPlayingPanel from './NowPlayingPanel'
import ReleaseAlertsPanel from './ReleaseAlertsPanel'
import UserMenu from './UserMenu'
import AppBarSearch from './AppBarSearch'
import { toggleCommunitySidebar } from '../actions'
import albumLists from '../album/albumLists'
import config from '../config'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      color: theme.palette.text.secondary,
    },
    active: {
      color: theme.palette.text.primary,
    },
    icon: { minWidth: theme.spacing(5) },
  }),
  {
    name: 'NDAppBar',
  },
)

const AboutMenuItem = forwardRef(({ onClick, ...rest }, ref) => {
  const classes = useStyles(rest)
  const translate = useTranslate()
  const [open, setOpen] = React.useState(false)

  const handleOpen = () => {
    setOpen(true)
  }
  const handleClose = () => {
    onClick && onClick()
    setOpen(false)
  }
  const label = translate('menu.about')
  return (
    <>
      <MenuItem ref={ref} onClick={handleOpen} className={classes.root}>
        <ListItemIcon className={classes.icon}>
          <MdInfo title={label} size={24} />
        </ListItemIcon>
        {label}
      </MenuItem>
      <AboutDialog onClose={handleClose} open={open} />
    </>
  )
})

AboutMenuItem.displayName = 'AboutMenuItem'

const CommunitySidebarToggle = () => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const open = useSelector((state) => state.settings.communitySidebarOpen)
  return (
    <Tooltip title={translate('community.sidebarTitle')}>
      <IconButton
        color="inherit"
        aria-label={translate('community.sidebarTitle')}
        aria-pressed={open}
        onClick={() => dispatch(toggleCommunitySidebar())}
      >
        <MdGroups size={22} />
      </IconButton>
    </Tooltip>
  )
}

const settingsResources = (resource) =>
  resource.name !== 'user' &&
  resource.hasList &&
  resource.options &&
  resource.options.subMenu === 'settings'

const CustomUserMenu = ({ onClick, ...rest }) => {
  const translate = useTranslate()
  const resources = useSelector(getResources)
  const classes = useStyles(rest)
  const { permissions } = usePermissions()

  const resourceDefinition = (resourceName) =>
    resources.find((r) => r?.name === resourceName)

  const renderUserMenuItemLink = () => {
    const userResource = resourceDefinition('user')
    if (!userResource) {
      return null
    }
    if (permissions !== 'admin') {
      if (!config.enableUserEditing) {
        return null
      }
      userResource.icon = MdPerson
    } else {
      userResource.icon = MdSupervisorAccount
    }
    return renderSettingsMenuItemLink(
      userResource,
      permissions !== 'admin' ? localStorage.getItem('userId') : null,
    )
  }

  const renderSettingsMenuItemLink = (resource, id) => {
    const label = translate(`resources.${resource.name}.name`, {
      smart_count: id ? 1 : 2,
    })
    const link = id ? `/${resource.name}/${id}` : `/${resource.name}`
    return (
      <MenuItemLink
        className={classes.root}
        activeClassName={classes.active}
        key={resource.name}
        to={link}
        primaryText={label}
        leftIcon={
          (resource.icon && createElement(resource.icon, { size: 24 })) || (
            <ViewListIcon />
          )
        }
        onClick={onClick}
        sidebarIsOpen={true}
      />
    )
  }

  return (
    <>
      {config.enableCommunity && <CommunitySidebarToggle />}
      {config.enableReleaseAlerts && <ReleaseAlertsPanel />}
      {config.enableNowPlaying && <NowPlayingPanel />}
      {config.devActivityPanel && permissions === 'admin' && <ActivityPanel />}
      <UserMenu {...rest}>
        <PersonalMenu sidebarIsOpen={true} onClick={onClick} />
        <Divider />
        {renderUserMenuItemLink()}
        {resources
          .filter(settingsResources)
          .map((r) => renderSettingsMenuItemLink(r))}
        <Divider />
        <AboutMenuItem />
      </UserMenu>
      <Dialogs />
    </>
  )
}

// Home goes to Recently Added — the preferred landing view
const HomeButton = () => {
  const translate = useTranslate()
  return (
    <Tooltip title={translate('ra.page.dashboard', { _: 'Home' })}>
      <IconButton
        color="inherit"
        component={Link}
        to={`/album/recentlyAdded?${albumLists.recentlyAdded.params}`}
        aria-label="home"
      >
        <HomeIcon />
      </IconButton>
    </Tooltip>
  )
}

const AppBar = (props) => (
  <RAAppBar {...props} container={Fragment} userMenu={<CustomUserMenu />}>
    <HomeButton />
    {/* Children replace react-admin's default title, so re-add it */}
    <Typography
      variant="h6"
      color="inherit"
      id="react-admin-title"
      style={{
        flex: 1,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}
    />
    <AppBarSearch />
  </RAAppBar>
)

export default AppBar
