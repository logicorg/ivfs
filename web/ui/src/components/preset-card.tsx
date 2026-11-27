import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import { useState } from "react"

import {
  applyPresetMutation,
  getDefaultPresetOptions,
  listCategoriesQueryKey,
  listCurrenciesQueryKey,
  listTagsQueryKey,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  countIncluded,
  newKey,
  presetForm,
  presetOf,
  updateCategory,
  type CategoryRow,
  type PresetForm,
} from "@/lib/preset"

// PresetCard offers the default preset to a user who has nothing yet. Nothing
// is written until they press Create, and the card goes away on its own once
// anything exists, so there is no dismissal to remember.
export function PresetCard() {
  const preset = useQuery(getDefaultPresetOptions())

  return (
    <Card>
      <CardHeader>
        <CardTitle>Start from a preset</CardTitle>
        <CardDescription>
          Currencies, categories, and tags to start recording with. Untick or rename anything, add your own, or
          ignore this and create your first currency by hand. Everything can be changed later.
        </CardDescription>
      </CardHeader>
      {preset.isPending ? (
        <CardContent className="text-muted-foreground">Loading…</CardContent>
      ) : preset.isError ? (
        <CardContent>
          <ProblemMessages error={preset.error} />
        </CardContent>
      ) : (
        <PresetFormFields initial={presetForm(preset.data)} />
      )}
    </Card>
  )
}

function PresetFormFields({ initial }: { initial: PresetForm }) {
  const [form, setForm] = useState(initial)
  const queryClient = useQueryClient()
  const apply = useMutation({
    ...applyPresetMutation(),
    onSuccess: () => {
      // A key without options matches the lists' queries whatever their
      // filters, and once they reload the card hides itself.
      for (const queryKey of [listCurrenciesQueryKey(), listCategoriesQueryKey(), listTagsQueryKey()]) {
        void queryClient.invalidateQueries({ queryKey })
      }
    },
  })
  const count = countIncluded(form)
  const setCategories = (change: (rows: CategoryRow[]) => CategoryRow[]) =>
    setForm((form) => ({ ...form, categories: change(form.categories) }))
  const changeCategory = (key: string, change: Partial<CategoryRow>) =>
    setCategories((rows) => updateCategory(rows, key, (row) => ({ ...row, ...change })))

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        apply.mutate({ body: presetOf(form) })
      }}
    >
      <CardContent className="flex flex-col gap-6">
        <Section
          title="Currencies"
          addLabel="Add currency"
          onAdd={() =>
            setForm((form) => ({
              ...form,
              currencies: [...form.currencies, { key: newKey(), included: true, code: "", name: "", precision: 2 }],
            }))
          }
        >
          {form.currencies.map((row) => {
            const change = (fields: Partial<typeof row>) =>
              setForm((form) => ({
                ...form,
                currencies: form.currencies.map((r) => (r.key === row.key ? { ...r, ...fields } : r)),
              }))
            return (
              <div key={row.key} className="flex items-center gap-2">
                <Checkbox
                  checked={row.included}
                  onCheckedChange={(checked) => change({ included: checked === true })}
                  aria-label={`Create ${row.code || "this currency"}`}
                />
                <Input
                  className="w-20"
                  value={row.code}
                  placeholder="EUR"
                  aria-label="Code"
                  disabled={!row.included}
                  onChange={(event) => change({ code: event.target.value.toUpperCase() })}
                />
                <Input
                  className="min-w-0 flex-1"
                  value={row.name}
                  placeholder="Euro"
                  aria-label="Name"
                  disabled={!row.included}
                  onChange={(event) => change({ name: event.target.value })}
                />
                <Input
                  className="w-16"
                  value={row.symbol ?? ""}
                  placeholder="€"
                  aria-label="Symbol"
                  disabled={!row.included}
                  onChange={(event) => change({ symbol: event.target.value })}
                />
                <Input
                  className="w-16"
                  type="number"
                  min={0}
                  value={row.precision ?? 2}
                  aria-label="Decimal places"
                  title="Decimal places"
                  disabled={!row.included}
                  onChange={(event) => change({ precision: event.target.valueAsNumber })}
                />
              </div>
            )
          })}
        </Section>

        <Section
          title="Categories"
          addLabel="Add expense category"
          onAdd={() => setCategories((rows) => [...rows, newCategory("expense")])}
          secondAddLabel="Add income category"
          onSecondAdd={() => setCategories((rows) => [...rows, newCategory("income")])}
        >
          {(["expense", "income"] as const).map((kind) => (
            <div key={kind} className="flex flex-col gap-2">
              <p className="text-sm text-muted-foreground">{kind === "expense" ? "Expenses" : "Income"}</p>
              {form.categories
                .filter((row) => row.kind === kind)
                .map((row) => (
                  <div key={row.key} className="flex flex-col gap-2">
                    <CategoryField
                      row={row}
                      onChange={(change) => changeCategory(row.key, change)}
                      onAddChild={() =>
                        changeCategory(row.key, { children: [...row.children, newCategory()], included: true })
                      }
                    />
                    {row.children.map((child) => (
                      <div key={child.key} className="ml-6">
                        <CategoryField
                          row={child}
                          disabled={!row.included}
                          onChange={(change) => changeCategory(child.key, change)}
                        />
                      </div>
                    ))}
                  </div>
                ))}
            </div>
          ))}
        </Section>

        <Section
          title="Tags"
          addLabel="Add tag"
          onAdd={() => setForm((form) => ({ ...form, tags: [...form.tags, { key: newKey(), included: true, name: "" }] }))}
        >
          {form.tags.map((row) => {
            const change = (fields: Partial<typeof row>) =>
              setForm((form) => ({ ...form, tags: form.tags.map((r) => (r.key === row.key ? { ...r, ...fields } : r)) }))
            return (
              <div key={row.key} className="flex items-center gap-2">
                <Checkbox
                  checked={row.included}
                  onCheckedChange={(checked) => change({ included: checked === true })}
                  aria-label={`Create ${row.name || "this tag"}`}
                />
                <Input
                  className="min-w-0 flex-1"
                  value={row.name}
                  placeholder="vacation"
                  aria-label="Name"
                  disabled={!row.included}
                  onChange={(event) => change({ name: event.target.value })}
                />
              </div>
            )
          })}
        </Section>

        <ProblemMessages error={apply.error} />
      </CardContent>
      <CardFooter className="mt-6 justify-end">
        <Button type="submit" disabled={apply.isPending || count === 0}>
          {apply.isPending ? "Creating…" : `Create ${count} ${count === 1 ? "item" : "items"}`}
        </Button>
      </CardFooter>
    </form>
  )
}

function newCategory(kind?: CategoryRow["kind"]): CategoryRow {
  return { key: newKey(), included: true, name: "", kind, children: [] }
}

function Section({
  title,
  addLabel,
  onAdd,
  secondAddLabel,
  onSecondAdd,
  children,
}: {
  title: string
  addLabel: string
  onAdd: () => void
  secondAddLabel?: string
  onSecondAdd?: () => void
  children: React.ReactNode
}) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 font-medium">{title}</legend>
      {children}
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="ghost" size="sm" onClick={onAdd}>
          <Plus /> {addLabel}
        </Button>
        {secondAddLabel && onSecondAdd && (
          <Button type="button" variant="ghost" size="sm" onClick={onSecondAdd}>
            <Plus /> {secondAddLabel}
          </Button>
        )}
      </div>
    </fieldset>
  )
}

// CategoryField is one category's box and name. A top-level one can be
// given a subcategory; the preset goes one level deep, which is as far as
// the form offers.
function CategoryField({
  row,
  disabled = false,
  onChange,
  onAddChild,
}: {
  row: CategoryRow
  disabled?: boolean
  onChange: (change: Partial<CategoryRow>) => void
  onAddChild?: () => void
}) {
  return (
    <div className="flex items-center gap-2">
      <Checkbox
        checked={row.included && !disabled}
        disabled={disabled}
        onCheckedChange={(checked) => onChange({ included: checked === true })}
        aria-label={`Create ${row.name || "this category"}`}
      />
      <Input
        className="min-w-0 flex-1"
        value={row.name}
        placeholder={onAddChild ? "Category" : "Subcategory"}
        aria-label="Name"
        disabled={disabled || !row.included}
        onChange={(event) => onChange({ name: event.target.value })}
      />
      {onAddChild && (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={onAddChild}
          disabled={!row.included}
          aria-label={`Add a subcategory to ${row.name || "this category"}`}
          title="Add subcategory"
        >
          <Plus />
        </Button>
      )}
    </div>
  )
}
