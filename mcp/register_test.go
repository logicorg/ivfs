package main

import (
	"encoding/json"
	"testing"

	"github.com/logicorg/ivfs/mcp/tools/accounts"
	"github.com/logicorg/ivfs/mcp/tools/assets"
	"github.com/logicorg/ivfs/mcp/tools/categories"
	"github.com/logicorg/ivfs/mcp/tools/currencies"
	"github.com/logicorg/ivfs/mcp/tools/fx"
	"github.com/logicorg/ivfs/mcp/tools/institution"
	"github.com/logicorg/ivfs/mcp/tools/journal"
	"github.com/logicorg/ivfs/mcp/tools/portfolio"
	"github.com/logicorg/ivfs/mcp/tools/reporting"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// newTestServer registers every domain's tools, without a Temporal client.
// mcp.AddTool infers a JSON schema for each tool's input and output type by
// reflection, and panics on a cyclic Go type rather than failing gracefully -
// most pkg/model types have relation fields that form one, so a new tool
// built directly on a model type makes this panic immediately, instead of
// only at runtime when the server actually starts.
func newTestServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	accounts.Register(server, nil)
	categories.Register(server, nil)
	currencies.Register(server, nil)
	assets.Register(server, nil)
	fx.Register(server, nil)
	institution.Register(server, nil)
	journal.Register(server, nil)
	portfolio.Register(server, nil)
	reporting.Register(server, nil)
	return server
}

func TestRegisterAllTools(t *testing.T) {
	newTestServer()
}

// TestToolSchemasHaveNoOpaqueObjects lists every tool through a client
// session and fails on any schema node that is an object with no properties
// that admits no others. That is what jsonschema-go infers for a struct with
// only unexported fields, such as decimal.Decimal, whose JSON form is not an
// object at all, so the SDK would reject every value of it. Such a type needs
// an entry in mcptypes' typeSchemas.
func TestToolSchemasHaveNoOpaqueObjects(t *testing.T) {
	ctx := t.Context()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := newTestServer().Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, tools.Tools)

	for _, tool := range tools.Tools {
		for kind, schema := range map[string]any{"input": tool.InputSchema, "output": tool.OutputSchema} {
			if schema == nil {
				continue
			}

			raw, err := json.Marshal(schema)
			require.NoError(t, err)

			var node any
			require.NoError(t, json.Unmarshal(raw, &node))
			for _, path := range opaqueObjects(node, "") {
				t.Errorf("tool %s: %s schema has an opaque object at %q", tool.Name, kind, path)
			}
		}
	}
}

// opaqueObjects returns the path of every object schema in node that has no
// properties and forbids additional ones. The root is exempt: a tool without
// parameters, such as list_currencies, rightly takes an empty object.
func opaqueObjects(node any, path string) []string {
	var found []string
	switch n := node.(type) {
	case map[string]any:
		props, hasProps := n["properties"].(map[string]any)
		if path != "" && n["type"] == "object" && (!hasProps || len(props) == 0) && n["additionalProperties"] == false {
			found = append(found, path)
		}
		for key, child := range n {
			found = append(found, opaqueObjects(child, path+"/"+key)...)
		}
	case []any:
		for _, child := range n {
			found = append(found, opaqueObjects(child, path+"/[]")...)
		}
	}
	return found
}
