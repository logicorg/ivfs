import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm } from "react-hook-form"

import type { FinancialAccount } from "@/api"
import { addBalanceMutation, listCurrenciesOptions } from "@/api/@tanstack/react-query.gen"
import { ProblemMessages, SelectField, TextField } from "@/components/form-fields"
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
import { addBalanceForm, newBalanceForm, type AddBalanceFormValues } from "@/lib/add-balance"

type DialogProps = {
  account: FinancialAccount
  open: boolean
  onOpenChange: (open: boolean) => void
}

// AddBalanceDialog adds a named balance, such as a savings vault, to a
// financial account.
export function AddBalanceDialog({ account, open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a balance to {account.name}</DialogTitle>
          <DialogDescription>
            Money kept apart in the account, such as a savings vault. Adding one that exists changes nothing.
          </DialogDescription>
        </DialogHeader>
        {open && <AddBalanceForm account={account} onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function AddBalanceForm({ account, onDone }: { account: FinancialAccount; onDone: () => void }) {
  const queryClient = useQueryClient()
  const currencies = useQuery(listCurrenciesOptions())
  const form = useForm<AddBalanceFormValues>({
    resolver: zodResolver(addBalanceForm),
    defaultValues: { name: "", currencyCode: "", openingBalance: "", openingBalanceAt: "" },
  })

  const addBalance = useMutation({
    ...addBalanceMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [{ _id: "getAccountBalances" }] })
      onDone()
    },
  })

  const currencyOptions = (currencies.data?.currencies ?? []).map((currency) => ({
    value: currency.code,
    label: `${currency.code} · ${currency.name}`,
  }))

  return (
    <form
      onSubmit={form.handleSubmit((values) =>
        addBalance.mutate({ path: { id: account.id }, body: newBalanceForm(values) }),
      )}
    >
      <FieldGroup>
        <TextField
          label="Name"
          placeholder="Savings vault"
          autoComplete="off"
          error={form.formState.errors.name}
          {...form.register("name")}
        />
        <Controller
          control={form.control}
          name="currencyCode"
          render={({ field, fieldState }) => (
            <SelectField
              id="currencyCode"
              label="Currency"
              value={field.value}
              onChange={field.onChange}
              options={currencyOptions}
              placeholder="Choose…"
              error={fieldState.error}
            />
          )}
        />
        <TextField
          label="Balance today"
          placeholder="0"
          inputMode="decimal"
          error={form.formState.errors.openingBalance}
          {...form.register("openingBalance")}
        />
        <TextField
          label="Balance as of"
          description="Today if left empty."
          type="date"
          error={form.formState.errors.openingBalanceAt}
          {...form.register("openingBalanceAt")}
        />
        <ProblemMessages error={addBalance.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={addBalance.isPending}>
          Add balance
        </Button>
      </DialogFooter>
    </form>
  )
}
