import { useState } from "react"
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryKey,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query"

import type { AccountDeletion, AccountDeletionPreview } from "@/api"
import { ProblemMessages } from "@/components/form-fields"
import { NegativeHoldingWarnings } from "@/components/negative-holding-warnings"
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
import { describeDeletionPreview } from "@/lib/deletion-preview"

type DeleteDialogProps<TError, TVariables, TKey extends QueryKey> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  // title names what is deleted, such as "Delete Revolut?".
  title: string
  // preview reads what deleting it would do.
  preview: UseQueryOptions<AccountDeletionPreview, TError, AccountDeletionPreview, TKey>
  // mutation deletes it, given variables.
  mutation: UseMutationOptions<AccountDeletion, TError, TVariables>
  variables: TVariables
  // onDeleted runs once it is deleted, such as to leave a page it no longer has.
  onDeleted?: () => void
}

// DeleteDialog deletes an account, a category or an institution once
// confirmed, after showing what the deletion would do (ADR 0012). It refuses
// to start while the preview lists a conflict. A deletion that outlasts the
// request goes on by itself, and the dialog says so instead of closing.
export function DeleteDialog<TError, TVariables, TKey extends QueryKey>({
  open,
  onOpenChange,
  title,
  preview: previewOptions,
  mutation,
  variables,
  onDeleted,
}: DeleteDialogProps<TError, TVariables, TKey>) {
  const queryClient = useQueryClient()
  // done is what the deletion answered when it is worth reading before the
  // dialog closes: it is still running, or it left a holding negative.
  const [done, setDone] = useState<AccountDeletion>()
  const preview = useQuery({ ...previewOptions, enabled: open && done === undefined })
  const remove = useMutation({
    ...mutation,
    onSuccess: (deletion) => {
      // Entries, balances and totals anywhere may have changed.
      void queryClient.invalidateQueries()
      if (deletion.workflowId || deletion.warnings?.length) {
        setDone(deletion)
        return
      }
      onOpenChange(false)
      onDeleted?.()
    },
  })

  const conflicts = preview.data?.conflicts ?? []
  const finish = () => {
    const ran = done !== undefined && !done.workflowId
    onOpenChange(false)
    if (ran) {
      onDeleted?.()
    }
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          remove.reset()
          setDone(undefined)
        }
        onOpenChange(next)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>
            {done?.workflowId
              ? "It is still being deleted, and finishes by itself. Deleting it again waits for it."
              : done
                ? "Deleted."
                : "This cannot be undone. Totals and holdings that stay do not change."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {done ? (
          <NegativeHoldingWarnings title="This leaves a holding negative" warnings={done.warnings ?? []} />
        ) : preview.isPending ? (
          <p className="text-sm text-muted-foreground">Reading what it would delete…</p>
        ) : preview.isError ? (
          <ProblemMessages error={preview.error} />
        ) : (
          <div className="flex flex-col gap-2 text-sm">
            <ul className="list-disc pl-5">
              {describeDeletionPreview(preview.data).map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
            {conflicts.length > 0 && (
              <div role="alert">
                <p className="font-medium">It cannot be deleted yet:</p>
                <ul className="list-disc pl-5 text-destructive">
                  {conflicts.map((conflict) => (
                    <li key={conflict}>{conflict}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
        <ProblemMessages error={remove.error} />
        <AlertDialogFooter>
          {done ? (
            <AlertDialogAction onClick={finish}>Done</AlertDialogAction>
          ) : (
            <>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                disabled={!preview.isSuccess || conflicts.length > 0 || remove.isPending}
                onClick={(event) => {
                  event.preventDefault()
                  remove.mutate(variables)
                }}
              >
                Delete
              </AlertDialogAction>
            </>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
