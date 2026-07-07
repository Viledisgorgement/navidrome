import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { activityReducer, settingsReducer } from '../reducers'
import AppBar from './AppBar'
import config from '../config'

let store

vi.mock('react-admin', () => ({
  AppBar: ({ userMenu }) => <div data-testid="appbar">{userMenu}</div>,
  useTranslate: () => (x) => x,
  usePermissions: () => ({ permissions: 'admin' }),
  getResources: () => [],
}))

vi.mock('./NowPlayingPanel', () => ({
  default: () => <div data-testid="now-playing-panel" />,
}))
vi.mock('./ActivityPanel', () => ({
  default: () => <div data-testid="activity-panel" />,
}))
vi.mock('./ReleaseAlertsPanel', () => ({
  default: () => <div data-testid="release-alerts-panel" />,
}))
vi.mock('./PersonalMenu', () => ({
  default: () => <div />,
}))
vi.mock('./UserMenu', () => ({
  default: ({ children }) => <div>{children}</div>,
}))
vi.mock('../dialogs/Dialogs', () => ({
  Dialogs: () => <div />,
}))
vi.mock('../dialogs', () => ({
  AboutDialog: () => <div />,
}))

describe('<AppBar />', () => {
  beforeEach(() => {
    config.devActivityPanel = true
    config.enableNowPlaying = true
    config.enableCommunity = true
    config.enableReleaseAlerts = true
    store = createStore(
      combineReducers({
        activity: activityReducer,
        settings: settingsReducer,
      }),
      { activity: { nowPlayingCount: 0 } },
    )
  })

  it('renders NowPlayingPanel when enabled', () => {
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
  })

  it('hides NowPlayingPanel when disabled', () => {
    config.enableNowPlaying = false
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.queryByTestId('now-playing-panel')).toBeNull()
  })

  it('renders the community sidebar toggle and release alerts bell', () => {
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(
      screen.getByRole('button', { name: 'community.sidebarTitle' }),
    ).toBeInTheDocument()
    expect(screen.getByTestId('release-alerts-panel')).toBeInTheDocument()
  })

  it('hides community elements when disabled', () => {
    config.enableCommunity = false
    config.enableReleaseAlerts = false
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(
      screen.queryByRole('button', { name: 'community.sidebarTitle' }),
    ).toBeNull()
    expect(screen.queryByTestId('release-alerts-panel')).toBeNull()
  })
})
