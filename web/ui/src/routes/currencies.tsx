import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { Pencil, Plus, Trash2 } from "lucide-react"

import type { Currency } from "@/api"
import { listCurrenciesOptions } from "@/api/@tanstack/react-query.gen"
import { CreateCurrencyDialog, DeleteCurrencyDialog, EditCurrencyDialog } from "@/components/currency-dialogs"
import { Button } from "@/components/ui/button"
import { FieldError } from "@/components/ui/field"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { problemMessages } from "@/lib/problem"

export const Route = createFileRoute("/currencies")({
  component: Currencies,
})

function Currencies() {
  const currencies = useQuery(listCurrenciesOptions())
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Currency | null>(null)
  const [deleting, setDeleting] = useState<Currency | null>(null)

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Currencies</h1>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          Add currency
        </Button>
      </div>

      {currencies.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : currencies.isError ? (
        <FieldError errors={problemMessages(currencies.error).map((message) => ({ message }))} />
      ) : currencies.data.currencies.length === 0 ? (
        <p className="text-muted-foreground">No currencies yet.</p>
      ) : (
        <CurrencyTable currencies={currencies.data.currencies} onEdit={setEditing} onDelete={setDeleting} />
      )}

      <CreateCurrencyDialog open={creating} onOpenChange={setCreating} />
      <EditCurrencyDialog currency={editing} onOpenChange={(open) => !open && setEditing(null)} />
      <DeleteCurrencyDialog currency={deleting} onOpenChange={(open) => !open && setDeleting(null)} />
    </div>
  )
}

function CurrencyTable({
  currencies,
  onEdit,
  onDelete,
}: {
  currencies: Currency[]
  onEdit: (currency: Currency) => void
  onDelete: (currency: Currency) => void
}) {
  const byCode = [...currencies].sort((a, b) => a.code.localeCompare(b.code))

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Code</TableHead>
          <TableHead>Name</TableHead>
          <TableHead>Symbol</TableHead>
          <TableHead className="text-right">Decimal places</TableHead>
          <TableHead className="w-24" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {byCode.map((currency) => (
          <TableRow key={currency.id}>
            <TableCell className="font-medium">{currency.code}</TableCell>
            <TableCell>{currency.name}</TableCell>
            <TableCell>{currency.symbol}</TableCell>
            <TableCell className="text-right">{currency.precision}</TableCell>
            <TableCell>
              <Button variant="ghost" size="icon" aria-label={`Edit ${currency.code}`} onClick={() => onEdit(currency)}>
                <Pencil />
              </Button>
              <Button variant="ghost" size="icon" aria-label={`Delete ${currency.code}`} onClick={() => onDelete(currency)}>
                <Trash2 />
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
