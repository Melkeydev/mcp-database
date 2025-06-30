package main

import (
	"flag"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
	"github.com/melkeydev/mcp-database/config"
	"github.com/melkeydev/mcp-database/databases"
	"github.com/melkeydev/mcp-database/mcp"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("config error", "error", err)
	}

	connStr, err := cfg.Database.GetConnectionString()
	if err != nil {
		slog.Error("connection string error", "error", err)
	}

	connector, err := databases.NewConnector(cfg.Database.DBType, connStr)
	if err != nil {
		slog.Error("failed to create connector", "error", err)
		return
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"mcp-database",
		"0.0.1", // TODO: move this to constant
		server.WithToolCapabilities(false),
		server.WithLogging(),
	)

	mcp.RegisterTools(s, connector)
	slog.Info("Info", "connected!", true)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}
