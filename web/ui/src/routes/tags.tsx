import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { Pencil, Plus, Trash2 } from "lucide-react"

import type { Tag } from "@/api"
import { listTagsOptions } from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { CreateTagDialog, DeleteTagDialog, EditTagDialog } from "@/components/tag-dialogs"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export const Route = createFileRoute("/tags")({
  component: Tags,
})

function Tags() {
  const tags = useQuery(listTagsOptions())
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Tag | null>(null)
  const [deleting, setDeleting] = useState<Tag | null>(null)

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Tags</h1>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          Add tag
        </Button>
      </div>

      {tags.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : tags.isError ? (
        <ProblemMessages error={tags.error} />
      ) : tags.data.tags.length === 0 ? (
        <p className="text-muted-foreground">No tags yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Description</TableHead>
              <TableHead className="w-24" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {tags.data.tags.map((tag) => (
              <TableRow key={tag.id}>
                <TableCell className="font-medium">{tag.name}</TableCell>
                <TableCell className="text-muted-foreground">{tag.description}</TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="icon" aria-label={`Edit ${tag.name}`} onClick={() => setEditing(tag)}>
                    <Pencil />
                  </Button>
                  <Button variant="ghost" size="icon" aria-label={`Delete ${tag.name}`} onClick={() => setDeleting(tag)}>
                    <Trash2 />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CreateTagDialog open={creating} onOpenChange={setCreating} />
      <EditTagDialog tag={editing} onOpenChange={(open) => !open && setEditing(null)} />
      <DeleteTagDialog tag={deleting} onOpenChange={(open) => !open && setDeleting(null)} />
    </div>
  )
}
