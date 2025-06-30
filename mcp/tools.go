package mcp

import (
	goMCP "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/melkeydev/mcp-database/databases"
	"github.com/melkeydev/mcp-database/handlers"
)

func RegisterTools(s *server.MCPServer, connector databases.DatabaseConnector) {
	// Sample tool
	sampleTool := goMCP.NewTool("sample_table",
		goMCP.WithDescription("Get sample data from a specific table"),
		goMCP.WithString("table",
			goMCP.Required(),
			goMCP.Description("Name of the table to sample"),
		),
		goMCP.WithNumber("limit",
			goMCP.Description("Number of rows to return (default: 10)"),
		),
	)

	// Query tool
	queryTool := goMCP.NewTool("query_database",
		goMCP.WithDescription("Execute a read-only SQL query on the database"),
		goMCP.WithString("query",
			goMCP.Required(),
			goMCP.Description("SQL query to execute (SELECT statements only)"),
		),
	)

	// Scan tool
	scanTool := goMCP.NewTool("scan_database",
		goMCP.WithDescription("Discover database tables and their structure"),
		goMCP.WithString("tables",
			goMCP.Description("Optional list of specific table names to scan. If empty, scans all tables"),
		),
	)

	s.AddTool(sampleTool, handlers.SampleHandler(connector))
	s.AddTool(queryTool, handlers.QueryHandler(connector))
	s.AddTool(scanTool, handlers.ScanHandler(connector))
}
