import type { ReactNode } from "react"

import type { EntryDetail, NegativeHolding } from "@/api"
import { RecordTransactionForm } from "@/components/record-transaction-form"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { formValuesOf } from "@/lib/correct-transaction"

type CorrectTransactionDialogProps = {
  // entry is the entry to correct, and the dialog is closed without one. Its
  // transaction is what the form starts from.
  entry: EntryDetail | null
  onOpenChange: (open: boolean) => void
  // onCorrected is given the holdings the correction left negative.
  onCorrected: (warnings: NegativeHolding[]) => void
}

// CorrectTransactionDialog corrects an entry a transaction posted, which keeps
// its id: to other amounts, accounts, or categories, or to another type.
export function CorrectTransactionDialog({ entry, onOpenChange, onCorrected }: CorrectTransactionDialogProps) {
  const values = entry?.transaction ? formValuesOf(entry.transaction) : undefined

  return (
    <Dialog open={entry !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit entry</DialogTitle>
          <DialogDescription>The entry becomes what the form states, and keeps its id.</DialogDescription>
        </DialogHeader>
        {entry && values && (
          <RecordTransactionForm
            correcting={{ entryId: entry.id, occurredAt: entry.occurredAt, values }}
            onRecorded={onCorrected}
            Footer={Footer}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function Footer({ children }: { children: ReactNode }) {
  return <DialogFooter className="mt-6">{children}</DialogFooter>
}
