import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, beforeEach, afterEach, vi } from 'vitest'

import ReleaseAlertsPanel from './ReleaseAlertsPanel'
import { activityReducer } from '../reducers'
import { httpClient } from '../dataProvider'

vi.mock('react-admin', () => ({
  useTranslate: () => (key) => key,
  // eslint-disable-next-line react/prop-types
  Link: ({ children }) => <span>{children}</span>,
}))

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

const alert = (id, releaseDate) => ({
  id,
  artistId: 'a',
  artistName: `Artist ${id}`,
  title: `Title ${id}`,
  releaseType: 'Full-length',
  releaseDate,
  source: 'musicbrainz',
})

describe('<ReleaseAlertsPanel />', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 10, 6, 12))
    httpClient.mockImplementation((url) =>
      Promise.resolve({
        json: url.includes('seen')
          ? {}
          : {
              // Server order: oldest of its 14-day window first
              alerts: [
                alert('old', '2026-10-23'),
                alert('week', '2026-11-03'),
                alert('today', '2026-11-06'),
                alert('soon', '2026-11-10'),
                alert('dec', '2026-12-04'),
              ],
              unreadCount: 0,
            },
      }),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('groups alerts by the current month and hides older months', async () => {
    const store = createStore(combineReducers({ activity: activityReducer }))
    render(
      <Provider store={store}>
        <MemoryRouter>
          <ReleaseAlertsPanel />
        </MemoryRouter>
      </Provider>,
    )
    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => screen.getByText(/Title today/))

    expect(screen.queryByText(/Title old/)).toBeNull()
    const text = document.body.textContent
    const order = [
      'newReleases.groups.today',
      'Title today',
      'newReleases.groups.thisWeek',
      'Title week',
      'newReleases.groups.nextWeek',
      'Title soon',
      'newReleases.groups.nextMonth',
      'Title dec',
    ].map((s) => text.indexOf(s))
    order.forEach((pos) => expect(pos).toBeGreaterThan(-1))
    expect([...order].sort((a, b) => a - b)).toEqual(order)
  })
})
