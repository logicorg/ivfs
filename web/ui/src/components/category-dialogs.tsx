import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm } from "react-hook-form"

import type { Category, Currency } from "@/api"
import {
  closeCategoryMutation,
  createCategoryMutation,
  deleteCategoryMutation,
  listCategoriesQueryKey,
  moveCategoryMutation,
  previewCategoryDeletionOptions,
  renameCategoryMutation,
} from "@/api/@tanstack/react-query.gen"
import { zCategoryName, zCategoryParent, zNewCategory } from "@/api/zod.gen"
import { DeleteDialog } from "@/components/delete-dialog"
import { ProblemMessages, SelectField, TextField, type SelectOption } from "@/components/form-fields"
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
import { parentChoices, type CategoryRow } from "@/lib/category-tree"

// topLevel is the parent choice for no parent: Radix selects cannot hold an
// empty value.
const topLevel = "top-level"

const kindOptions: SelectOption[] = [
  { value: "expense", label: "Expense" },
  { value: "income", label: "Income" },
]

function parentOptions(rows: CategoryRow[]): SelectOption[] {
  return [
    { value: topLevel, label: "None (top level)" },
    ...rows.map(({ category, depth }) => ({ value: String(category.id), label: category.name, depth })),
  ]
}

const parentValue = (parentId: number | undefined) => (parentId === undefined ? topLevel : String(parentId))
const parentFromValue = (value: string) => (value === topLevel ? undefined : Number(value))

// useCategoriesChanged returns what a mutation does once it succeeds: list
// the categories again, and close its dialog.
function useCategoriesChanged(onDone: () => void) {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: listCategoriesQueryKey() })
    onDone()
  }
}

type CreateCategoryDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  categories: Category[]
  currencies: Currency[]
}

// CreateCategoryDialog adds a category. Adding one whose name exists under
// the same parent finds that category instead (ADR 0005).
export function CreateCategoryDialog({ open, onOpenChange, categories, currencies }: CreateCategoryDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a category</DialogTitle>
          <DialogDescription>Its kind and currency cannot change once it is used.</DialogDescription>
        </DialogHeader>
        <CreateCategoryForm categories={categories} currencies={currencies} onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}

function CreateCategoryForm({
  categories,
  currencies,
  onDone,
}: Omit<CreateCategoryDialogProps, "open" | "onOpenChange"> & { onDone: () => void }) {
  const form = useForm({
    resolver: zodResolver(zNewCategory),
    defaultValues: { kind: "expense", currencyId: currencies.length === 1 ? currencies[0]?.id : undefined },
  })
  const create = useMutation({ ...createCategoryMutation(), onSuccess: useCategoriesChanged(onDone) })

  const [kind, currencyId] = form.watch(["kind", "currencyId"])
  const parents = currencyId === undefined ? [] : parentChoices(categories, kind, currencyId)
  const currencyOptions = [...currencies]
    .sort((a, b) => a.code.localeCompare(b.code))
    .map((currency) => ({ value: String(currency.id), label: `${currency.code} · ${currency.name}` }))

  return (
    <form onSubmit={form.handleSubmit((body) => create.mutate({ body }))}>
      <FieldGroup>
        <TextField label="Name" placeholder="Groceries" error={form.formState.errors.name} {...form.register("name")} />
        <Controller
          control={form.control}
          name="kind"
          render={({ field, fieldState }) => (
            <SelectField
              id="kind"
              label="Kind"
              value={field.value}
              onChange={(value) => {
                field.onChange(value)
                form.setValue("parentId", undefined)
              }}
              options={kindOptions}
              error={fieldState.error}
            />
          )}
        />
        <Controller
          control={form.control}
          name="currencyId"
          render={({ field, fieldState }) => (
            <SelectField
              id="currencyId"
              label="Currency"
              value={field.value === undefined ? undefined : String(field.value)}
              onChange={(value) => {
                field.onChange(Number(value))
                form.setValue("parentId", undefined)
              }}
              options={currencyOptions}
              placeholder="Choose a currency"
              error={fieldState.error}
            />
          )}
        />
        <Controller
          control={form.control}
          name="parentId"
          render={({ field, fieldState }) => (
            <SelectField
              id="parentId"
              label="Under"
              value={parentValue(field.value)}
              onChange={(value) => field.onChange(parentFromValue(value))}
              options={parentOptions(parents)}
              description="Only open categories of the same kind and currency can hold it."
              error={fieldState.error}
            />
          )}
        />
        <ProblemMessages error={create.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={create.isPending}>
          Add category
        </Button>
      </DialogFooter>
    </form>
  )
}

type CategoryDialogProps = {
  category: Category | null
  onOpenChange: (open: boolean) => void
}

// RenameCategoryDialog renames a category. Its postings follow, since they
// refer to it by id.
export function RenameCategoryDialog({ category, onOpenChange }: CategoryDialogProps) {
  return (
    <Dialog open={category !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename {category?.name}</DialogTitle>
          <DialogDescription>Names are unique among the categories under the same parent.</DialogDescription>
        </DialogHeader>
        {category && <RenameCategoryForm category={category} onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function RenameCategoryForm({ category, onDone }: { category: Category; onDone: () => void }) {
  const form = useForm({ resolver: zodResolver(zCategoryName), defaultValues: { name: category.name } })
  const rename = useMutation({ ...renameCategoryMutation(), onSuccess: useCategoriesChanged(onDone) })

  return (
    <form onSubmit={form.handleSubmit((body) => rename.mutate({ path: { id: category.id }, body }))}>
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

// MoveCategoryDialog moves a category under another, or to the top level.
export function MoveCategoryDialog({
  category,
  categories,
  onOpenChange,
}: CategoryDialogProps & { categories: Category[] }) {
  return (
    <Dialog open={category !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Move {category?.name}</DialogTitle>
          <DialogDescription>A category can move only while it and the ones under it have no postings.</DialogDescription>
        </DialogHeader>
        {category && (
          <MoveCategoryForm category={category} categories={categories} onDone={() => onOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function MoveCategoryForm({
  category,
  categories,
  onDone,
}: {
  category: Category
  categories: Category[]
  onDone: () => void
}) {
  const form = useForm({ resolver: zodResolver(zCategoryParent), defaultValues: { parentId: category.parentId } })
  const move = useMutation({ ...moveCategoryMutation(), onSuccess: useCategoriesChanged(onDone) })
  const parents = parentChoices(categories, category.kind, category.currencyId, category)

  return (
    <form onSubmit={form.handleSubmit((body) => move.mutate({ path: { id: category.id }, body }))}>
      <FieldGroup>
        <Controller
          control={form.control}
          name="parentId"
          render={({ field, fieldState }) => (
            <SelectField
              id="parentId"
              label="Under"
              value={parentValue(field.value)}
              onChange={(value) => field.onChange(parentFromValue(value))}
              options={parentOptions(parents)}
              error={fieldState.error}
            />
          )}
        />
        <ProblemMessages error={move.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={move.isPending}>
          Move
        </Button>
      </DialogFooter>
    </form>
  )
}

// CloseCategoryDialog closes a category.
export function CloseCategoryDialog({ category, onOpenChange }: CategoryDialogProps) {
  const close = useMutation({ ...closeCategoryMutation(), onSuccess: useCategoriesChanged(() => onOpenChange(false)) })

  return (
    <AlertDialog
      open={category !== null}
      onOpenChange={(open) => {
        if (!open) {
          close.reset()
        }
        onOpenChange(open)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Close {category?.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            It keeps its history but takes no new postings.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <ProblemMessages error={close.error} />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={close.isPending}
            onClick={(event) => {
              // Keep the dialog open until the gateway answers, to show why
              // it refused.
              event.preventDefault()
              if (category) {
                close.mutate({ path: { id: category.id } })
              }
            }}
          >
            Close category
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

// DeleteCategoryDialog deletes a category, once its preview shows what that
// does to the entries booked to it.
export function DeleteCategoryDialog({ category, onOpenChange }: CategoryDialogProps) {
  const path = { id: category?.id ?? 0 }

  return (
    <DeleteDialog
      open={category !== null}
      onOpenChange={onOpenChange}
      title={`Delete ${category?.name ?? ""}?`}
      preview={previewCategoryDeletionOptions({ path })}
      mutation={deleteCategoryMutation()}
      variables={{ path }}
    />
  )
}
