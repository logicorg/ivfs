import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useFieldArray, useForm, useWatch } from "react-hook-form"

import type { AccountForm } from "@/api"
import {
  listCurrenciesOptions,
  listFinancialAccountsOptions,
  openAccountMutation,
} from "@/api/@tanstack/react-query.gen"
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
import { FieldError, FieldGroup, FieldLegend, FieldSet } from "@/components/ui/field"
import { newAccountForm, openAccountForm, type OpenAccountFormValues } from "@/lib/open-account"

const accountTypes = [
  { value: "bank", label: "Bank account" },
  { value: "card", label: "Credit card" },
  { value: "brokerage", label: "Brokerage account" },
  { value: "cash", label: "Cash" },
]

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// OpenAccountDialog opens a financial account in one or more currencies. An
// institution that does not exist yet is created only once the person
// confirms its name, so a typo does not become a new bank.
export function OpenAccountDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Open an account</DialogTitle>
          <DialogDescription>
            Opening an account that exists adds the currencies it does not hold yet.
          </DialogDescription>
        </DialogHeader>
        {open && <OpenAccountForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function OpenAccountForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient()
  const currencies = useQuery(listCurrenciesOptions())
  const accounts = useQuery(listFinancialAccountsOptions({ query: { includeClosed: false } }))
  const form = useForm<OpenAccountFormValues>({
    resolver: zodResolver(openAccountForm),
    defaultValues: {
      type: "bank",
      institutionName: "",
      accountName: "",
      currencies: [{ code: "", openingBalance: "" }],
      openingBalancesAt: "",
    },
  })
  const rows = useFieldArray({ control: form.control, name: "currencies" })
  const type = useWatch({ control: form.control, name: "type" })
  // unconfirmed is the form whose institution was not found, waiting for
  // the person to confirm that it should be created.
  const [unconfirmed, setUnconfirmed] = useState<AccountForm | null>(null)

  const openAccount = useMutation({
    ...openAccountMutation(),
    onSuccess: (result, { body }) => {
      if (result.outcome === "institution_not_found") {
        setUnconfirmed(body)
        return
      }
      for (const _id of ["listFinancialAccounts", "getAccountBalances"]) {
        void queryClient.invalidateQueries({ queryKey: [{ _id }] })
      }
      onDone()
    },
  })

  const institutions = [...new Set(accounts.data?.financialAccounts.map((account) => account.institutionName))].sort()
  const currencyOptions = (currencies.data?.currencies ?? []).map((currency) => ({
    value: currency.code,
    label: `${currency.code} · ${currency.name}`,
  }))

  if (unconfirmed) {
    return (
      <div className="flex flex-col gap-4">
        <p>
          There is no institution called <strong>{unconfirmed.institutionName}</strong> yet. Create it and open the
          account there?
        </p>
        <ProblemMessages error={openAccount.error} />
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              openAccount.reset()
              setUnconfirmed(null)
            }}
          >
            Back
          </Button>
          <Button
            disabled={openAccount.isPending}
            onClick={() => openAccount.mutate({ body: { ...unconfirmed, createInstitution: true } })}
          >
            Create {unconfirmed.institutionName}
          </Button>
        </DialogFooter>
      </div>
    )
  }

  return (
    <form onSubmit={form.handleSubmit((values) => openAccount.mutate({ body: newAccountForm(values) }))}>
      <FieldGroup>
        <Controller
          control={form.control}
          name="type"
          render={({ field, fieldState }) => (
            <SelectField
              id="type"
              label="Type"
              value={field.value}
              onChange={field.onChange}
              options={accountTypes}
              error={fieldState.error}
            />
          )}
        />
        {type !== "cash" && (
          <>
            <TextField
              label="Institution"
              placeholder="Revolut"
              list="institutions"
              autoComplete="off"
              error={form.formState.errors.institutionName}
              {...form.register("institutionName")}
            />
            <datalist id="institutions">
              {institutions.map((name) => (
                <option key={name} value={name} />
              ))}
            </datalist>
          </>
        )}
        <TextField
          label="Account name"
          description={type === "cash" ? "Wallet if left empty." : "The institution's name if left empty."}
          error={form.formState.errors.accountName}
          {...form.register("accountName")}
        />
        <FieldSet>
          <FieldLegend variant="label">Currencies</FieldLegend>
          {rows.fields.map((row, index) => (
            <div key={row.id} className="flex items-start gap-2">
              <div className="flex-1">
                <Controller
                  control={form.control}
                  name={`currencies.${index}.code`}
                  render={({ field, fieldState }) => (
                    <SelectField
                      id={`currencies.${index}.code`}
                      label="Currency"
                      value={field.value}
                      onChange={field.onChange}
                      options={currencyOptions}
                      placeholder="Choose…"
                      error={fieldState.error}
                    />
                  )}
                />
              </div>
              <div className="flex-1">
                <TextField
                  label={type === "card" ? "Owed today" : "Balance today"}
                  placeholder="0"
                  inputMode="decimal"
                  error={form.formState.errors.currencies?.[index]?.openingBalance}
                  {...form.register(`currencies.${index}.openingBalance`)}
                />
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="mt-7"
                disabled={rows.fields.length === 1}
                onClick={() => rows.remove(index)}
              >
                Remove
              </Button>
            </div>
          ))}
          <FieldError errors={[form.formState.errors.currencies?.root]} />
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="self-start"
            onClick={() => rows.append({ code: "", openingBalance: "" })}
          >
            Add a currency
          </Button>
        </FieldSet>
        <TextField
          label="Balances as of"
          description="Today if left empty."
          type="date"
          error={form.formState.errors.openingBalancesAt}
          {...form.register("openingBalancesAt")}
        />
        <ProblemMessages error={openAccount.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={openAccount.isPending}>
          Open account
        </Button>
      </DialogFooter>
    </form>
  )
}
