import { useState } from "react"
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import { Pencil, Plus } from "lucide-react"

import type { Counterparty } from "@/api"
import { listCounterpartiesOptions, renameCounterpartyMutation } from "@/api/@tanstack/react-query.gen"
import { CreateCounterpartyDialog } from "@/components/counterparty-dialogs"
import { ProblemMessages } from "@/components/form-fields"
import { RenameDialog } from "@/components/rename-dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export const Route = createFileRoute("/counterparties/")({
  component: Counterparties,
})

function Counterparties() {
  const [search, setSearch] = useState("")
  const nameContains = search.trim()
  const counterparties = useQuery({
    ...listCounterpartiesOptions({ query: nameContains === "" ? {} : { nameContains } }),
    // Keep the last results on screen while the next search loads.
    placeholderData: keepPreviousData,
  })
  const [creating, setCreating] = useState(false)
  const [renaming, setRenaming] = useState<Counterparty | null>(null)

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Counterparties</h1>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          Add counterparty
        </Button>
      </div>

      <Input
        type="search"
        placeholder="Search by name"
        aria-label="Search by name"
        className="max-w-xs"
        value={search}
        onChange={(event) => setSearch(event.target.value)}
      />

      {counterparties.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : counterparties.isError ? (
        <ProblemMessages error={counterparties.error} />
      ) : counterparties.data.counterparties.length === 0 ? (
        <p className="text-muted-foreground">{nameContains === "" ? "No counterparties yet." : "No counterparty matches."}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>External id</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {counterparties.data.counterparties.map((counterparty) => (
              <TableRow key={counterparty.id}>
                <TableCell className="font-medium">
                  <Link to="/counterparties/$id" params={{ id: String(counterparty.id) }} className="hover:underline">
                    {counterparty.name}
                  </Link>
                </TableCell>
                <TableCell className="text-muted-foreground">{counterparty.externalId}</TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Rename ${counterparty.name}`}
                    onClick={() => setRenaming(counterparty)}
                  >
                    <Pencil />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CreateCounterpartyDialog open={creating} onOpenChange={setCreating} />
      <RenameDialog
        open={renaming !== null}
        onOpenChange={(open) => !open && setRenaming(null)}
        name={renaming?.name ?? ""}
        description="No other counterparty can have the same name, in any casing. Its entries show the new name."
        mutation={renameCounterpartyMutation()}
        variables={(name) => ({ path: { id: renaming?.id ?? 0 }, body: { name } })}
      />
    </div>
  )
}
