import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"

import type { FinancialAccount } from "@/api"
import {
  closeFinancialAccountMutation,
  closeInstitutionMutation,
  deleteFinancialAccountMutation,
  deleteInstitutionMutation,
  getAccountBalancesOptions,
  listCurrenciesOptions,
  listFinancialAccountsOptions,
  previewFinancialAccountDeletionOptions,
  previewInstitutionDeletionOptions,
  renameFinancialAccountMutation,
} from "@/api/@tanstack/react-query.gen"
import { AddBalanceDialog } from "@/components/add-balance-dialog"
import { CloseDialog } from "@/components/close-dialog"
import { DeleteDialog } from "@/components/delete-dialog"
import { ProblemMessages } from "@/components/form-fields"
import { OpenAccountDialog } from "@/components/open-account-dialog"
import { RecordTransactionDialog } from "@/components/record-transaction-dialog"
import { RenameDialog } from "@/components/rename-dialog"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Table, TableBody, TableCell, TableRow } from "@/components/ui/table"
import { formatAmount } from "@/lib/money"

// AccountsSearch is the screen's filter, kept in the URL.
type AccountsSearch = { includeClosed?: boolean }

export const Route = createFileRoute("/accounts")({
  validateSearch: (search: Record<string, unknown>): AccountsSearch =>
    search.includeClosed === true || search.includeClosed === "true" ? { includeClosed: true } : {},
  component: Accounts,
})

function Accounts() {
  const { includeClosed } = Route.useSearch()
  const navigate = Route.useNavigate()
  const accounts = useQuery(listFinancialAccountsOptions({ query: { includeClosed } }))
  const currencies = useQuery(listCurrenciesOptions())
  const [opening, setOpening] = useState(false)
  const [recording, setRecording] = useState(false)

  const precisions = new Map(currencies.data?.currencies.map((currency) => [currency.code, currency.precision]))
  const byInstitution = groupByInstitution(accounts.data?.financialAccounts ?? [])

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="flex items-center">
        <h1 className="text-2xl font-semibold">Accounts</h1>
        <Button variant="outline" className="ml-auto" onClick={() => setRecording(true)}>
          Record transaction
        </Button>
        <Button className="ml-2" onClick={() => setOpening(true)}>
          Open account
        </Button>
        <RecordTransactionDialog open={recording} onOpenChange={setRecording} />
        <OpenAccountDialog open={opening} onOpenChange={setOpening} />
      </div>

      <div className="flex items-center gap-2">
        <Checkbox
          id="include-closed"
          checked={includeClosed ?? false}
          onCheckedChange={(checked) => navigate({ search: checked === true ? { includeClosed: true } : {} })}
        />
        <Label htmlFor="include-closed">Show closed accounts</Label>
      </div>

      {accounts.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : accounts.isError ? (
        <ProblemMessages error={accounts.error} />
      ) : byInstitution.length === 0 ? (
        <p className="text-muted-foreground">No accounts yet.</p>
      ) : (
        byInstitution.map(([institution, institutionAccounts]) => (
          <section key={institution} className="flex flex-col gap-3">
            <InstitutionHeader account={institutionAccounts[0]} />
            {institutionAccounts.map((account) => (
              <AccountCard key={account.id} account={account} precisions={precisions} />
            ))}
          </section>
        ))
      )}
    </div>
  )
}

// groupByInstitution groups accounts under their institution's name, both
// sorted by name.
function groupByInstitution(accounts: FinancialAccount[]): [string, FinancialAccount[]][] {
  const groups = new Map<string, FinancialAccount[]>()
  for (const account of [...accounts].sort((a, b) => a.name.localeCompare(b.name))) {
    groups.set(account.institutionName, [...(groups.get(account.institutionName) ?? []), account])
  }
  return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b))
}

// InstitutionHeader names an institution, given one of its accounts, closes it
// once each of its accounts is closed, and deletes it with its accounts.
function InstitutionHeader({ account }: { account: FinancialAccount }) {
  const [closing, setClosing] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const path = { id: account.institutionId }

  return (
    <div className="flex items-center gap-2">
      <h2 className="text-lg font-medium">{account.institutionName}</h2>
      {account.institutionClosed ? (
        <Badge variant="outline">closed</Badge>
      ) : (
        <>
          <Button variant="ghost" size="sm" className="ml-auto" onClick={() => setClosing(true)}>
            Close institution
          </Button>
          <CloseDialog
            open={closing}
            onOpenChange={setClosing}
            title={`Close ${account.institutionName}?`}
            description="It stays in the ledger with its history, and no account can be opened at it. Each of its accounts must be closed first."
            mutation={closeInstitutionMutation()}
            variables={{ path }}
          />
        </>
      )}
      <Button
        variant="ghost"
        size="sm"
        className={account.institutionClosed ? "ml-auto" : undefined}
        onClick={() => setDeleting(true)}
      >
        Delete institution
      </Button>
      <DeleteDialog
        open={deleting}
        onOpenChange={setDeleting}
        title={`Delete ${account.institutionName} and its accounts?`}
        preview={previewInstitutionDeletionOptions({ path })}
        mutation={deleteInstitutionMutation()}
        variables={{ path }}
      />
    </div>
  )
}

// AccountCard shows a financial account's balances, each in its natural
// direction: what an asset holds, or what a liability owes.
function AccountCard({ account, precisions }: { account: FinancialAccount; precisions: Map<string, number> }) {
  const balances = useQuery(getAccountBalancesOptions({ path: { id: account.id } }))
  const [closing, setClosing] = useState(false)
  const [adding, setAdding] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const path = { id: account.id }

  return (
    <Card className={account.closed ? "opacity-60" : undefined}>
      <CardHeader>
        <CardTitle>{account.name}</CardTitle>
        <CardDescription>{account.institutionName}</CardDescription>
        <CardAction>
          <Button variant="ghost" size="sm" onClick={() => setRenaming(true)}>
            Rename
          </Button>
          {account.closed ? (
            <Badge variant="outline">closed</Badge>
          ) : (
            <>
              <Button variant="ghost" size="sm" onClick={() => setAdding(true)}>
                Add balance
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setClosing(true)}>
                Close
              </Button>
            </>
          )}
          <Button variant="ghost" size="sm" onClick={() => setDeleting(true)}>
            Delete
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {balances.isPending ? (
          <p className="text-muted-foreground">Loading…</p>
        ) : balances.isError ? (
          <ProblemMessages error={balances.error} />
        ) : balances.data.balances.length === 0 ? (
          <p className="text-muted-foreground">No balances yet.</p>
        ) : (
          <Table>
            <TableBody>
              {balances.data.balances.map((balance) => (
                <TableRow key={balance.ledgerAccountId}>
                  <TableCell>
                    <span className="flex items-center gap-2">
                      <Link
                        to="/ledger-accounts/$id"
                        params={{ id: String(balance.ledgerAccountId) }}
                        className="hover:underline"
                      >
                        {balance.ledgerAccountName}
                      </Link>
                      {balance.kind === "liability" && <Badge variant="secondary">owed</Badge>}
                      {balance.closed && <Badge variant="outline">closed</Badge>}
                    </span>
                  </TableCell>
                  <TableCell className="w-16 text-muted-foreground">{balance.currencyCode}</TableCell>
                  <TableCell className="w-40 text-right tabular-nums">
                    {formatAmount(balance.balance, balance.currencyCode, precisions.get(balance.currencyCode))}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
      <AddBalanceDialog account={account} open={adding} onOpenChange={setAdding} />
      <RenameDialog
        open={renaming}
        onOpenChange={setRenaming}
        name={account.name}
        description={`No other account at ${account.institutionName} can have the same name.`}
        mutation={renameFinancialAccountMutation()}
        variables={(name) => ({ path: { id: account.id }, body: { name } })}
      />
      <CloseDialog
        open={closing}
        onOpenChange={setClosing}
        title={`Close ${account.name}?`}
        description="It stays in the ledger with its history, and nothing can be booked to it. Each of its balances must be closed first."
        mutation={closeFinancialAccountMutation()}
        variables={{ path }}
      />
      <DeleteDialog
        open={deleting}
        onOpenChange={setDeleting}
        title={`Delete ${account.name} and its balances?`}
        preview={previewFinancialAccountDeletionOptions({ path })}
        mutation={deleteFinancialAccountMutation()}
        variables={{ path }}
      />
    </Card>
  )
}
