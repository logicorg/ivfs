import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useForm } from "react-hook-form"

import type { Currency } from "@/api"
import {
  createCurrencyMutation,
  deleteCurrencyMutation,
  listCurrenciesQueryKey,
  updateCurrencyMutation,
} from "@/api/@tanstack/react-query.gen"
import { zCurrencyChanges, zInsertCurrencyParams } from "@/api/zod.gen"
import { ProblemMessages, TextField } from "@/components/form-fields"
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

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// CreateCurrencyDialog adds a currency. Adding one whose code exists finds
// that currency instead, as every create does (ADR 0005).
export function CreateCurrencyDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a currency</DialogTitle>
          <DialogDescription>Its precision cannot change once it is used.</DialogDescription>
        </DialogHeader>
        <CreateCurrencyForm onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}

function CreateCurrencyForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient()
  const form = useForm({ resolver: zodResolver(zInsertCurrencyParams) })
  const create = useMutation({
    ...createCurrencyMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listCurrenciesQueryKey() })
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => create.mutate({ body }))}>
      <FieldGroup>
        <TextField label="Code" placeholder="EUR" error={form.formState.errors.code} {...form.register("code")} />
        <TextField label="Name" placeholder="Euro" error={form.formState.errors.name} {...form.register("name")} />
        <TextField label="Symbol" placeholder="€" error={form.formState.errors.symbol} {...form.register("symbol")} />
        <TextField
          label="Decimal places"
          description="2 if left empty; 0 for a currency without cents, such as JPY."
          type="number"
          min={0}
          error={form.formState.errors.precision}
          {...form.register("precision", { setValueAs: (value: string) => (value === "" ? undefined : Number(value)) })}
        />
        <ProblemMessages error={create.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={create.isPending}>
          Add currency
        </Button>
      </DialogFooter>
    </form>
  )
}

// EditCurrencyDialog changes a currency's code, name, or symbol.
export function EditCurrencyDialog({ currency, onOpenChange }: { currency: Currency | null } & Omit<DialogProps, "open">) {
  return (
    <Dialog open={currency !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit {currency?.code}</DialogTitle>
          <DialogDescription>Accounts and postings refer to the currency by id, so they follow the change.</DialogDescription>
        </DialogHeader>
        {currency && <EditCurrencyForm currency={currency} onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function EditCurrencyForm({ currency, onDone }: { currency: Currency; onDone: () => void }) {
  const queryClient = useQueryClient()
  const form = useForm({
    resolver: zodResolver(zCurrencyChanges),
    defaultValues: { code: currency.code, name: currency.name, symbol: currency.symbol ?? "" },
  })
  const update = useMutation({
    ...updateCurrencyMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listCurrenciesQueryKey() })
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => update.mutate({ path: { id: currency.id }, body }))}>
      <FieldGroup>
        <TextField label="Code" error={form.formState.errors.code} {...form.register("code")} />
        <TextField label="Name" error={form.formState.errors.name} {...form.register("name")} />
        <TextField label="Symbol" error={form.formState.errors.symbol} {...form.register("symbol")} />
        <ProblemMessages error={update.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={update.isPending}>
          Save
        </Button>
      </DialogFooter>
    </form>
  )
}

// DeleteCurrencyDialog deletes a currency, which the gateway refuses while
// anything, such as an account, still uses it.
export function DeleteCurrencyDialog({ currency, onOpenChange }: { currency: Currency | null } & Omit<DialogProps, "open">) {
  const queryClient = useQueryClient()
  const remove = useMutation({
    ...deleteCurrencyMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listCurrenciesQueryKey() })
      onOpenChange(false)
    },
  })

  return (
    <AlertDialog
      open={currency !== null}
      onOpenChange={(open) => {
        if (!open) {
          remove.reset()
        }
        onOpenChange(open)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {currency?.code}?</AlertDialogTitle>
          <AlertDialogDescription>Only a currency that no account uses can be deleted.</AlertDialogDescription>
        </AlertDialogHeader>
        <ProblemMessages error={remove.error} />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={remove.isPending}
            onClick={(event) => {
              // Keep the dialog open until the gateway answers, to show why
              // it refused.
              event.preventDefault()
              if (currency) {
                remove.mutate({ path: { id: currency.id } })
              }
            }}
          >
            Delete currency
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
