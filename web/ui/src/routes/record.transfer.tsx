import { createFileRoute } from "@tanstack/react-router"

import { RecordTransactionPage } from "@/components/record-transaction-page"

export const Route = createFileRoute("/record/transfer")({
  component: RecordTransfer,
})

function RecordTransfer() {
  return <RecordTransactionPage type="transfer" />
}
