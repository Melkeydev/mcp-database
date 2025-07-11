# MCP Database Server

A Model Context Protocol (MCP) server that provides safe, read-only database access to Large Language Models (LLMs). This enables LLMs like Claude to explore database schemas, sample data, and execute SELECT queries through a standardized interface.

> ⚠️ **Early Development Notice**: This project is in early development. While functional, it lacks comprehensive testing and production-ready features. Use at your own risk in production environments.

## Features

- **Schema Exploration**: Discover database structure, tables, and columns
- **Data Sampling**: Preview table contents with configurable row limits
- **Safe Querying**: Execute read-only SELECT queries with built-in safety measures
- **Multi-Database Support**: PostgreSQL, MySQL, and SQLite
- **MCP Protocol**: Seamless integration with Claude Desktop and other MCP-compatible clients

## Installation

### Prerequisites

- Go 1.24.4 or higher
- Docker (for development database)
- One of the supported databases:
  - PostgreSQL
  - MySQL
  - SQLite

### Quick Start

1. Clone the repository:

```bash
git clone https://github.com/melkeydev/mcp-database
cd mcp-database
```

2. Start the development database (PostgreSQL):

```bash
docker-compose up -d
```

3. Configure your database connection in `config.yaml`:

```yaml
database:
  type: "postgres" # Options: postgres, mysql, sqlite
  connection_string: "postgres://postgres:postgres@localhost:5432/mcp_db?sslmode=disable"
  file: "database.db" # For SQLite only
```

4. Build and run:

```bash
go build -o mcp-database
./mcp-database -config config.yaml
```

## Usage with Claude Desktop

Add the server to your Claude Desktop configuration:

### macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`

### Windows: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "database": {
      "command": "/path/to/mcp-database",
      "args": ["-config", "/path/to/config.yaml"]
    }
  }
}
```

## Available Tools

The server exposes three MCP tools:

### 1. `scan_database`

Discovers the database schema including all tables, columns, and their types.

```typescript
// No parameters required
```

### 2. `sample_table`

Returns a sample of rows from a specified table.

```typescript
{
  "table_name": "users",  // Required
  "limit": 10            // Optional, default: 10
}
```

### 3. `query_database`

Executes a read-only SELECT query.

```typescript
{
  "query": "SELECT name, email FROM users WHERE created_at > '2024-01-01'"
}
```

## Configuration

The `config.yaml` file supports the following database configurations:

### PostgreSQL

```yaml
database:
  type: "postgres"
  connection_string: "postgres://user:password@host:port/dbname?sslmode=disable"
```

### MySQL

```yaml
database:
  type: "mysql"
  connection_string: "user:password@tcp(host:port)/dbname"
```

### SQLite

```yaml
database:
  type: "sqlite"
  file: "path/to/database.db"
```

## Architecture

```
mcp-database/
├── main.go              # Entry point
├── config/              # Configuration management
├── databases/           # Database connectors
│   ├── connector.go     # Common interface
│   ├── postgres/        # PostgreSQL implementation
│   ├── mysql/          # MySQL implementation
│   └── sqlite/         # SQLite implementation
├── handlers/           # Request handlers
├── mcp/               # MCP protocol implementation
└── types/             # Shared type definitions
```

## Safety Features

- **Read-only operations**: All database operations are executed within read-only transactions
- **Query validation**: Only SELECT statements are allowed
- **Resource limits**: Configurable row limits for data sampling
- **Error handling**: Comprehensive error handling and reporting

## Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build
go build -o mcp-database

# Run tests (coming soon)
go test ./...
```

### Adding a New Database

1. Create a new package under `databases/`
2. Implement the `DatabaseConnector` interface
3. Update `GetConnector()` in `databases/connector.go`
4. Add configuration support in `config/`

## Roadmap

- [ ] Comprehensive test suite
- [ ] Connection pooling
- [ ] Query result caching
- [ ] Support for more databases (Oracle, SQL Server)
- [ ] Query timeout configuration
- [ ] Result size limits
- [ ] Schema caching for better performance
- [ ] Support for custom SQL functions
- [ ] Docker image for easier deployment

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. Since this is an early-stage project, please open an issue first to discuss major changes.

### Areas where help is needed:

- Writing comprehensive tests
- Adding support for more databases
- Improving error messages and debugging
- Documentation improvements
- Performance optimizations

## License

[MIT License](LICENSE)

## Disclaimer

This project is in early development and not yet production-ready. While it implements safety measures for read-only access, please use caution when connecting to production databases. Always use appropriate credentials with minimal required permissions.

## Acknowledgments

Built on the [Model Context Protocol](https://modelcontextprotocol.io) by Anthropic.