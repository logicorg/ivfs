package main

import (
	"context"
	"log"
	"os"

	"github.com/logicorg/ivfs/internal/pkg/temporalenv"
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
	"go.temporal.io/sdk/client"
)

func main() {
	ctx := context.Background()

	options, err := temporalenv.Options(os.Getenv)
	if err != nil {
		log.Fatalln(err)
	}
	c, err := client.Dial(options)
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	server := mcp.NewServer(&mcp.Implementation{
		Name: "Financer", Title: "Personal Finance",
	}, nil)
	accounts.Register(server, c)
	categories.Register(server, c)
	currencies.Register(server, c)
	assets.Register(server, c)
	fx.Register(server, c)
	institution.Register(server, c)
	journal.Register(server, c)
	portfolio.Register(server, c)
	reporting.Register(server, c)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
