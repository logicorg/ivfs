package health

// Health is the gateway's health. Each response body is a named type, which
// names its schema, and so its generated TypeScript type.
type Health struct {
	Status string `json:"status" enum:"ok" doc:"ok when the gateway can reach Temporal"`
}

// Output is the response of GET /api/health.
type Output struct {
	Body Health
}
