import { useState } from "react"
import { useInfiniteQuery, useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"

import { listCounterpartiesOptions, listCurrenciesOptions, listJournalEntriesInfiniteOptions } from "@/api/@tanstack/react-query.gen"
import { DeleteCounterpartyDialog } from "@/components/counterparty-dialogs"
import { ProblemMessages } from "@/components/form-fields"
import { JournalEntrySheet } from "@/components/journal-entry-sheet"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatDate } from "@/lib/dates"
import { entryAmounts } from "@/lib/entry-amounts"
import { formatAmount } from "@/lib/money"

// pageSize is how many entries each "Show older entries" loads.
const pageSize = 50

// CounterpartySearch is the entry open in the sheet, kept in the URL.
type CounterpartySearch = { entry?: number }

export const Route = createFileRoute("/counterparties/$id")({
  validateSearch: (search: Record<string, unknown>): CounterpartySearch => {
    const entry = Number(search.entry)
    return Number.isInteger(entry) && entry > 0 ? { entry } : {}
  },
  component: CounterpartyEntries,
})

// CounterpartyEntries is a counterparty's posted entries, newest first. Its
// name comes from the list the counterparties screen loads, as there is no
// endpoint for one counterparty.
function CounterpartyEntries() {
  const id = Number(Route.useParams().id)
  const { entry } = Route.useSearch()
  const navigate = Route.useNavigate()
  const counterparties = useQuery(listCounterpartiesOptions({ query: {} }))
  const counterparty = counterparties.data?.counterparties.find((candidate) => candidate.id === id)
  const [deleting, setDeleting] = useState(false)

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <Button variant="ghost" size="sm" className="self-start" asChild>
        <Link to="/counterparties">
          <ArrowLeft /> Counterparties
        </Link>
      </Button>

      {counterparties.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : counterparties.isError ? (
        <ProblemMessages error={counterparties.error} />
      ) : !counterparty ? (
        <p className="text-muted-foreground">There is no counterparty {id}.</p>
      ) : (
        <>
          <div className="flex flex-col gap-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-semibold">{counterparty.name}</h1>
              <Button variant="ghost" size="sm" onClick={() => setDeleting(true)}>
                Delete
              </Button>
            </div>
            {counterparty.externalId && <p className="text-muted-foreground">{counterparty.externalId}</p>}
          </div>
          <Entries
            counterpartyId={id}
            onSelect={(entryId) => navigate({ search: { entry: entryId }, replace: entry !== undefined })}
          />
          <DeleteCounterpartyDialog
            counterparty={counterparty}
            open={deleting}
            onOpenChange={setDeleting}
            onDeleted={() => void navigate({ to: "/counterparties" })}
          />
        </>
      )}

      <JournalEntrySheet
        entryId={entry}
        onSelect={(entryId) => navigate({ search: entryId === undefined ? {} : { entry: entryId } })}
      />
    </div>
  )
}

// Entries lists the counterparty's posted entries, a page at a time, each
// with the accounts it moves money between and how much.
function Entries({ counterpartyId, onSelect }: { counterpartyId: number; onSelect: (entryId: number) => void }) {
  const currencies = useQuery(listCurrenciesOptions())
  const byID = new Map(currencies.data?.currencies.map((currency) => [currency.id, currency]))
  const entries = useInfiniteQuery({
    ...listJournalEntriesInfiniteOptions({
      query: { counterpartyId, status: ["posted"], descending: true, limit: pageSize },
    }),
    // The first page has no after; each next page is after the previous one.
    initialPageParam: {},
    getNextPageParam: (page) => page.next,
  })

  if (entries.isPending) {
    return <p className="text-muted-foreground">Loading…</p>
  }
  if (entries.isError) {
    return <ProblemMessages error={entries.error} />
  }

  const list = entries.data.pages.flatMap((page) => page.entries)
  if (list.length === 0) {
    return <p className="text-muted-foreground">No posted entries name this counterparty yet.</p>
  }

  return (
    <div className="flex flex-col gap-3">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-32">Date</TableHead>
            <TableHead>Description</TableHead>
            <TableHead>Accounts</TableHead>
            <TableHead className="text-right">Amount</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {list.map((item) => (
            <TableRow key={item.id} className="cursor-pointer" onClick={() => onSelect(item.id)}>
              <TableCell className="text-muted-foreground">{formatDate(item.occurredAt)}</TableCell>
              <TableCell>
                <button
                  type="button"
                  className="flex items-center gap-2 text-left hover:underline"
                  onClick={(event) => {
                    event.stopPropagation()
                    onSelect(item.id)
                  }}
                >
                  {item.description ?? <span className="text-muted-foreground">Journal entry {item.id}</span>}
                  {item.reversalOfEntryId !== undefined && <Badge variant="outline">reversal</Badge>}
                </button>
              </TableCell>
              <TableCell className="text-muted-foreground">
                {[...new Set(item.postings.map((posting) => posting.ledgerAccountName))].join(", ")}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {entryAmounts(item).map(({ currencyId, amount }) => {
                  const currency = byID.get(currencyId)
                  return <div key={currencyId}>{formatAmount(amount, currency?.code ?? "", currency?.precision)}</div>
                })}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {entries.hasNextPage && (
        <Button
          variant="outline"
          className="self-center"
          disabled={entries.isFetchingNextPage}
          onClick={() => void entries.fetchNextPage()}
        >
          {entries.isFetchingNextPage ? "Loading…" : "Show older entries"}
        </Button>
      )}
    </div>
  )
}
