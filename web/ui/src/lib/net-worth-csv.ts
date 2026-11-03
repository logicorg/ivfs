import type { NetWorthPoint } from "./net-worth-series"

// netWorthCsv is the valued series as CSV, oldest first, with the columns ADR
// 0015 names. Amounts are exact and unrounded, unlike the table's, so a
// spreadsheet gets every digit; flags are joined with "; ".
export function netWorthCsv(points: NetWorthPoint[], currencyCode: string): string {
  const header = "date,balances,investments,liabilities,net_worth,currency,flags"
  const rows = points.map(({ date, balances, investments, liabilities, netWorth, flags }) =>
    [date, balances, investments, liabilities, netWorth, currencyCode, flags.join("; ")].map(field).join(","),
  )
  return [header, ...rows].join("\r\n") + "\r\n"
}

// field quotes a value as RFC 4180 asks when it holds a comma, a quote or a
// line break, as an asset's name in a flag can.
function field(value: string): string {
  return /[",\r\n]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value
}

// netWorthCsvName names the file after what it holds, such as
// net-worth-EUR-2025-11-01-2026-10-31.csv.
export function netWorthCsvName(currencyCode: string, from: string, to: string): string {
  return `net-worth-${currencyCode}-${from}-${to}.csv`
}
