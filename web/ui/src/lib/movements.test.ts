import { describe, expect, it } from "vitest"

import type { Currency, Movement } from "@/api"

import { movementRows, type MovementRow } from "./movements"

const currencies: Currency[] = [{ id: 1, code: "EUR", name: "Euro", precision: 2 }]
const movement = (id: number, unitsDelta: string, extra: Partial<Movement> = {}): Movement => ({
  id,
  kind: "trade",
  financialAccountId: 3,
  assetId: 7,
  unitsDelta,
  occurredAt: `2026-10-0${id}T10:00:00Z`,
  ...extra,
})
const row = (id: number, unitsDelta: string, unitsAfter: string, extra: Partial<MovementRow> = {}): MovementRow => ({
  id,
  kind: "trade",
  occurredAt: `2026-10-0${id}T10:00:00Z`,
  unitsDelta,
  unitsAfter,
  costBasis: undefined,
  journalEntryId: undefined,
  description: undefined,
  ...extra,
})

describe("movementRows", () => {
  const cases: { name: string; movements: Movement[]; want: MovementRow[] }[] = [
    { name: "no movements", movements: [], want: [] },
    {
      name: "a buy, a vest and a sale, counted in order",
      movements: [
        movement(1, "10", { costBasis: { amount: "1200.50", currencyId: 1 }, journalEntryId: 40, description: "First buy" }),
        movement(2, "2.5", { kind: "vest", costBasis: { amount: "300", currencyId: 1 }, journalEntryId: 41 }),
        movement(3, "-4", { journalEntryId: 42 }),
      ],
      want: [
        row(1, "10", "10", {
          costBasis: { amount: "1200.50", currencyCode: "EUR", precision: 2 },
          journalEntryId: 40,
          description: "First buy",
        }),
        row(2, "2.5", "12.5", {
          kind: "vest",
          costBasis: { amount: "300", currencyCode: "EUR", precision: 2 },
          journalEntryId: 41,
        }),
        row(3, "-4", "8.5", { journalEntryId: 42 }),
      ],
    },
    {
      name: "a currency not in the list",
      movements: [movement(1, "1", { costBasis: { amount: "5", currencyId: 9 } })],
      want: [row(1, "1", "1", { costBasis: { amount: "5", currencyCode: "currency 9", precision: undefined } })],
    },
  ]

  for (const { name, movements, want } of cases) {
    it(name, () => {
      expect(movementRows(movements, currencies)).toEqual(want)
    })
  }
})
