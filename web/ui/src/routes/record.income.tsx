import { createFileRoute } from "@tanstack/react-router"

import { RecordTransactionPage } from "@/components/record-transaction-page"

export const Route = createFileRoute("/record/income")({
  component: RecordIncome,
})

function RecordIncome() {
  return <RecordTransactionPage type="income" />
}
