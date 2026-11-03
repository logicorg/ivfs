package mcptypes

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"
)

// typeSchemas overrides the schema jsonschema-go would infer for types whose
// JSON form differs from their Go struct. A decimal.Decimal is inferred as an
// empty object from its unexported fields, yet it is marshalled as a string
// such as "10.50", so the SDK would reject every amount sent to a tool and
// every amount a tool returns.
var typeSchemas = map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[decimal.Decimal](): {
		Type:        "string",
		Pattern:     `^-?[0-9]+(\.[0-9]+)?$`,
		Description: `a decimal number written as a string, such as "10.50"`,
	},
}

// AddTool registers a tool on server like mcp.AddTool, but infers its input
// and output schemas with typeSchemas applied. Every tool must be registered
// through it rather than through mcp.AddTool directly.
func AddTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if tool.InputSchema == nil {
		tool.InputSchema = schemaFor[In]()
	}
	if tool.OutputSchema == nil && reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		tool.OutputSchema = schemaFor[Out]()
	}
	mcp.AddTool(server, tool, handler)
}

// schemaFor infers T's schema, dereferencing a pointer first as mcp.AddTool
// does, since a handler returns its output by pointer but the output itself
// is an object.
func schemaFor[T any]() *jsonschema.Schema {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	schema, err := jsonschema.ForType(t, &jsonschema.ForOptions{TypeSchemas: typeSchemas})
	if err != nil {
		panic(fmt.Sprintf("infer schema for %s: %v", t, err))
	}
	return schema
}
