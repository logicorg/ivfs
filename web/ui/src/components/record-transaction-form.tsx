import { useState, type ComponentType, type ReactNode } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm } from "react-hook-form"

import {
  getAccountBalancesOptions,
  listCategoriesOptions,
  listFinancialAccountsOptions,
  recordExpenseMutation,
  recordIncomeMutation,
  recordTransferMutation,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages, SelectField, TextField, type SelectOption } from "@/components/form-fields"
import { Button } from "@/components/ui/button"
import { FieldGroup } from "@/components/ui/field"
import {
  balanceChoices,
  crossesCurrencies,
  newTransaction,
  offers,
  pickers,
  recordTransactionForm,
  type Choice,
  type RecordTransactionFormValues,
  type TransactionType,
} from "@/lib/record-transaction"

const typeOptions: SelectOption[] = [
  { value: "expense", label: "Expense" },
  { value: "income", label: "Income" },
  { value: "transfer", label: "Transfer" },
]

type RecordTransactionFormProps = {
  // type fixes the transaction's type and hides the select; the person
  // chooses it when it is omitted.
  type?: TransactionType
  onRecorded: () => void
  // Footer holds the submit button, laid out as the form's container lays
  // out its actions.
  Footer: ComponentType<{ children: ReactNode }>
}

// RecordTransactionForm records an expense, an income, or a transfer, the
// type deciding which accounts each side offers (ADR 0008). It makes its
// idempotency key when it mounts, so a submit sent twice records the
// transaction once; remount it to record another.
export function RecordTransactionForm({ type, onRecorded, Footer }: RecordTransactionFormProps) {
  const [idempotencyKey] = useState(() => crypto.randomUUID())
  const queryClient = useQueryClient()
  const accounts = useQuery(listFinancialAccountsOptions())
  const accountBalances = useQueries({
    queries: (accounts.data?.financialAccounts ?? []).map((account) =>
      getAccountBalancesOptions({ path: { id: account.id } }),
    ),
  })
  const categories = useQuery(listCategoriesOptions())
  const form = useForm<RecordTransactionFormValues>({
    resolver: zodResolver(recordTransactionForm),
    defaultValues: { type: type ?? "expense", fromId: "", toId: "", amount: "", toAmount: "", occurredOn: "", description: "" },
  })

  const onSuccess = () => {
    for (const _id of ["getAccountBalances", "getLedgerAccountBalance", "listJournalEntries", "getCategoryTotals"]) {
      void queryClient.invalidateQueries({ queryKey: [{ _id }] })
    }
    onRecorded()
  }
  const recordExpense = useMutation({ ...recordExpenseMutation(), onSuccess })
  const recordIncome = useMutation({ ...recordIncomeMutation(), onSuccess })
  const recordTransfer = useMutation({ ...recordTransferMutation(), onSuccess })
  const recording = { expense: recordExpense, income: recordIncome, transfer: recordTransfer }

  const balances = balanceChoices(
    accounts.data?.financialAccounts ?? [],
    accountBalances.flatMap((query) => (query.data ? [query.data] : [])),
  )
  const allCategories = categories.data?.categories ?? []
  const values = form.watch()
  const sides = pickers(values.type, balances, allCategories, values.fromId, values.toId)
  const crossing = crossesCurrencies(values, balances)
  const mutation = recording[values.type]

  const submit = form.handleSubmit((submitted) => {
    if (crossing && submitted.toAmount === "") {
      form.setError("toAmount", { message: "The amount that arrives, in its currency" })
      return
    }
    const request = newTransaction(submitted, idempotencyKey, crossing, new Date())
    switch (request.type) {
      case "expense":
        return recordExpense.mutate({ body: request.body })
      case "income":
        return recordIncome.mutate({ body: request.body })
      case "transfer":
        return recordTransfer.mutate({ body: request.body })
    }
  })

  return (
    <form onSubmit={submit}>
      <FieldGroup>
        {type === undefined && (
          <Controller
            control={form.control}
            name="type"
            render={({ field }) => (
              <SelectField
                id="type"
                label="Type"
                value={field.value}
                onChange={(chosen) => {
                  field.onChange(chosen as TransactionType)
                  form.resetField("fromId")
                  form.resetField("toId")
                  form.resetField("toAmount")
                }}
                options={typeOptions}
              />
            )}
          />
        )}
        <Controller
          control={form.control}
          name="fromId"
          render={({ field, fieldState }) => (
            <SelectField
              id="fromId"
              label={sides.from.label}
              value={field.value}
              onChange={(fromId) => {
                field.onChange(fromId)
                // Only a choice the new one rules out is cleared, so the
                // sides can be chosen in either order.
                if (!offers(pickers(values.type, balances, allCategories, fromId, values.toId).to, values.toId)) {
                  form.resetField("toId")
                }
              }}
              options={choiceOptions(sides.from.choices)}
              placeholder="Choose…"
              error={fieldState.error}
            />
          )}
        />
        <Controller
          control={form.control}
          name="toId"
          render={({ field, fieldState }) => (
            <SelectField
              id="toId"
              label={sides.to.label}
              value={field.value}
              onChange={(toId) => {
                field.onChange(toId)
                if (!offers(pickers(values.type, balances, allCategories, values.fromId, toId).from, values.fromId)) {
                  form.resetField("fromId")
                }
              }}
              options={choiceOptions(sides.to.choices)}
              placeholder="Choose…"
              error={fieldState.error}
            />
          )}
        />
        <TextField
          label={crossing ? "Amount that leaves" : "Amount"}
          placeholder="12.50"
          inputMode="decimal"
          error={form.formState.errors.amount}
          {...form.register("amount")}
        />
        {crossing && (
          <TextField
            label="Amount that arrives"
            description="What the destination receives, in its currency. The exchange has no fee."
            placeholder="13.60"
            inputMode="decimal"
            error={form.formState.errors.toAmount}
            {...form.register("toAmount")}
          />
        )}
        <TextField
          label="Date"
          description="Now if left empty."
          type="date"
          error={form.formState.errors.occurredOn}
          {...form.register("occurredOn")}
        />
        <TextField
          label="Note"
          placeholder="Weekly shop"
          autoComplete="off"
          error={form.formState.errors.description}
          {...form.register("description")}
        />
        <ProblemMessages error={mutation.error} />
      </FieldGroup>
      <Footer>
        <Button type="submit" disabled={mutation.isPending}>
          Record
        </Button>
      </Footer>
    </form>
  )
}

// choiceOptions shows each choice with its currency, which decides what the
// other side may be.
function choiceOptions(choices: Choice[]): SelectOption[] {
  return choices.map((choice) => ({
    value: String(choice.ledgerAccountId),
    label: `${choice.label} · ${choice.currencyCode}`,
    depth: choice.depth,
  }))
}
