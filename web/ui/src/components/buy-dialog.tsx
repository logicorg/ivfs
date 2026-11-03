import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm, useWatch } from "react-hook-form"

import type { NegativeHolding } from "@/api"
import {
  buyAssetMutation,
  createAssetMutation,
  getAccountBalancesOptions,
  listAssetsOptions,
  listFinancialAccountsOptions,
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
import {
  assetKinds,
  assetLabel,
  newAsset,
  newAssetForm,
  newBuy,
  tradeCurrencies,
  tradeForm,
  type TradeFormValues,
} from "@/lib/trade"

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// BuyDialog records units of an asset bought with a brokerage account's cash,
// creating the asset first when it is new.
export function BuyDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Buy</DialogTitle>
          <DialogDescription>Units of an asset paid for with the account's cash, fee included in the cost.</DialogDescription>
        </DialogHeader>
        {open && <RecordBuyForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

// RecordBuyForm makes its idempotency key when it mounts, once per opening of
// the dialog, so a submit sent twice records the buy once.
function RecordBuyForm({ onDone }: { onDone: () => void }) {
  const [idempotencyKey] = useState(() => crypto.randomUUID())
  // warnings are those of the buy just recorded, read before the dialog closes.
  const [warnings, setWarnings] = useState<NegativeHolding[]>()
  const queryClient = useQueryClient()
  const accounts = useQuery(listFinancialAccountsOptions())
  const assets = useQuery(listAssetsOptions())
  const form = useForm<TradeFormValues>({
    resolver: zodResolver(tradeForm),
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
  const balances = useQuery({
    ...getAccountBalancesOptions({ path: { id: Number(financialAccountId) } }),
    enabled: financialAccountId !== "",
  })

  const invalidate = (...ids: string[]) => {
    for (const _id of ids) {
      void queryClient.invalidateQueries({ queryKey: [{ _id }] })
    }
  }
  // A new asset is created even when the buy then fails; trying again finds it
  // by name rather than making another.
  const createAsset = useMutation({ ...createAssetMutation(), onSuccess: () => invalidate("listAssets") })
  const buyAsset = useMutation({
    ...buyAssetMutation(),
    onSuccess: (trade) => {
      invalidate(
        "listHoldings",
        "listHoldingMovements",
        "getAccountBalances",
        "getLedgerAccountBalance",
        "listJournalEntries",
      )
      if (trade.warnings?.length) {
        setWarnings(trade.warnings)
        return
      }
      onDone()
    },
  })

  const submit = form.handleSubmit(async (values) => {
    let id = Number(values.assetId)
    if (values.assetId === newAsset) {
      try {
        id = (await createAsset.mutateAsync({ body: newAssetForm(values) })).id
      } catch {
        return
      }
    }
    buyAsset.mutate({ body: newBuy(values, id, idempotencyKey, new Date()) })
  })

  if (warnings) {
    return (
      <div className="flex flex-col gap-4">
        <NegativeHoldingWarnings title="The buy is recorded, and leaves a holding negative" warnings={warnings} />
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
  const assetOptions: SelectOption[] = [
    ...[...(assets.data?.assets ?? [])]
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((asset) => ({ value: String(asset.id), label: assetLabel(asset) })),
    { value: newAsset, label: "New asset…" },
  ]
  const pending = createAsset.isPending || buyAsset.isPending

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
              label="Paid in"
              description="The account's Cash pays, and its Investments holds the cost, in this currency."
              value={field.value || undefined}
              onChange={field.onChange}
              options={currencyOptions}
              placeholder={financialAccountId === "" ? "Choose an account first" : "Choose…"}
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
        {assetId === newAsset && (
          <>
            <TextField
              label="Asset name"
              placeholder="Vanguard FTSE All-World"
              autoComplete="off"
              error={form.formState.errors.assetName}
              {...form.register("assetName")}
            />
            <div className="flex items-start gap-2">
              <div className="flex-1">
                <TextField
                  label="Ticker"
                  placeholder="VWCE"
                  autoComplete="off"
                  error={form.formState.errors.assetTicker}
                  {...form.register("assetTicker")}
                />
              </div>
              <div className="flex-1">
                <Controller
                  control={form.control}
                  name="assetKind"
                  render={({ field, fieldState }) => (
                    <SelectField
                      id="assetKind"
                      label="Kind"
                      value={field.value || undefined}
                      onChange={field.onChange}
                      options={assetKinds}
                      placeholder="Choose…"
                      error={fieldState.error}
                    />
                  )}
                />
              </div>
            </div>
          </>
        )}
        <TextField
          label="Units"
          placeholder="12.5"
          inputMode="decimal"
          error={form.formState.errors.units}
          {...form.register("units")}
        />
        <div className="flex items-start gap-2">
          <div className="flex-1">
            <TextField
              label="Price paid"
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
          placeholder="Monthly buy"
          autoComplete="off"
          error={form.formState.errors.description}
          {...form.register("description")}
        />
        <ProblemMessages error={createAsset.error ?? buyAsset.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={pending}>
          Buy
        </Button>
      </DialogFooter>
    </form>
  )
}
