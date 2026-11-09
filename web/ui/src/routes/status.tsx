import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

import { getHealthOptions } from "@/api/@tanstack/react-query.gen"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export const Route = createFileRoute("/status")({
  component: Status,
})

function Status() {
  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">System status</h1>
      <TemporalStatus />
    </div>
  )
}

// TemporalStatus shows whether the gateway reaches Temporal, which every
// screen needs to load anything.
function TemporalStatus() {
  const health = useQuery(getHealthOptions())

  return (
    <Card>
      <CardHeader>
        <CardTitle>Temporal</CardTitle>
        <CardDescription>The gateway runs every request as a workflow on Temporal.</CardDescription>
      </CardHeader>
      <CardContent>
        {health.isPending ? (
          <Badge variant="outline">checking…</Badge>
        ) : health.isSuccess ? (
          <Badge variant="secondary">reachable</Badge>
        ) : (
          <Badge variant="destructive">{health.error.detail ?? "the gateway is unreachable"}</Badge>
        )}
      </CardContent>
    </Card>
  )
}
