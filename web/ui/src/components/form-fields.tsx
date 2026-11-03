import type { FieldError as FormFieldError, UseFormRegisterReturn } from "react-hook-form"

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { problemMessages } from "@/lib/problem"

// TextFieldProps are an input's own props, which form.register's result
// supplies, and the field around it.
type TextFieldProps = React.ComponentProps<typeof Input> &
  UseFormRegisterReturn & {
    label: string
    description?: string
    error?: FormFieldError
  }

// TextField is a labelled input with its validation error.
export function TextField({ label, description, error, ...input }: TextFieldProps) {
  return (
    <Field data-invalid={error !== undefined}>
      <FieldLabel htmlFor={input.name}>{label}</FieldLabel>
      <Input id={input.name} aria-invalid={error !== undefined} {...input} />
      {description && <FieldDescription>{description}</FieldDescription>}
      <FieldError errors={[error]} />
    </Field>
  )
}

// A SelectOption is one choice of a SelectField. depth indents it, for a
// choice inside a tree such as a category.
export type SelectOption = { value: string; label: string; depth?: number }

type SelectFieldProps = {
  id: string
  label: string
  value: string | undefined
  onChange: (value: string) => void
  options: SelectOption[]
  placeholder?: string
  description?: string
  error?: FormFieldError
}

// SelectField is a labelled select with its validation error. Radix selects
// hold strings, so a numeric field converts in its Controller. No choice
// reaches Radix as "", never undefined: undefined turns the select
// uncontrolled, and it keeps showing a choice the form has cleared.
export function SelectField({
  id,
  label,
  value,
  onChange,
  options,
  placeholder,
  description,
  error,
}: SelectFieldProps) {
  return (
    <Field data-invalid={error !== undefined}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Select value={value ?? ""} onValueChange={onChange}>
        <SelectTrigger id={id} aria-invalid={error !== undefined} className="w-full">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              <span style={{ paddingLeft: `${(option.depth ?? 0) * 1}rem` }}>{option.label}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {description && <FieldDescription>{description}</FieldDescription>}
      <FieldError errors={[error]} />
    </Field>
  )
}

// ProblemMessages shows why the gateway refused a request, such as a name
// that is already taken.
export function ProblemMessages({ error }: { error: unknown }) {
  if (!error) {
    return null
  }
  return <FieldError errors={problemMessages(error).map((message) => ({ message }))} />
}
