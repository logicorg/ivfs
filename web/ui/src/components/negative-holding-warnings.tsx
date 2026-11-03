import { TriangleAlert } from "lucide-react"

import type { NegativeHolding } from "@/api"
import { describeNegativeHolding } from "@/lib/negative-holdings"

// NegativeHoldingWarnings shows the positions a reversal, a deletion or a merge left
// negative at some point in time. The change is made all the same, so they are
// for the reader to fix, by recording what is missing or undoing what was wrong.
export function NegativeHoldingWarnings({ title, warnings }: { title: string; warnings: NegativeHolding[] }) {
  if (warnings.length === 0) {
    return null
  }
  return (
    <div role="status" className="mx-4 flex gap-3 rounded-md border px-3 py-2 text-sm">
      <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div>
        <p className="font-medium">{title}</p>
        <ul className="text-muted-foreground">
          {warnings.map((warning) => (
            <li key={`${warning.financialAccountId}-${warning.assetId}`}>{describeNegativeHolding(warning)}</li>
          ))}
        </ul>
      </div>
    </div>
  )
}
