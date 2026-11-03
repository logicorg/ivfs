import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

import type { Currency, UserSettings } from "@/api"
import {
  listCurrenciesOptions,
  retrieveUserSettingsOptions,
  retrieveUserSettingsQueryKey,
  updateUserSettingsMutation,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export const Route = createFileRoute("/settings")({
  component: Settings,
})

function Settings() {
  const settings = useQuery(retrieveUserSettingsOptions())
  const currencies = useQuery(listCurrenciesOptions())

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">Settings</h1>
      {settings.isPending || currencies.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : settings.isError ? (
        <ProblemMessages error={settings.error} />
      ) : currencies.isError ? (
        <ProblemMessages error={currencies.error} />
      ) : (
        // Keyed so a save elsewhere, such as another tab's, starts the form
        // over from what is stored.
        <SettingsForm
          key={JSON.stringify(settings.data)}
          settings={settings.data}
          currencies={currencies.data.currencies}
        />
      )}
    </div>
  )
}

// none stands for no default currency in the select, which needs a value.
const none = "none"

function SettingsForm({ settings, currencies }: { settings: UserSettings; currencies: Currency[] }) {
  const queryClient = useQueryClient()
  const [currency, setCurrency] = useState(settings.defaultCurrencyId ? String(settings.defaultCurrencyId) : none)
  const update = useMutation({
    ...updateUserSettingsMutation(),
    onSuccess: (saved) => queryClient.setQueryData(retrieveUserSettingsQueryKey(), saved),
  })
  const body: UserSettings = {
    ...(currency === none ? {} : { defaultCurrencyId: Number(currency) }),
    gapPolicy: settings.gapPolicy,
  }
  const changed = (body.defaultCurrencyId ?? null) !== (settings.defaultCurrencyId ?? null)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Net worth</CardTitle>
        <CardDescription>How the net worth dashboard values what you hold.</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            update.mutate({ body })
          }}
        >
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="default-currency">Default currency</FieldLabel>
              <Select value={currency} onValueChange={setCurrency}>
                <SelectTrigger id="default-currency" className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={none}>None, ask each time</SelectItem>
                  {[...currencies]
                    .sort((a, b) => a.code.localeCompare(b.code))
                    .map((currency) => (
                      <SelectItem key={currency.id} value={String(currency.id)}>
                        {currency.code} · {currency.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              <FieldDescription>Used when the dashboard's link names no currency.</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="gap-policy">Holdings without a value</FieldLabel>
              <Select value={settings.gapPolicy} disabled>
                <SelectTrigger id="gap-policy" className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="exclude">Leave out and flag</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                A date leaves out a holding nobody priced yet, and is shaded. Counting it at cost is not built yet.
              </FieldDescription>
            </Field>
            <ProblemMessages error={update.error} />
            <div>
              <Button type="submit" disabled={!changed || update.isPending}>
                {update.isPending ? "Saving…" : "Save"}
              </Button>
            </div>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  )
}
