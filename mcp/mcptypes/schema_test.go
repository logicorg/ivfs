package mcptypes

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type decimalInput struct {
	Amount decimal.Decimal `json:"amount"`
}

type decimalOutput struct {
	Amount decimal.Decimal `json:"amount"`
}

// TestAddTool_Decimal calls a tool that echoes a decimal through a real
// client session, so the SDK validates both the arguments and the result
// against the schemas AddTool inferred.
func TestAddTool_Decimal(t *testing.T) {
	ctx := t.Context()

	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	AddTool(server, &mcp.Tool{Name: "echo"}, func(_ context.Context, _ *mcp.CallToolRequest, in decimalInput) (*mcp.CallToolResult, *decimalOutput, error) {
		return nil, &decimalOutput{Amount: in.Amount}, nil
	})
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	tests := []struct {
		name      string
		amount    any
		want      any
		wantError string
	}{
		{
			name:   "decimal string",
			amount: "10.50",
			want:   map[string]any{"amount": "10.5"},
		},
		{
			name:   "negative decimal string",
			amount: "-3",
			want:   map[string]any{"amount": "-3"},
		},
		{
			name: "number", amount: 10.5,
			wantError: `has type "number", want "string"`,
		},
		{
			name:      "not a number",
			amount:    "ten",
			wantError: "does not match regular expression",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"amount": tt.amount}})
			require.NoError(t, err)
			if tt.wantError != "" {
				require.True(t, res.IsError)
				require.Len(t, res.Content, 1)
				require.Contains(t, res.Content[0].(*mcp.TextContent).Text, tt.wantError)
				return
			}
			require.False(t, res.IsError)
			require.Equal(t, tt.want, res.StructuredContent)
		})
	}
}
