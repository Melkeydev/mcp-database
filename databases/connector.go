package databases

import (
	"context"
	"fmt"

	"github.com/melkeydev/mcp-database/databases/postgres"
	"github.com/melkeydev/mcp-database/types"
)

type DatabaseConnector interface {
	Ping(ctx context.Context) error
	Scan(ctx context.Context, tableList []string) ([]types.Table, error)
	// DescribeTable(ctx context.Context, table string) ([]types.Column, error)
	// ListTables(ctx context.Context) ([]string, error)
	Query(ctx context.Context, sql string) ([]map[string]any, error)
	Sample(ctx context.Context, table string, limit int) ([]map[string]any, error)
	Close() error
}

func NewConnector(dbType, connectionString string) (DatabaseConnector, error) {
	switch dbType {
	case "postgres", "postgresql":
		return postgres.NewPostgresConnector(connectionString)
	// case "mysql":
	// 	return mysql.NewMySQLConnector(connectionString)
	// case "sqlite":
	// 	return sqlite.NewSQLiteConnector(connectionString)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
