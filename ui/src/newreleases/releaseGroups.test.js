import { describe, it, expect } from 'vitest'
import { groupFor, groupAlerts } from './releaseGroups'

const now = new Date(2026, 9, 24, 15, 0) // Oct 24 2026

describe('groupFor', () => {
  it.each([
    ['2026-10-24', 'today'],
    ['2026-10-20', 'thisWeek'],
    ['2026-10-01', 'thisMonth'],
    ['2026-08-01', 'earlier'],
    ['2026-09-30', 'earlier'],
    ['2026-10-28', 'nextWeek'],
    ['2026-10-31', 'nextWeek'],
    ['2026-11-14', 'nextMonth'],
    ['2026-11-30', 'nextMonth'],
    ['2026-12-05', 'later'],
    ['2026-10', 'restOfMonth'],
    ['2026-11', 'nextMonth'],
    ['2026-12', 'later'],
    ['2026-09', 'earlier'],
    ['TBA', 'tba'],
    ['', 'tba'],
  ])('%s -> %s', (date, group) => {
    expect(groupFor(date, now)).toBe(group)
  })
})

describe('groupAlerts', () => {
  it('lists upcoming soonest first and released newest first', () => {
    const alerts = [
      { id: 'a', releaseDate: '2026-12' },
      { id: 'b', releaseDate: '2026-12-01' },
      { id: 'c', releaseDate: '2027-01-10' },
      { id: 'd', releaseDate: '2026-05-01' },
      { id: 'e', releaseDate: '2026-07-01' },
    ]
    const groups = groupAlerts(alerts, now)
    expect(groups.later.map((a) => a.id)).toEqual(['b', 'a', 'c'])
    expect(groups.nextMonth).toEqual([])
    expect(groups.earlier.map((a) => a.id)).toEqual(['e', 'd'])
  })
})
