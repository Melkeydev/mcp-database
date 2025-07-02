# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Model Context Protocol (MCP) database server that provides a standardized interface for AI assistants to interact with SQL databases. It implements read-only database operations exposed as MCP tools: `scan_database`, `sample_table`, and `query_database`.

## Build and Development Commands

```bash
# Start the development database (PostgreSQL)
docker-compose up -d

# Build the project
go build -o main

# Run the server
./main -config config.yaml

# Run directly without building
go run main.go -config config.yaml

# Install/update dependencies
go mod download
go mod tidy

# Format code
go fmt ./...
```

## Architecture

The project follows an interface-based design with clear separation of concerns:

- **databases/**: Database connector implementations
  - `connector.go`: Defines the `DatabaseConnector` interface
  - Each subdirectory (postgres/, mysql/, sqlite/) implements this interface
- **mcp/**: MCP protocol implementation
  - `tools.go`: Defines the three MCP tools and their handlers
- **handlers/**: Request processing logic
  - `handlers.go`: Implements the business logic for each MCP tool
- **config/**: Configuration management
  - Loads database connection details from `config.yaml`
- **types/**: Shared type definitions

## Key Implementation Details

1. **Database Safety**: All operations are read-only. The code enforces this by:
   - Using transactions with `ReadOnly: true`
   - Only exposing SELECT query capabilities
   - No write operations in any database connector

2. **Adding New Database Support**:
   - Create a new package under `databases/`
   - Implement the `DatabaseConnector` interface
   - Update `databases/connector.go` to include the new type in `GetConnector()`
   - Update `config.yaml` to support the new database type

3. **MCP Tool Structure**: Each tool in `mcp/tools.go` follows the pattern:
   - Tool definition with schema
   - Handler function that validates input and calls appropriate handler
   - Response formatting

## Important Notes

- The project currently lacks tests - consider adding unit tests for database connectors and integration tests for MCP handlers
- Error handling is consistent throughout - maintain this pattern
- Database connections are created per request - consider connection pooling for production use
- The MCP server runs on stdio by default (standard input/output communication)