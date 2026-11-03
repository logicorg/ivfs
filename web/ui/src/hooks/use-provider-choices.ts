import { useState } from "react"

import { parseProviderChoices, valueProvidersStorageKey, type ProviderChoices } from "@/lib/asset-values"

// Choices are the reader's provider choices and how to change one.
export type Choices = { chosen: ProviderChoices; choose: (assetId: number, provider: string) => void }

// useProviderChoices keeps the reader's provider choices in localStorage, so
// they outlast a reload; the gateway never sees them.
export function useProviderChoices(): Choices {
  const [chosen, setChosen] = useState<ProviderChoices>(() => parseProviderChoices(readStored()))
  const choose = (assetId: number, provider: string) => {
    const next = { ...chosen, [assetId]: provider }
    try {
      localStorage.setItem(valueProvidersStorageKey, JSON.stringify(next))
    } catch {
      // Storage can be off, as in some private windows; the choice then lasts
      // until the page reloads.
    }
    setChosen(next)
  }
  return { chosen, choose }
}

function readStored(): string | null {
  try {
    return localStorage.getItem(valueProvidersStorageKey)
  } catch {
    return null
  }
}
