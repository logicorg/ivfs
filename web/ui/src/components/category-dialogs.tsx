import { zodResolver } from "@hookform/resolvers/zod"
import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Controller, useForm } from "react-hook-form"

import type { Category } from "@/api"
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
import { describeCategoryDeletion, parentChoices, replacementChoices, type CategoryRow } from "@/lib/category-tree"

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
}

// CreateCategoryDialog adds a category. Adding one whose name exists under
// the same parent finds that category instead (ADR 0005).
export function CreateCategoryDialog({ open, onOpenChange, categories }: CreateCategoryDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a category</DialogTitle>
          <DialogDescription>Its kind cannot change. It takes entries in any currency.</DialogDescription>
        </DialogHeader>
        <CreateCategoryForm categories={categories} onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}

function CreateCategoryForm({ categories, onDone }: { categories: Category[]; onDone: () => void }) {
  const form = useForm({
    resolver: zodResolver(zNewCategory),
    defaultValues: { kind: "expense" },
  })
  const create = useMutation({ ...createCategoryMutation(), onSuccess: useCategoriesChanged(onDone) })

  const parents = parentChoices(categories, form.watch("kind"))

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
          name="parentId"
          render={({ field, fieldState }) => (
            <SelectField
              id="parentId"
              label="Under"
              value={parentValue(field.value)}
              onChange={(value) => field.onChange(parentFromValue(value))}
              options={parentOptions(parents)}
              description="Only open categories of the same kind can hold it."
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
          <DialogDescription>A category can move only while it and the ones under it have no entries.</DialogDescription>
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
  const parents = parentChoices(categories, category.kind, category)

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
            It keeps its entries, and is no longer offered for new ones.
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

// noReplacement is the choice for merging into no category: Radix selects
// cannot hold an empty value.
const noReplacement = "none"

// DeleteCategoryDialog deletes a category, or merges it into another, after
// showing how many entries that touches. No entry is deleted, so there is
// nothing else to preview.
export function DeleteCategoryDialog({
  category,
  categories,
  onOpenChange,
}: CategoryDialogProps & { categories: Category[] }) {
  return (
    <AlertDialog open={category !== null} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        {/* Keyed so a replacement chosen for one category is not kept for the next. */}
        {category && (
          <DeleteCategoryContent
            key={category.id}
            category={category}
            categories={categories}
            onDone={() => onOpenChange(false)}
          />
        )}
      </AlertDialogContent>
    </AlertDialog>
  )
}

function DeleteCategoryContent({
  category,
  categories,
  onDone,
}: {
  category: Category
  categories: Category[]
  onDone: () => void
}) {
  const queryClient = useQueryClient()
  const [replacementId, setReplacementId] = useState<number>()
  const path = { id: category.id }
  const preview = useQuery(previewCategoryDeletionOptions({ path }))
  const remove = useMutation({
    ...deleteCategoryMutation(),
    onSuccess: () => {
      // Entries and totals show the category, so they changed too.
      void queryClient.invalidateQueries()
      onDone()
    },
  })
  const replacements = replacementChoices(categories, category)
  const replacement = replacements.find((choice) => choice.id === replacementId)

  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete {category.name}?</AlertDialogTitle>
        <AlertDialogDescription>This cannot be undone. No journal entry is deleted, and no balance changes.</AlertDialogDescription>
      </AlertDialogHeader>
      {preview.isPending ? (
        <p className="text-sm text-muted-foreground">Reading what it would change…</p>
      ) : preview.isError ? (
        <ProblemMessages error={preview.error} />
      ) : (
        <FieldGroup>
          <p className="text-sm">{describeCategoryDeletion(preview.data.entries, replacement?.label)}</p>
          {preview.data.entries > 0 && replacements.length > 0 && (
            <SelectField
              id="replacementId"
              label="Move its entries to"
              value={replacementId === undefined ? noReplacement : String(replacementId)}
              onChange={(value) => setReplacementId(value === noReplacement ? undefined : Number(value))}
              options={[
                { value: noReplacement, label: "No category" },
                ...replacements.map((choice) => ({ value: String(choice.id), label: choice.label })),
              ]}
            />
          )}
        </FieldGroup>
      )}
      <ProblemMessages error={remove.error} />
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction
          variant="destructive"
          disabled={!preview.isSuccess || remove.isPending}
          onClick={(event) => {
            // Keep the dialog open until the gateway answers, to show why it
            // refused.
            event.preventDefault()
            remove.mutate({ path, query: { replacementId } })
          }}
        >
          Delete
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  )
}
