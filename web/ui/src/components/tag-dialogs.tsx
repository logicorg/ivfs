import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useForm } from "react-hook-form"

import type { Tag } from "@/api"
import {
  createTagMutation,
  deleteTagMutation,
  listTagsQueryKey,
  updateTagMutation,
} from "@/api/@tanstack/react-query.gen"
import { zNewTag, zTagChanges } from "@/api/zod.gen"
import { ProblemMessages, TextField } from "@/components/form-fields"
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

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

// CreateTagDialog adds a tag. Adding one whose name exists, in any casing,
// finds that tag and leaves it as it is, as every create does (ADR 0005).
export function CreateTagDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a tag</DialogTitle>
          <DialogDescription>
            A label for context that cuts across categories. If the name exists, in any casing, that tag stays as it is.
          </DialogDescription>
        </DialogHeader>
        {open && <CreateTagForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function CreateTagForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient()
  const form = useForm({ resolver: zodResolver(zNewTag) })
  const create = useMutation({
    ...createTagMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listTagsQueryKey() })
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => create.mutate({ body }))}>
      <FieldGroup>
        <TextField label="Name" placeholder="vacation" error={form.formState.errors.name} {...form.register("name")} />
        <TextField label="Description" error={form.formState.errors.description} {...form.register("description")} />
        <ProblemMessages error={create.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={create.isPending}>
          Add tag
        </Button>
      </DialogFooter>
    </form>
  )
}

// EditTagDialog changes a tag's name or description.
export function EditTagDialog({ tag, onOpenChange }: { tag: Tag | null } & Omit<DialogProps, "open">) {
  return (
    <Dialog open={tag !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit {tag?.name}</DialogTitle>
          <DialogDescription>No other tag can have the same name, in any casing.</DialogDescription>
        </DialogHeader>
        {tag && <EditTagForm tag={tag} onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function EditTagForm({ tag, onDone }: { tag: Tag; onDone: () => void }) {
  const queryClient = useQueryClient()
  const form = useForm({
    resolver: zodResolver(zTagChanges),
    defaultValues: { name: tag.name, description: tag.description },
  })
  const update = useMutation({
    ...updateTagMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listTagsQueryKey() })
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => update.mutate({ path: { id: tag.id }, body }))}>
      <FieldGroup>
        <TextField label="Name" error={form.formState.errors.name} {...form.register("name")} />
        <TextField label="Description" error={form.formState.errors.description} {...form.register("description")} />
        <ProblemMessages error={update.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={update.isPending}>
          Save
        </Button>
      </DialogFooter>
    </form>
  )
}

// DeleteTagDialog deletes a tag once confirmed.
export function DeleteTagDialog({ tag, onOpenChange }: { tag: Tag | null } & Omit<DialogProps, "open">) {
  const queryClient = useQueryClient()
  const remove = useMutation({
    ...deleteTagMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listTagsQueryKey() })
      onOpenChange(false)
    },
  })

  return (
    <AlertDialog
      open={tag !== null}
      onOpenChange={(open) => {
        if (!open) {
          remove.reset()
        }
        onOpenChange(open)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {tag?.name}?</AlertDialogTitle>
          <AlertDialogDescription>This cannot be undone.</AlertDialogDescription>
        </AlertDialogHeader>
        <ProblemMessages error={remove.error} />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={remove.isPending}
            onClick={(event) => {
              // Keep the dialog open until the gateway answers, to show why
              // it refused.
              event.preventDefault()
              if (tag) {
                remove.mutate({ path: { id: tag.id } })
              }
            }}
          >
            Delete tag
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
