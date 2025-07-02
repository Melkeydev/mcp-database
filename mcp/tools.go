package mcp

import (
	goMCP "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/melkeydev/mcp-database/databases"
	"github.com/melkeydev/mcp-database/handlers"
)

func RegisterTools(s *server.MCPServer, connector databases.DatabaseConnector) {
	// Scan tool - Use this FIRST to discover available tables
	scanTool := goMCP.NewTool("scan_database",
		goMCP.WithDescription(`Discover database tables and their structure. Use this tool FIRST before querying to understand the database schema.
Returns a list of tables with their columns, data types, and nullable information.
Examples:
- Scan all tables: tables=""
- Scan specific tables: tables="users,orders,products"`),
		goMCP.WithString("tables",
			goMCP.Description("Comma-separated list of table names to scan. Leave empty to scan all tables. Example: 'users,orders' or empty string for all"),
		),
	)

	// Sample tool - Use to preview table data
	sampleTool := goMCP.NewTool("sample_table",
		goMCP.WithDescription(`Get a preview of data from a specific table. Useful for understanding table contents before writing queries.
Returns actual row data from the specified table.
Use scan_database first to discover available tables.
Examples:
- Sample 10 rows: table="users", limit=10
- Sample default rows: table="products" (defaults to 10 rows)`),
		goMCP.WithString("table",
			goMCP.Required(),
			goMCP.Description("Exact name of the table to sample (case-sensitive). Get table names from scan_database first"),
		),
		goMCP.WithNumber("limit",
			goMCP.Description("Number of rows to return. Default: 10, Maximum recommended: 100"),
		),
	)

	// Query tool - Execute SQL queries
	queryTool := goMCP.NewTool("query_database",
		goMCP.WithDescription(`Execute a read-only SQL query on the database. Only SELECT statements are allowed.
Use scan_database first to understand the schema, then write your query.
The query must be valid SQL for the database type (PostgreSQL, MySQL, or SQLite).
Examples:
- Simple query: "SELECT * FROM users WHERE age > 21"
- Join query: "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id"
- Aggregate query: "SELECT category, COUNT(*) as count FROM products GROUP BY category"`),
		goMCP.WithString("query",
			goMCP.Required(),
			goMCP.Description("SQL SELECT query to execute. Must be a valid SELECT statement. Other operations (INSERT, UPDATE, DELETE) are not allowed"),
		),
	)

	s.AddTool(scanTool, handlers.ScanHandler(connector))
	s.AddTool(sampleTool, handlers.SampleHandler(connector))
	s.AddTool(queryTool, handlers.QueryHandler(connector))
}

// Helper Function
func GetToolUsageGuide() string {
	return `
Database MCP Tools Usage Guide:

1. ALWAYS start with 'scan_database' to discover available tables and their structure
2. Use 'sample_table' to preview data and understand table contents
3. Use 'query_database' to execute specific SELECT queries

Workflow example:
- First: scan_database (discover schema)
- Then: sample_table with table="users" (preview data)
- Finally: query_database with query="SELECT * FROM users WHERE created_at > '2024-01-01'"
`
}

