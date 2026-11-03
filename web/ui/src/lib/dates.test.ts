import { describe, expect, it } from "vitest"

import { formatDate, localDate } from "./dates"

describe("formatDate", () => {
  const cases: { name: string; instant: string; withTime: boolean; timeZone: string; want: string }[] = [
    { name: "a date", instant: "2026-01-31T10:00:00Z", withTime: false, timeZone: "UTC", want: "Jan 31, 2026" },
    { name: "a date with its time", instant: "2026-01-31T10:00:00Z", withTime: true, timeZone: "UTC", want: "Jan 31, 2026, 10:00 AM" },
    {
      name: "the date in the reader's time zone, not UTC's",
      instant: "2026-01-31T23:30:00Z",
      withTime: false,
      timeZone: "Europe/Rome",
      want: "Feb 1, 2026",
    },
  ]

  for (const { name, instant, withTime, timeZone, want } of cases) {
    it(name, () => {
      expect(formatDate(instant, withTime, "en-US", timeZone)).toBe(want)
    })
  }
})

describe("localDate", () => {
  // Built from local fields, so each case means the same in any time zone.
  const cases: { name: string; now: Date; want: string }[] = [
    { name: "a single-digit month and day", now: new Date(2026, 0, 5, 12), want: "2026-01-05" },
    { name: "just before midnight", now: new Date(2026, 9, 9, 23, 59), want: "2026-10-09" },
    { name: "just after midnight", now: new Date(2026, 9, 10, 0, 1), want: "2026-10-10" },
  ]

  for (const { name, now, want } of cases) {
    it(name, () => {
      expect(localDate(now)).toBe(want)
    })
  }
})
