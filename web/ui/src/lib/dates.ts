// formatDate shows an instant from the API, such as "2026-01-31T10:00:00Z",
// as a date, or with its time too when withTime is set, in the reader's
// locale and time zone unless they are given.
export function formatDate(instant: string, withTime = false, locale?: string, timeZone?: string): string {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    ...(withTime ? { timeStyle: "short" } : {}),
    timeZone,
  }).format(new Date(instant))
}

// localDate is the date of now in the reader's time zone, as the API takes a
// date, such as "2026-10-09": toISOString would give the UTC one, a day off
// around midnight.
export function localDate(now: Date): string {
  const month = String(now.getMonth() + 1).padStart(2, "0")
  const day = String(now.getDate()).padStart(2, "0")
  return `${now.getFullYear()}-${month}-${day}`
}
