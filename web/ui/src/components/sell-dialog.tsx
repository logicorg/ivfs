import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm, useWatch } from "react-hook-form"

import type { NegativeHolding } from "@/api"
import {
  getAccountBalancesOptions,
  listAssetsOptions,
  listFinancialAccountsOptions,
  listHoldingsOptions,
  sellAssetMutation,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages, SelectField, TextField, type SelectOption } from "@/components/form-fields"
import { NegativeHoldingWarnings } from "@/components/negative-holding-warnings"
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
import { formatUnits } from "@/lib/holdings"
import { heldUnits, newSell, saleAssets, saleForm, tradeCurrencies, type TradeFormValues } from "@/lib/trade"

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// SellDialog records units of an asset sold for a brokerage account's cash.
export function SellDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Sell</DialogTitle>
          <DialogDescription>Units of an asset sold for the account's cash, fee taken from the proceeds.</DialogDescription>
        </DialogHeader>
        {open && <RecordSellForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

// RecordSellForm makes its idempotency key when it mounts, once per opening of
// the dialog, so a submit sent twice records the sale once.
function RecordSellForm({ onDone }: { onDone: () => void }) {
  const [idempotencyKey] = useState(() => crypto.randomUUID())
  // warnings are those of the sale just recorded, read before the dialog closes.
  const [warnings, setWarnings] = useState<NegativeHolding[]>()
  const queryClient = useQueryClient()
  const accounts = useQuery(listFinancialAccountsOptions())
  const assets = useQuery(listAssetsOptions())
  const form = useForm<TradeFormValues>({
    resolver: zodResolver(saleForm),
    defaultValues: {
      financialAccountId: "",
      currencyCode: "",
      assetId: "",
      assetName: "",
      assetTicker: "",
      assetKind: "",
      units: "",
      amount: "",
      fee: "",
      occurredOn: "",
      description: "",
    },
  })
  const [financialAccountId, assetId] = useWatch({ control: form.control, name: ["financialAccountId", "assetId"] })
  const accountChosen = financialAccountId !== ""
  const balances = useQuery({
    ...getAccountBalancesOptions({ path: { id: Number(financialAccountId) } }),
    enabled: accountChosen,
  })
  const holdings = useQuery({
    ...listHoldingsOptions({ query: { financialAccountId: Number(financialAccountId) } }),
    enabled: accountChosen,
  })

  const sellAsset = useMutation({
    ...sellAssetMutation(),
    onSuccess: (trade) => {
      for (const _id of [
        "listHoldings",
        "listHoldingMovements",
        "getAccountBalances",
        "getLedgerAccountBalance",
        "listJournalEntries",
      ]) {
        void queryClient.invalidateQueries({ queryKey: [{ _id }] })
      }
      if (trade.warnings?.length) {
        setWarnings(trade.warnings)
        return
      }
      onDone()
    },
  })

  const submit = form.handleSubmit((values) => {
    sellAsset.mutate({ body: newSell(values, idempotencyKey, new Date()) })
  })

  if (warnings) {
    return (
      <div className="flex flex-col gap-4">
        <NegativeHoldingWarnings title="The sale is recorded, and leaves a holding negative" warnings={warnings} />
        <DialogFooter>
          <Button onClick={onDone}>Done</Button>
        </DialogFooter>
      </div>
    )
  }

  const accountOptions: SelectOption[] = (accounts.data?.financialAccounts ?? [])
    .filter((account) => !account.closed)
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((account) => ({ value: String(account.id), label: `${account.name} · ${account.institutionName}` }))
  const currencyOptions: SelectOption[] = tradeCurrencies(balances.data?.balances ?? []).map((code) => ({
    value: code,
    label: code,
  }))
  const assetOptions = saleAssets(assets.data?.assets ?? [], holdings.data?.holdings ?? [])
  // Only once the account's holdings are in, so "0 held" is never a guess.
  const held =
    accountChosen && assetId !== "" && holdings.isSuccess
      ? `${formatUnits(heldUnits(holdings.data.holdings, assetId) ?? "0")} held in this account.`
      : undefined

  return (
    <form onSubmit={submit}>
      <FieldGroup>
        <Controller
          control={form.control}
          name="financialAccountId"
          render={({ field, fieldState }) => (
            <SelectField
              id="financialAccountId"
              label="Account"
              value={field.value || undefined}
              onChange={(id) => {
                field.onChange(id)
                form.resetField("currencyCode")
              }}
              options={accountOptions}
              placeholder="Choose…"
              error={fieldState.error}
            />
          )}
        />
        <Controller
          control={form.control}
          name="currencyCode"
          render={({ field, fieldState }) => (
            <SelectField
              id="currencyCode"
              label="Received in"
              description="The account's Investments pays, and its Cash receives, in this currency."
              value={field.value || undefined}
              onChange={field.onChange}
              options={currencyOptions}
              placeholder={accountChosen ? "Choose…" : "Choose an account first"}
              error={fieldState.error}
            />
          )}
        />
        <Controller
          control={form.control}
          name="assetId"
          render={({ field, fieldState }) => (
            <SelectField
              id="assetId"
              label="Asset"
              value={field.value || undefined}
              onChange={field.onChange}
              options={assetOptions}
              placeholder="Choose…"
              error={fieldState.error}
            />
          )}
        />
        <TextField
          label="Units"
          description={held}
          placeholder="12.5"
          inputMode="decimal"
          error={form.formState.errors.units}
          {...form.register("units")}
        />
        <div className="flex items-start gap-2">
          <div className="flex-1">
            <TextField
              label="Price received"
              description="For all the units, before the fee."
              placeholder="1200.50"
              inputMode="decimal"
              error={form.formState.errors.amount}
              {...form.register("amount")}
            />
          </div>
          <div className="flex-1">
            <TextField
              label="Fee"
              description="None if left empty."
              placeholder="1.50"
              inputMode="decimal"
              error={form.formState.errors.fee}
              {...form.register("fee")}
            />
          </div>
        </div>
        <TextField
          label="Date"
          description="Now if left empty."
          type="date"
          error={form.formState.errors.occurredOn}
          {...form.register("occurredOn")}
        />
        <TextField
          label="Note"
          placeholder="Rebalance"
          autoComplete="off"
          error={form.formState.errors.description}
          {...form.register("description")}
        />
        <ProblemMessages error={sellAsset.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={sellAsset.isPending}>
          Sell
        </Button>
      </DialogFooter>
    </form>
  )
}
