import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient, type UseMutationOptions } from "@tanstack/react-query"
import { useForm } from "react-hook-form"

import { zAccountName } from "@/api/zod.gen"
import { ProblemMessages, TextField } from "@/components/form-fields"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { FieldGroup } from "@/components/ui/field"

type RenameDialogProps<TData, TError, TVariables> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  // name is the current name, which the form starts from.
  name: string
  // description says which names are taken.
  description: string
  // mutation renames it, given variables(name).
  mutation: UseMutationOptions<TData, TError, TVariables>
  variables: (name: string) => TVariables
}

// RenameDialog renames a financial account, a ledger account or a
// counterparty, and refreshes what shows its name once it is renamed.
export function RenameDialog<TData, TError, TVariables>({
  open,
  onOpenChange,
  name,
  ...form
}: RenameDialogProps<TData, TError, TVariables>) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename {name}</DialogTitle>
          <DialogDescription>{form.description}</DialogDescription>
        </DialogHeader>
        {open && <RenameForm name={name} onDone={() => onOpenChange(false)} {...form} />}
      </DialogContent>
    </Dialog>
  )
}

function RenameForm<TData, TError, TVariables>({
  name,
  mutation,
  variables,
  onDone,
}: Omit<RenameDialogProps<TData, TError, TVariables>, "open" | "onOpenChange" | "description"> & {
  onDone: () => void
}) {
  const queryClient = useQueryClient()
  const form = useForm({ resolver: zodResolver(zAccountName), defaultValues: { name } })
  const rename = useMutation({
    ...mutation,
    onSuccess: () => {
      for (const _id of [
        "listFinancialAccounts",
        "listCounterparties",
        "getAccountBalances",
        "getLedgerAccountBalance",
        "listJournalEntries",
        "getJournalEntry",
      ]) {
        void queryClient.invalidateQueries({ queryKey: [{ _id }] })
      }
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => rename.mutate(variables(body.name)))}>
      <FieldGroup>
        <TextField label="Name" error={form.formState.errors.name} {...form.register("name")} />
        <ProblemMessages error={rename.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={rename.isPending}>
          Rename
        </Button>
      </DialogFooter>
    </form>
  )
}
