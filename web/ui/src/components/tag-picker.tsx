import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"

import { listTagsOptions } from "@/api/@tanstack/react-query.gen"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { toggleTag } from "@/lib/tags"

type TagPickerProps = {
  id: string
  value: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
}

// TagPicker chooses any number of tags, the chosen ones shown on its trigger.
// The menu stays open while choosing, as several tags are usually picked at
// once. disabled freezes the choices but leaves the menu open, so a change
// being saved does not close it.
export function TagPicker({ id, value, onChange, disabled }: TagPickerProps) {
  const tags = useQuery(listTagsOptions())
  const all = tags.data?.tags ?? []
  const chosen = all.filter((tag) => value.includes(tag.id))

  if (tags.isSuccess && all.length === 0) {
    return (
      <span className="text-sm text-muted-foreground">
        No tags yet.{" "}
        <Link to="/tags" className="underline">
          Create one
        </Link>
      </span>
    )
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button id={id} type="button" variant="outline" disabled={!tags.isSuccess} className="h-auto min-h-9 w-full justify-start flex-wrap gap-1">
          {chosen.length === 0 ? (
            <span className="text-muted-foreground">None</span>
          ) : (
            chosen.map((tag) => (
              <Badge key={tag.id} variant="secondary">
                {tag.name}
              </Badge>
            ))
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="max-h-72 overflow-y-auto">
        {all.map((tag) => (
          <DropdownMenuCheckboxItem
            key={tag.id}
            checked={value.includes(tag.id)}
            disabled={disabled}
            onCheckedChange={() => onChange(toggleTag(value, tag.id))}
            onSelect={(event) => event.preventDefault()}
          >
            {tag.name}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

type TagsFieldProps = Omit<TagPickerProps, "id">

// TagsField is a record form's labelled TagPicker.
export function TagsField(props: TagsFieldProps) {
  return (
    <Field>
      <FieldLabel htmlFor="tagIds">Tags</FieldLabel>
      <TagPicker id="tagIds" {...props} />
      <FieldDescription>Optional.</FieldDescription>
    </Field>
  )
}
