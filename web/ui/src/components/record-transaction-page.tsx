import { useState, type ReactNode } from "react"

import { RecordTransactionForm } from "@/components/record-transaction-form"
import type { TransactionType } from "@/lib/record-transaction"

const copy: Record<TransactionType, { title: string; description: string; recorded: string }> = {
  expense: {
    title: "Record an expense",
    description: "Money spent from one of your balances, filed under a category.",
    recorded: "Expense recorded.",
  },
  income: {
    title: "Record an income",
    description: "Money received into one of your balances, filed under a category.",
    recorded: "Income recorded.",
  },
  transfer: {
    title: "Record a transfer",
    description: "Money moved between two of your balances, in one currency or across two.",
    recorded: "Transfer recorded.",
  },
}

// RecordTransactionPage records transactions of one type, one after another:
// each recording remounts the form, which clears it and makes a new
// idempotency key for the next.
export function RecordTransactionPage({ type }: { type: TransactionType }) {
  const [recorded, setRecorded] = useState(0)
  const { title, description, recorded: confirmation } = copy[type]

  return (
    <div className="flex max-w-md flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold">{title}</h1>
        <p className="text-muted-foreground">{description}</p>
      </div>
      <RecordTransactionForm
        key={recorded}
        type={type}
        onRecorded={() => setRecorded((count) => count + 1)}
        Footer={Footer}
      />
      {recorded > 0 && (
        <p role="status" className="text-sm text-muted-foreground">
          {confirmation}
        </p>
      )}
    </div>
  )
}

function Footer({ children }: { children: ReactNode }) {
  return <div className="mt-6 flex justify-end">{children}</div>
}
