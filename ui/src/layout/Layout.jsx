import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Layout as RALayout, toggleSidebar } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { useTheme, useMediaQuery } from '@material-ui/core'
import { HotKeys } from 'react-hotkeys'
import Menu from './Menu'
import AppBar from './AppBar'
import Notification from './Notification'
import CommunitySidebar, { COMMUNITY_SIDEBAR_WIDTH } from './CommunitySidebar'
import useCurrentTheme from '../themes/useCurrentTheme'
import { useSearchRefocus } from '../common'
import config from '../config'

const useStyles = makeStyles({
  root: {
    paddingBottom: (props) => (props.addPadding ? '80px' : 0),
    paddingRight: (props) => (props.sidebarOpen ? COMMUNITY_SIDEBAR_WIDTH : 0),
  },
})

const Layout = (props) => {
  const theme = useCurrentTheme()
  const queue = useSelector((state) => state.player?.queue)
  const communitySidebarOpen = useSelector(
    (state) => state.settings.communitySidebarOpen,
  )
  const muiTheme = useTheme()
  const isSmallScreen = useMediaQuery(muiTheme.breakpoints.down('md'))
  const classes = useStyles({
    addPadding: queue.length > 0,
    sidebarOpen:
      config.enableCommunity && communitySidebarOpen && !isSmallScreen,
  })
  const dispatch = useDispatch()
  useSearchRefocus()

  const keyHandlers = {
    TOGGLE_MENU: useCallback(() => dispatch(toggleSidebar()), [dispatch]),
  }

  return (
    <HotKeys handlers={keyHandlers}>
      <RALayout
        {...props}
        className={classes.root}
        menu={Menu}
        appBar={AppBar}
        theme={theme}
        notification={Notification}
      />
      {config.enableCommunity && <CommunitySidebar />}
    </HotKeys>
  )
}

export default Layout
