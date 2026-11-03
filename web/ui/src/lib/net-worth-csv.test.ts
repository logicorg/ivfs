import { describe, expect, it } from "vitest"

import { netWorthCsv, netWorthCsvName } from "./net-worth-csv"
import type { NetWorthPoint } from "./net-worth-series"

describe("netWorthCsv", () => {
  const header = "date,balances,investments,liabilities,net_worth,currency,flags\r\n"
  const cases: { name: string; points: NetWorthPoint[]; want: string }[] = [
    { name: "no dates, the header alone", points: [], want: header },
    {
      name: "exact amounts, oldest first",
      points: [
        { date: "2026-01-01", balances: "1000", investments: "2893.125", liabilities: "50", netWorth: "3843.125", flags: [] },
        { date: "2026-01-02", balances: "1000.1", investments: "0", liabilities: "0", netWorth: "1000.1", flags: [] },
      ],
      want: `${header}2026-01-01,1000,2893.125,50,3843.125,EUR,\r\n2026-01-02,1000.1,0,0,1000.1,EUR,\r\n`,
    },
    {
      name: "flags joined, and quoted when a name holds a comma or a quote",
      points: [
        {
          date: "2026-01-01",
          balances: "1",
          investments: "0",
          liabilities: "0",
          netWorth: "1",
          flags: ["no GBP rate", 'no value for Berkshire "B", Class B'],
        },
      ],
      want: `${header}2026-01-01,1,0,0,1,EUR,"no GBP rate; no value for Berkshire ""B"", Class B"\r\n`,
    },
  ]

  for (const { name, points, want } of cases) {
    it(name, () => {
      expect(netWorthCsv(points, "EUR")).toBe(want)
    })
  }
})

describe("netWorthCsvName", () => {
  it("the currency and the range", () => {
    expect(netWorthCsvName("EUR", "2025-11-01", "2026-10-31")).toBe("net-worth-EUR-2025-11-01-2026-10-31.csv")
  })
})
