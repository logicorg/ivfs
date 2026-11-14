import type { Currency, Movement } from "@/api"

import { addDecimals } from "./decimal"
import type { Money } from "./holdings"

// MovementRow is a movement as a holding's history lists it.
export type MovementRow = {
  id: number
  kind: Movement["kind"]
  occurredAt: string
  unitsDelta: string
  // unitsAfter is the units held once this movement and every earlier one
  // count.
  unitsAfter: string
  costBasis?: Money
  journalEntryId?: number
  description?: string
}

// movementRows names the currencies of movements, which come oldest first as
// the API pages them, and keeps a running count of the units.
export function movementRows(movements: Movement[], currencies: Currency[]): MovementRow[] {
  const currenciesById = new Map(currencies.map((currency) => [currency.id, currency]))
  let units = "0"
  return movements.map((movement) => {
    units = addDecimals(units, movement.unitsDelta)
    const currency = movement.costBasis && currenciesById.get(movement.costBasis.currencyId)
    return {
      id: movement.id,
      kind: movement.kind,
      occurredAt: movement.occurredAt,
      unitsDelta: movement.unitsDelta,
      unitsAfter: units,
      costBasis: movement.costBasis && {
        amount: movement.costBasis.amount,
        currencyCode: currency?.code ?? `currency ${movement.costBasis.currencyId}`,
        precision: currency?.precision,
      },
      journalEntryId: movement.journalEntryId,
      description: movement.description || undefined,
    }
  })
}
