import { createFileRoute } from "@tanstack/react-router"

import { RecordTransactionPage } from "@/components/record-transaction-page"

export const Route = createFileRoute("/record/expense")({
  component: RecordExpense,
})

function RecordExpense() {
  return <RecordTransactionPage type="expense" />
}
