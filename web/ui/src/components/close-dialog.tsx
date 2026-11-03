import { useMutation, useQueryClient, type UseMutationOptions } from "@tanstack/react-query"

import { ProblemMessages } from "@/components/form-fields"
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

type CloseDialogProps<TData, TError, TVariables> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  // title names what closes, such as "Close Revolut?".
  title: string
  // description says what closing it means and what it needs first.
  description: string
  // mutation closes it, given variables.
  mutation: UseMutationOptions<TData, TError, TVariables>
  variables: TVariables
  // onClosed runs once it is closed, such as to leave a page it no longer has.
  onClosed?: () => void
}

// CloseDialog closes an institution or an account, once confirmed. It stays
// open until the gateway answers, to show why it refused, and refreshes the
// accounts and balances once it is closed.
export function CloseDialog<TData, TError, TVariables>({
  open,
  onOpenChange,
  title,
  description,
  mutation,
  variables,
  onClosed,
}: CloseDialogProps<TData, TError, TVariables>) {
  const queryClient = useQueryClient()
  const close = useMutation({
    ...mutation,
    onSuccess: () => {
      for (const _id of ["listFinancialAccounts", "getAccountBalances", "getLedgerAccountBalance"]) {
        void queryClient.invalidateQueries({ queryKey: [{ _id }] })
      }
      onOpenChange(false)
      onClosed?.()
    },
  })

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          close.reset()
        }
        onOpenChange(next)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <ProblemMessages error={close.error} />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={close.isPending}
            onClick={(event) => {
              event.preventDefault()
              close.mutate(variables)
            }}
          >
            Close
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
