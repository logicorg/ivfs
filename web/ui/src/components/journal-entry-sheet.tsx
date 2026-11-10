import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"

import type { Currency, EntryDetail, NegativeHolding } from "@/api"
import {
  deleteJournalEntryMutation,
  getJournalEntryOptions,
  listCurrenciesOptions,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { NegativeHoldingWarnings } from "@/components/negative-holding-warnings"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatDate } from "@/lib/dates"
import { formatAmount } from "@/lib/money"

type JournalEntrySheetProps = {
  // entryId is the entry to show, and the sheet is closed without one.
  entryId: number | undefined
  // onSelect shows another entry, and closes the sheet when given none.
  onSelect: (entryId: number | undefined) => void
}

// JournalEntrySheet shows a journal entry with its postings, and deletes it.
export function JournalEntrySheet({ entryId, onSelect }: JournalEntrySheetProps) {
  // deleted is the entry just deleted along with the holdings that deletion left
  // negative, shown in place of the entry while the sheet still shows its id.
  const [deleted, setDeleted] = useState<{ entryId: number; warnings: NegativeHolding[] }>()
  const isDeleted = entryId !== undefined && deleted?.entryId === entryId
  const entry = useQuery({
    ...getJournalEntryOptions({ path: { id: entryId ?? 0 } }),
    enabled: entryId !== undefined && !isDeleted,
  })
  const currencies = useQuery(listCurrenciesOptions())
  const [deleting, setDeleting] = useState(false)

  return (
    <Sheet open={entryId !== undefined} onOpenChange={(open) => !open && onSelect(undefined)}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-xl">
        {isDeleted ? (
          <>
            <SheetHeader>
              <SheetTitle>Journal entry deleted</SheetTitle>
            </SheetHeader>
            <NegativeHoldingWarnings title="This deletion leaves a holding negative" warnings={deleted.warnings} />
          </>
        ) : entry.isPending ? (
          <SheetHeader>
            <SheetTitle>Journal entry</SheetTitle>
            <SheetDescription>Loading…</SheetDescription>
          </SheetHeader>
        ) : entry.isError ? (
          <SheetHeader>
            <SheetTitle>Journal entry</SheetTitle>
            <ProblemMessages error={entry.error} />
          </SheetHeader>
        ) : (
          <>
            <SheetHeader>
              <SheetTitle>{entry.data.description ?? `Journal entry ${entry.data.id}`}</SheetTitle>
              <SheetDescription>{formatDate(entry.data.occurredAt, true)}</SheetDescription>
            </SheetHeader>
            <EntryFacts entry={entry.data} />
            <Postings entry={entry.data} currencies={currencies.data?.currencies ?? []} />
            <SheetFooter>
              <Button variant="destructive" onClick={() => setDeleting(true)}>
                Delete entry
              </Button>
            </SheetFooter>
            <DeleteEntryDialog
              entry={deleting ? entry.data : null}
              onOpenChange={setDeleting}
              onDeleted={(deletedId, warnings) => {
                setDeleting(false)
                // With nothing to warn about, there is nothing left to show.
                if (warnings.length === 0) {
                  onSelect(undefined)
                } else {
                  setDeleted({ entryId: deletedId, warnings })
                }
              }}
            />
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}

function EntryFacts({ entry }: { entry: EntryDetail }) {
  return (
    <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 px-4">
      <dt className="text-muted-foreground">Status</dt>
      <dd>
        <Badge variant={entry.status === "posted" ? "secondary" : "outline"}>{entry.status}</Badge>
      </dd>
      {entry.counterpartyId !== undefined && (
        <>
          <dt className="text-muted-foreground">Counterparty</dt>
          <dd>
            <Link to="/counterparties/$id" params={{ id: String(entry.counterpartyId) }} className="underline">
              {entry.counterpartyName}
            </Link>
          </dd>
        </>
      )}
      {entry.externalId && (
        <>
          <dt className="text-muted-foreground">External id</dt>
          <dd className="font-mono text-xs">{entry.externalId}</dd>
        </>
      )}
      {entry.postedAt && (
        <>
          <dt className="text-muted-foreground">Posted</dt>
          <dd>{formatDate(entry.postedAt, true)}</dd>
        </>
      )}
    </dl>
  )
}

// Postings shows an entry's postings in line order, in debit and credit
// columns, each in its ledger account's currency.
function Postings({ entry, currencies }: { entry: EntryDetail; currencies: Currency[] }) {
  const byId = new Map(currencies.map((currency) => [currency.id, currency]))
  const amount = (value: string, currencyId: number) => {
    const currency = byId.get(currencyId)
    return currency ? formatAmount(value, currency.code, currency.precision) : value
  }

  return (
    <div className="px-4">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Account</TableHead>
            <TableHead className="text-right">Debit</TableHead>
            <TableHead className="text-right">Credit</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entry.postings.map((posting) => (
            <TableRow key={posting.lineNumber}>
              <TableCell>{posting.ledgerAccountName}</TableCell>
              <TableCell className="text-right tabular-nums">
                {posting.side === "debit" && amount(posting.amount, posting.currencyId)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {posting.side === "credit" && amount(posting.amount, posting.currencyId)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

type DeleteEntryDialogProps = {
  entry: EntryDetail | null
  onOpenChange: (open: boolean) => void
  // onDeleted is given the deleted entry and the positions its deletion left
  // negative.
  onDeleted: (entryId: number, warnings: NegativeHolding[]) => void
}

// DeleteEntryDialog deletes an entry, once confirmed, as if it had never been
// recorded (ADR 0011).
function DeleteEntryDialog({ entry, onOpenChange, onDeleted }: DeleteEntryDialogProps) {
  const queryClient = useQueryClient()
  const remove = useMutation({
    ...deleteJournalEntryMutation(),
    onSuccess: (deletion, { path }) => {
      // The deletion changes the statements and balances of every account the
      // entry posted to, and takes the entry's movements with it, which may be
      // open elsewhere.
      for (const _id of [
        "listJournalEntries",
        "getJournalEntry",
        "getLedgerAccountBalance",
        "getAccountBalances",
        "getCategoryTotals",
        "listHoldings",
        "listHoldingMovements",
      ]) {
        void queryClient.invalidateQueries({ queryKey: [{ _id }] })
      }
      onDeleted(path.id, deletion.warnings ?? [])
    },
  })

  return (
    <AlertDialog
      open={entry !== null}
      onOpenChange={(open) => {
        if (!open) {
          remove.reset()
        }
        onOpenChange(open)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this entry?</AlertDialogTitle>
          <AlertDialogDescription>
            The entry, its postings, and its position movements leave the ledger as if never recorded.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <ProblemMessages error={remove.error} />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={remove.isPending}
            onClick={(event) => {
              // Keep the dialog open until the gateway answers, to show why
              // it refused.
              event.preventDefault()
              if (entry) {
                remove.mutate({ path: { id: entry.id } })
              }
            }}
          >
            Delete entry
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
