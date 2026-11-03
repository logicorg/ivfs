import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { MoreHorizontal, Plus } from "lucide-react"

import type { Category } from "@/api"
import { listCategoriesOptions, listCurrenciesOptions } from "@/api/@tanstack/react-query.gen"
import {
  CloseCategoryDialog,
  CreateCategoryDialog,
  DeleteCategoryDialog,
  MoveCategoryDialog,
  RenameCategoryDialog,
} from "@/components/category-dialogs"
import { ProblemMessages } from "@/components/form-fields"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Label } from "@/components/ui/label"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { categoryRows } from "@/lib/category-tree"

// CategoriesSearch is the screen's filter, kept in the URL.
type CategoriesSearch = { includeClosed?: boolean }

export const Route = createFileRoute("/categories")({
  validateSearch: (search: Record<string, unknown>): CategoriesSearch =>
    search.includeClosed === true || search.includeClosed === "true" ? { includeClosed: true } : {},
  component: Categories,
})

// An Action is what a category's menu opened, on which category.
type Action = { kind: "rename" | "move" | "close" | "delete"; category: Category }

function Categories() {
  const { includeClosed } = Route.useSearch()
  const navigate = Route.useNavigate()
  const categories = useQuery(listCategoriesOptions({ query: { includeClosed } }))
  const currencies = useQuery(listCurrenciesOptions())
  const [creating, setCreating] = useState(false)
  const [action, setAction] = useState<Action | null>(null)

  const all = categories.data?.categories ?? []
  const dialogCategory = (kind: Action["kind"]) => (action?.kind === kind ? action.category : null)
  const closeDialog = (open: boolean) => !open && setAction(null)

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Categories</h1>
        <Button onClick={() => setCreating(true)} disabled={!currencies.data?.currencies.length}>
          <Plus />
          Add category
        </Button>
      </div>

      <div className="flex items-center gap-2">
        <Checkbox
          id="include-closed"
          checked={includeClosed ?? false}
          onCheckedChange={(checked) => navigate({ search: checked === true ? { includeClosed: true } : {} })}
        />
        <Label htmlFor="include-closed">Show closed categories</Label>
      </div>

      {categories.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : categories.isError ? (
        <ProblemMessages error={categories.error} />
      ) : (
        <>
          <CategorySection
            title="Expenses"
            categories={all.filter((category) => category.kind === "expense")}
            onAction={setAction}
          />
          <CategorySection
            title="Income"
            categories={all.filter((category) => category.kind === "income")}
            onAction={setAction}
          />
        </>
      )}

      <CreateCategoryDialog
        open={creating}
        onOpenChange={setCreating}
        categories={all}
        currencies={currencies.data?.currencies ?? []}
      />
      <RenameCategoryDialog category={dialogCategory("rename")} onOpenChange={closeDialog} />
      <MoveCategoryDialog category={dialogCategory("move")} categories={all} onOpenChange={closeDialog} />
      <CloseCategoryDialog category={dialogCategory("close")} onOpenChange={closeDialog} />
      <DeleteCategoryDialog category={dialogCategory("delete")} onOpenChange={closeDialog} />
    </div>
  )
}

function CategorySection({
  title,
  categories,
  onAction,
}: {
  title: string
  categories: Category[]
  onAction: (action: Action) => void
}) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-lg font-medium">{title}</h2>
      {categories.length === 0 ? (
        <p className="text-muted-foreground">None yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead className="w-24">Currency</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {categoryRows(categories).map(({ category, depth }) => (
              <TableRow key={category.id} className={category.closed ? "text-muted-foreground" : undefined}>
                <TableCell>
                  <span className="flex items-center gap-2" style={{ paddingLeft: `${depth * 1.5}rem` }}>
                    {category.name}
                    {category.closed && <Badge variant="outline">closed</Badge>}
                  </span>
                </TableCell>
                <TableCell>{category.currencyCode}</TableCell>
                <TableCell>
                  <CategoryMenu category={category} onAction={onAction} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  )
}

function CategoryMenu({ category, onAction }: { category: Category; onAction: (action: Action) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={`Actions for ${category.name}`}>
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {!category.closed && (
          <>
            <DropdownMenuItem onSelect={() => onAction({ kind: "rename", category })}>Rename</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => onAction({ kind: "move", category })}>Move</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onSelect={() => onAction({ kind: "close", category })}>
              Close
            </DropdownMenuItem>
          </>
        )}
        <DropdownMenuItem variant="destructive" onSelect={() => onAction({ kind: "delete", category })}>
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
