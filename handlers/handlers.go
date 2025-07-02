package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/melkeydev/mcp-database/databases"
)

// SampleHandler creates a handler for the sample_table tool
func SampleHandler(connector databases.DatabaseConnector) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		table, err := request.RequireString("table")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing table parameter: %v", err)), nil
		}

		limit := 10

		results, err := connector.Sample(ctx, table, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sample failed: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// QueryHandler creates a handler for the query_database tool
func QueryHandler(connector databases.DatabaseConnector) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing query parameter: %v", err)), nil
		}

		results, err := connector.Query(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// ScanHandler creates a handler for the scan_database tool
func ScanHandler(connector databases.DatabaseConnector) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var tablesList []string

		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if tablesParam, exists := args["tables"]; exists {
				if tablesArray, ok := tablesParam.([]interface{}); ok {
					for _, table := range tablesArray {
						if tableStr, ok := table.(string); ok {
							tablesList = append(tablesList, tableStr)
						}
					}
				}
			}
		}

		tables, err := connector.Scan(ctx, tablesList)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Scan failed: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(tables, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
