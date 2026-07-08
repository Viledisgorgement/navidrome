import React, { useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Layout as RALayout, toggleSidebar } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { HotKeys } from 'react-hotkeys'
import Menu from './Menu'
import AppBar from './AppBar'
import Notification from './Notification'
import CommunitySidebar from './CommunitySidebar'
import useCurrentTheme from '../themes/useCurrentTheme'
import { useSearchRefocus } from '../common'
import config from '../config'

const useStyles = makeStyles({
  root: { paddingBottom: (props) => (props.addPadding ? '80px' : 0) },
  // The community sidebar is a flex sibling of the app, so the main
  // content shrinks to make room instead of being overlaid
  flexWrapper: {
    display: 'flex',
    alignItems: 'stretch',
  },
  main: {
    flex: 1,
    minWidth: 0,
  },
})

const Layout = (props) => {
  const theme = useCurrentTheme()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const dispatch = useDispatch()
  useSearchRefocus()

  const keyHandlers = {
    TOGGLE_MENU: useCallback(() => dispatch(toggleSidebar()), [dispatch]),
  }

  return (
    <HotKeys handlers={keyHandlers}>
      <div className={classes.flexWrapper}>
        <div className={classes.main}>
          <RALayout
            {...props}
            className={classes.root}
            menu={Menu}
            appBar={AppBar}
            theme={theme}
            notification={Notification}
          />
        </div>
        {config.enableCommunity && <CommunitySidebar />}
      </div>
    </HotKeys>
  )
}

export default Layout
