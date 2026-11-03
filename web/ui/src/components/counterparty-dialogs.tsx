import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useForm } from "react-hook-form"

import { createCounterpartyMutation } from "@/api/@tanstack/react-query.gen"
import { zNewCounterparty } from "@/api/zod.gen"
import { ProblemMessages, TextField } from "@/components/form-fields"
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

// CreateCounterpartyDialog adds a counterparty. Adding one whose name exists,
// in any casing, finds that counterparty instead, as every create does
// (ADR 0005).
export function CreateCounterpartyDialog({ open, onOpenChange }: DialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a counterparty</DialogTitle>
          <DialogDescription>Someone money is paid to or received from, such as a shop or an employer.</DialogDescription>
        </DialogHeader>
        {open && <CreateCounterpartyForm onDone={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  )
}

function CreateCounterpartyForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient()
  const form = useForm({ resolver: zodResolver(zNewCounterparty) })
  const create = useMutation({
    ...createCounterpartyMutation(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [{ _id: "listCounterparties" }] })
      onDone()
    },
  })

  return (
    <form onSubmit={form.handleSubmit((body) => create.mutate({ body }))}>
      <FieldGroup>
        <TextField label="Name" placeholder="Tesco" error={form.formState.errors.name} {...form.register("name")} />
        <TextField
          label="External id"
          description="How a bank export or payslip identifies it, if you know. It cannot be changed later."
          error={form.formState.errors.externalId}
          {...form.register("externalId", { setValueAs: (value: string) => (value.trim() === "" ? undefined : value) })}
        />
        <ProblemMessages error={create.error} />
      </FieldGroup>
      <DialogFooter className="mt-6">
        <Button type="submit" disabled={create.isPending}>
          Add counterparty
        </Button>
      </DialogFooter>
    </form>
  )
}
