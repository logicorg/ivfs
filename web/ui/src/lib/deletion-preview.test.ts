import { describe, expect, it } from "vitest"

import type { AccountDeletionPreview } from "@/api"

import { describeDeletionPreview } from "./deletion-preview"

describe("describeDeletionPreview", () => {
  const none: AccountDeletionPreview = { deletedEntries: 0, keptEntries: 0, deletedMovementGroups: 0, keptMovementGroups: 0 }
  const cases: { name: string; preview: AccountDeletionPreview; want: string[] }[] = [
    { name: "an account without lines", preview: none, want: ["Nothing is recorded on it."] },
    {
      name: "entries and groups deleted",
      preview: { ...none, deletedEntries: 2, deletedMovementGroups: 1 },
      want: ["Deletes 2 journal entries and 1 movement group."],
    },
    {
      name: "a kept entry moving a balance",
      preview: {
        ...none,
        deletedEntries: 1,
        keptEntries: 1,
        strayAmounts: [{ name: "Untracked accounts", kind: "equity", currencyCode: "EUR", amount: "1200.5" }],
      },
      want: [
        "Deletes 1 journal entry.",
        "Keeps 1 journal entry: each also touches an account that stays, so only this side moves to the untracked accounts.",
        "Untracked accounts (equity) receives €1,200.50.",
      ],
    },
    {
      name: "income reads as earned",
      preview: {
        ...none,
        keptEntries: 3,
        strayAmounts: [{ name: "Uncategorized", kind: "income", currencyCode: "EUR", amount: "-300" }],
      },
      want: [
        "Keeps 3 journal entries: each also touches an account that stays, so only this side moves to the untracked accounts.",
        "Uncategorized (income) receives €300.00.",
      ],
    },
    {
      name: "units, kept exact",
      preview: { ...none, keptMovementGroups: 2, strayUnits: [{ assetId: 12, assetName: "ACME", units: "-0.000123" }] },
      want: [
        "Keeps 2 movement groups: each also touches an account that stays, so only this side moves to the untracked accounts.",
        "Untracked accounts receives -0.000123 ACME.",
      ],
    },
  ]

  for (const { name, preview, want } of cases) {
    it(name, () => {
      expect(describeDeletionPreview(preview, "en-US")).toEqual(want)
    })
  }
})
