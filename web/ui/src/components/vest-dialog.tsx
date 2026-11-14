import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm, useWatch } from "react-hook-form"

import type { NegativeHolding } from "@/api"
import {
  createAssetMutation,
  getAccountBalancesOptions,
  listAssetsOptions,
  listCategoriesOptions,
  listFinancialAccountsOptions,
  sellAssetMutation,
  vestAssetMutation,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages, SelectField, TextField, type SelectOption } from "@/components/form-fields"
import { NegativeHoldingWarnings } from "@/components/negative-holding-warnings"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { categoryChoices } from "@/lib/record-transaction"
import { assetKinds, assetLabel, newAsset, newAssetForm, tradeCurrencies } from "@/lib/trade"
import { newSellToCover, newVest, vestForm, type VestFormValues } from "@/lib/vest"

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// VestDialog records units of an asset received as compensation, creating the
// asset first when it is new, and the sale that covers its tax when asked.
export function VestDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Vest</DialogTitle>
          <DialogDescription>
            Units of an asset received as compensation, such as RSUs, booked as income at their value.
          </DialogDescription>
        </DialogHeader>
        {open && <RecordVestForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

// RecordVestForm makes the idempotency keys of the vest and of its sale when
// it mounts, once per opening of the dialog, so a submit sent twice records
// each once. The sale is sent only once the vest is recorded (ADR 0009); when
// it then fails, the vest's fields lock and submitting again sends the sale
// alone.
function RecordVestForm({ onDone }: { onDone: () => void }) {
  const [keys] = useState(() => ({ vest: crypto.randomUUID(), sale: crypto.randomUUID() }))
  // vestedAssetId is the asset of the vest once it is recorded, the new
  // asset's when one was created.
  const [vestedAssetId, setVestedAssetId] = useState<number>()
  // warnings are those of the sale just recorded, read before the dialog closes.
  const [warnings, setWarnings] = useState<NegativeHolding[]>()
  const queryClient = useQueryClient()
  const accounts = useQuery(listFinancialAccountsOptions())
  const assets = useQuery(listAssetsOptions())
  const categories = useQuery(listCategoriesOptions())
  const form = useForm<VestFormValues>({
    resolver: zodResolver(vestForm),
    defaultValues: {
      financialAccountId: "",
      currencyCode: "",
      assetId: "",
      assetName: "",
      assetTicker: "",
      assetKind: "",
      unitsVested: "",
      unitsWithheld: "",
      fairMarketValue: "",
      incomeCategoryId: "",
      taxCategoryId: "",
      occurredOn: "",
      description: "",
      sellToCover: false,
      soldUnits: "",
      soldAmount: "",
      soldFee: "",
    },
  })
  const [financialAccountId, currencyCode, assetId, sellToCover] = useWatch({
    control: form.control,
    name: ["financialAccountId", "currencyCode", "assetId", "sellToCover"],
  })
  const balances = useQuery({
    ...getAccountBalancesOptions({ path: { id: Number(financialAccountId) } }),
    enabled: financialAccountId !== "",
  })

  const invalidate = (...ids: string[]) => {
    for (const _id of ids) {
      void queryClient.invalidateQueries({ queryKey: [{ _id }] })
    }
  }
  const recorded = () =>
    invalidate(
      "listHoldings",
      "listHoldingMovements",
      "getAccountBalances",
      "getLedgerAccountBalance",
      "listJournalEntries",
      "getCategoryTotals",
    )
  // A new asset is created even when the vest then fails; trying again finds
  // it by name rather than making another.
  const createAsset = useMutation({ ...createAssetMutation(), onSuccess: () => invalidate("listAssets") })
  const vestAsset = useMutation({ ...vestAssetMutation(), onSuccess: recorded })
  const sellAsset = useMutation({ ...sellAssetMutation(), onSuccess: recorded })

  const submit = form.handleSubmit(async (values) => {
    const now = new Date()
    let id = vestedAssetId
    if (id === undefined) {
      try {
        id =
          values.assetId === newAsset
            ? (await createAsset.mutateAsync({ body: newAssetForm(values) })).id
            : Number(values.assetId)
        await vestAsset.mutateAsync({ body: newVest(values, id, keys.vest, now) })
      } catch {
        return
      }
      setVestedAssetId(id)
    }
    if (!values.sellToCover) {
      onDone()
      return
    }
    try {
      const sale = await sellAsset.mutateAsync({ body: newSellToCover(values, id, keys.sale, now) })
      if (sale.warnings?.length) {
        setWarnings(sale.warnings)
        return
      }
    } catch {
      return
    }
    onDone()
  })

  if (warnings) {
    return (
      <div className="flex flex-col gap-4">
        <NegativeHoldingWarnings
          title="The vest and its sale are recorded, and the sale leaves a holding negative"
          warnings={warnings}
        />
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
  // Only in the vest's currency, the one category and Investments share.
  const categoryOptions = (kind: "income" | "expense"): SelectOption[] =>
    currencyCode === ""
      ? []
      : categoryChoices(categories.data?.categories ?? [], kind, currencyCode).map((choice) => ({
          value: String(choice.ledgerAccountId),
          label: choice.label,
          depth: choice.depth,
        }))
  const categoryPlaceholder = currencyCode === "" ? "Choose a currency first" : "Choose…"
  const vested = vestedAssetId !== undefined
  const pending = createAsset.isPending || vestAsset.isPending || sellAsset.isPending

  return (
    <form onSubmit={submit}>
      <FieldGroup>
        <fieldset disabled={vested} className="contents">
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
                  form.resetField("incomeCategoryId")
                  form.resetField("taxCategoryId")
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
                label="Valued in"
                description="The account's Investments holds the value, and the categories take it, in this currency."
                value={field.value || undefined}
                onChange={(code) => {
                  field.onChange(code)
                  form.resetField("incomeCategoryId")
                  form.resetField("taxCategoryId")
                }}
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
                placeholder="Acme Corp"
                autoComplete="off"
                error={form.formState.errors.assetName}
                {...form.register("assetName")}
              />
              <div className="flex items-start gap-2">
                <div className="flex-1">
                  <TextField
                    label="Ticker"
                    placeholder="ACME"
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
          <div className="flex items-start gap-2">
            <div className="flex-1">
              <TextField
                label="Units vested"
                description="Those withheld included."
                placeholder="100"
                inputMode="decimal"
                error={form.formState.errors.unitsVested}
                {...form.register("unitsVested")}
              />
            </div>
            <div className="flex-1">
              <TextField
                label="Units withheld"
                description="Kept for tax; none if left empty."
                placeholder="40"
                inputMode="decimal"
                error={form.formState.errors.unitsWithheld}
                {...form.register("unitsWithheld")}
              />
            </div>
          </div>
          <TextField
            label="Value of a unit"
            description="Its fair market value on the vesting date."
            placeholder="182.40"
            inputMode="decimal"
            error={form.formState.errors.fairMarketValue}
            {...form.register("fairMarketValue")}
          />
          <Controller
            control={form.control}
            name="incomeCategoryId"
            render={({ field, fieldState }) => (
              <SelectField
                id="incomeCategoryId"
                label="Income category"
                description="Takes the value of every unit vested."
                value={field.value || undefined}
                onChange={field.onChange}
                options={categoryOptions("income")}
                placeholder={categoryPlaceholder}
                error={fieldState.error}
              />
            )}
          />
          <Controller
            control={form.control}
            name="taxCategoryId"
            render={({ field, fieldState }) => (
              <SelectField
                id="taxCategoryId"
                label="Tax category"
                description="Takes the value of the units withheld; needed only when some are."
                value={field.value || undefined}
                onChange={field.onChange}
                options={categoryOptions("expense")}
                placeholder={categoryPlaceholder}
                error={fieldState.error}
              />
            )}
          />
          <TextField
            label="Date"
            description="Now if left empty."
            type="date"
            error={form.formState.errors.occurredOn}
            {...form.register("occurredOn")}
          />
          <TextField
            label="Note"
            placeholder="Q1 vest"
            autoComplete="off"
            error={form.formState.errors.description}
            {...form.register("description")}
          />
        </fieldset>
        <Controller
          control={form.control}
          name="sellToCover"
          render={({ field }) => (
            <Field orientation="horizontal">
              <Checkbox id="sellToCover" checked={field.value} onCheckedChange={(checked) => field.onChange(checked === true)} />
              <FieldLabel htmlFor="sellToCover">Some units were sold to cover the tax</FieldLabel>
            </Field>
          )}
        />
        {sellToCover && (
          <>
            <FieldDescription>
              Recorded as a sale of the units delivered, once the vest is. Its gain is against the holding's average
              cost.
            </FieldDescription>
            <TextField
              label="Units sold"
              placeholder="12"
              inputMode="decimal"
              error={form.formState.errors.soldUnits}
              {...form.register("soldUnits")}
            />
            <div className="flex items-start gap-2">
              <div className="flex-1">
                <TextField
                  label="Price received"
                  description="For all the units, before the fee."
                  placeholder="2188.80"
                  inputMode="decimal"
                  error={form.formState.errors.soldAmount}
                  {...form.register("soldAmount")}
                />
              </div>
              <div className="flex-1">
                <TextField
                  label="Fee"
                  description="None if left empty."
                  placeholder="1.50"
                  inputMode="decimal"
                  error={form.formState.errors.soldFee}
                  {...form.register("soldFee")}
                />
              </div>
            </div>
          </>
        )}
        {vested && sellToCover && (
          <p className="text-sm text-muted-foreground">
            The vest is recorded. Fix the sale and submit again to record it, or close to leave it out.
          </p>
        )}
        <ProblemMessages error={createAsset.error ?? vestAsset.error ?? sellAsset.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={pending}>
          {vested ? "Record the sale" : "Vest"}
        </Button>
      </DialogFooter>
    </form>
  )
}
