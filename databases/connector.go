package databases

import (
	"context"
)

type Database interface {
	Ping(ctx context.Context) error
	Scan(ctx context.Context, tableList []string) ([]Table, error)
	DescribeTable(ctx context.Context, table string) ([]Column, error)
	ListTables(ctx context.Context) ([]string, error)
	Query(ctx context.Context, sql string) ([]map[string]any, error)
	Sample(ctx context.Context, table string, limit int) ([]map[string]any, error)
	Close() error
}

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

type Table struct {
	Name    string
	Columns []Column
}

// TODO add function to add database
