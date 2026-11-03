import type { ReactNode } from "react"

import { RecordTransactionForm } from "@/components/record-transaction-form"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// RecordTransactionDialog records a transaction of any type, chosen in the
// dialog. The form mounts per opening, so each opening records one.
export function RecordTransactionDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Record a transaction</DialogTitle>
          <DialogDescription>Money spent, earned, or moved between your balances.</DialogDescription>
        </DialogHeader>
        {open && <RecordTransactionForm onRecorded={() => onOpenChange(false)} Footer={Footer} />}
      </DialogContent>
    </Dialog>
  )
}

function Footer({ children }: { children: ReactNode }) {
  return <DialogFooter className="mt-6">{children}</DialogFooter>
}
