// Groups release alerts by how their date relates to today. Alerts store
// ISO dates (YYYY-MM-DD or YYYY-MM); month-only dates are bucketed by month.

const DAY = 24 * 60 * 60 * 1000

export const GROUPS = [
  { key: 'today', upcoming: false },
  { key: 'thisWeek', upcoming: false },
  { key: 'thisMonth', upcoming: false },
  { key: 'earlier', upcoming: false },
  { key: 'nextWeek', upcoming: true },
  { key: 'restOfMonth', upcoming: true },
  { key: 'nextMonth', upcoming: true },
  { key: 'later', upcoming: true },
  { key: 'tba', upcoming: true },
]

const startOfDay = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate())

// Returns { day, monthOnly } where day is a local Date, or null when the
// date can't be parsed.
export const parseReleaseDate = (date) => {
  const m = /^(\d{4})-(\d{2})(?:-(\d{2}))?$/.exec(date || '')
  if (!m) return null
  const [, year, month, day] = m
  return {
    day: new Date(Number(year), Number(month) - 1, Number(day || 1)),
    monthOnly: !day,
  }
}

// Months from today's month to the date's month (0 = this month)
const monthDiff = (day, today) =>
  (day.getFullYear() - today.getFullYear()) * 12 +
  day.getMonth() -
  today.getMonth()

// The week groups are rolling 7 days; the month groups are calendar months.
export const groupFor = (date, now = new Date()) => {
  const parsed = parseReleaseDate(date)
  if (!parsed) return 'tba'
  const today = startOfDay(now)
  const months = monthDiff(parsed.day, today)
  if (parsed.monthOnly) {
    // "sometime this month" is most likely still to come
    if (months === 0) return 'restOfMonth'
    if (months === 1) return 'nextMonth'
    return months > 0 ? 'later' : 'earlier'
  }
  const days = Math.round((parsed.day - today) / DAY)
  if (days === 0) return 'today'
  if (days < 0) {
    if (days >= -7) return 'thisWeek'
    return months === 0 ? 'thisMonth' : 'earlier'
  }
  if (days <= 7) return 'nextWeek'
  if (months === 0) return 'restOfMonth'
  if (months === 1) return 'nextMonth'
  return 'later'
}

// Released groups list newest first; upcoming groups list soonest first.
// Month-only dates sort after full dates in the same month.
const sortKey = (alert) => {
  const parsed = parseReleaseDate(alert.releaseDate)
  if (!parsed) return Number.MAX_SAFE_INTEGER
  return parsed.day.getTime() + (parsed.monthOnly ? 31 * DAY - 1 : 0)
}

export const groupAlerts = (alerts, now = new Date()) => {
  const byGroup = Object.fromEntries(GROUPS.map((g) => [g.key, []]))
  alerts.forEach((alert) =>
    byGroup[groupFor(alert.releaseDate, now)].push(alert),
  )
  GROUPS.forEach(({ key, upcoming }) => {
    byGroup[key].sort((a, b) =>
      upcoming ? sortKey(a) - sortKey(b) : sortKey(b) - sortKey(a),
    )
  })
  return byGroup
}
