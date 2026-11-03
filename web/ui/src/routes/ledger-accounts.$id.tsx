import { useState } from "react"
import { useInfiniteQuery, useQuery } from "@tanstack/react-query"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"

import type { Balance } from "@/api"
import {
  closeLedgerAccountMutation,
  deleteLedgerAccountMutation,
  getLedgerAccountBalanceOptions,
  listCurrenciesOptions,
  listJournalEntriesInfiniteOptions,
  previewLedgerAccountDeletionOptions,
  renameLedgerAccountMutation,
} from "@/api/@tanstack/react-query.gen"
import { CloseDialog } from "@/components/close-dialog"
import { DeleteDialog } from "@/components/delete-dialog"
import { ProblemMessages } from "@/components/form-fields"
import { JournalEntrySheet } from "@/components/journal-entry-sheet"
import { RenameDialog } from "@/components/rename-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatDate } from "@/lib/dates"
import { formatAmount } from "@/lib/money"
import { statementLines } from "@/lib/statement"

// pageSize is how many entries each "Show older entries" loads.
const pageSize = 50

// StatementSearch is the entry open in the sheet, kept in the URL.
type StatementSearch = { entry?: number }

export const Route = createFileRoute("/ledger-accounts/$id")({
  validateSearch: (search: Record<string, unknown>): StatementSearch => {
    const entry = Number(search.entry)
    return Number.isInteger(entry) && entry > 0 ? { entry } : {}
  },
  component: Statement,
})

// Statement is a ledger account's posted entries, newest first, with its
// balance now.
function Statement() {
  const id = Number(Route.useParams().id)
  const { entry } = Route.useSearch()
  const navigate = Route.useNavigate()
  const balance = useQuery(getLedgerAccountBalanceOptions({ path: { id } }))

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <Button variant="ghost" size="sm" className="self-start" asChild>
        <Link to="/accounts">
          <ArrowLeft /> Accounts
        </Link>
      </Button>

      {balance.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : balance.isError ? (
        <ProblemMessages error={balance.error} />
      ) : (
        <>
          <StatementHeader balance={balance.data} />
          <Entries
            account={balance.data}
            onSelect={(entryId) => navigate({ search: { entry: entryId }, replace: entry !== undefined })}
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

function StatementHeader({ balance }: { balance: Balance }) {
  const currencies = useQuery(listCurrenciesOptions())
  const precision = currencies.data?.currencies.find((currency) => currency.id === balance.currencyId)?.precision
  const [closing, setClosing] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const navigate = useNavigate()
  const path = { id: balance.ledgerAccountId }

  return (
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold">{balance.ledgerAccountName}</h1>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Badge variant="outline">{balance.kind}</Badge>
          {balance.currencyCode}
          <Button variant="ghost" size="sm" onClick={() => setRenaming(true)}>
            Rename
          </Button>
          {balance.closed ? (
            <Badge variant="outline">closed</Badge>
          ) : (
            <Button variant="ghost" size="sm" onClick={() => setClosing(true)}>
              Close account
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => setDeleting(true)}>
            Delete
          </Button>
        </div>
      </div>
      <div className="text-right">
        <div className="text-sm text-muted-foreground">{balance.kind === "liability" ? "Owed" : "Balance"}</div>
        <div className="text-2xl font-semibold tabular-nums">
          {formatAmount(balance.balance, balance.currencyCode, precision)}
        </div>
      </div>
      <RenameDialog
        open={renaming}
        onOpenChange={setRenaming}
        name={balance.ledgerAccountName}
        description="No other balance of its account in the same currency can have the same name."
        mutation={renameLedgerAccountMutation()}
        variables={(name) => ({ path: { id: balance.ledgerAccountId }, body: { name } })}
      />
      <CloseDialog
        open={closing}
        onOpenChange={setClosing}
        title={`Close ${balance.ledgerAccountName}?`}
        description="It stays in the ledger with its entries, and nothing more can be booked to it. An asset or liability must have no balance left."
        mutation={closeLedgerAccountMutation()}
        variables={{ path: { id: balance.ledgerAccountId } }}
        onClosed={() => void navigate({ to: "/accounts" })}
      />
      <DeleteDialog
        open={deleting}
        onOpenChange={setDeleting}
        title={`Delete ${balance.ledgerAccountName}?`}
        preview={previewLedgerAccountDeletionOptions({ path })}
        mutation={deleteLedgerAccountMutation()}
        variables={{ path }}
        onDeleted={() => void navigate({ to: "/accounts" })}
      />
    </div>
  )
}

// Entries lists the account's posted entries, a page at a time.
function Entries({ account, onSelect }: { account: Balance; onSelect: (entryId: number) => void }) {
  const currencies = useQuery(listCurrenciesOptions())
  const precision = currencies.data?.currencies.find((currency) => currency.id === account.currencyId)?.precision
  const entries = useInfiniteQuery({
    ...listJournalEntriesInfiniteOptions({
      query: { ledgerAccountId: account.ledgerAccountId, status: ["posted"], descending: true, limit: pageSize },
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

  const lines = statementLines(
    entries.data.pages.flatMap((page) => page.entries),
    account.ledgerAccountId,
    account.kind,
  )
  if (lines.length === 0) {
    return <p className="text-muted-foreground">No posted entries yet.</p>
  }

  return (
    <div className="flex flex-col gap-3">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-32">Date</TableHead>
            <TableHead>Description</TableHead>
            <TableHead>Against</TableHead>
            <TableHead className="text-right">Amount</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {lines.map((line) => (
            <TableRow key={line.key} className="cursor-pointer" onClick={() => onSelect(line.entryId)}>
              <TableCell className="text-muted-foreground">{formatDate(line.occurredAt)}</TableCell>
              <TableCell>
                <button
                  type="button"
                  className="flex items-center gap-2 text-left hover:underline"
                  onClick={(event) => {
                    event.stopPropagation()
                    onSelect(line.entryId)
                  }}
                >
                  {line.description ?? <span className="text-muted-foreground">Journal entry {line.entryId}</span>}
                  {line.reversalOfEntryId !== undefined && <Badge variant="outline">reversal</Badge>}
                </button>
              </TableCell>
              <TableCell className="text-muted-foreground">{line.otherAccounts.join(", ")}</TableCell>
              <TableCell className="text-right tabular-nums">
                {formatAmount(line.amount, account.currencyCode, precision)}
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
