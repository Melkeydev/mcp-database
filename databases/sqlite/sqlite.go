package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/melkeydev/mcp-database/types"
)

type SQLiteConnector struct {
	db *sqlx.DB
}

func NewSQLiteConnector(connectionString string) (*SQLiteConnector, error) {
	db, err := sqlx.Open("sqlite3", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	connector := &SQLiteConnector{
		db: db,
	}

	// Test the connection
	if err := connector.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return connector, nil
}

func (c *SQLiteConnector) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Discover
func (c *SQLiteConnector) Scan(ctx context.Context, tablesList []string) ([]types.Table, error) {
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Commit()

	var query string
	var args []interface{}

	if len(tablesList) > 0 {
		// Query specific tables
		placeholders := make([]string, len(tablesList))
		args = make([]interface{}, len(tablesList))

		for i, table := range tablesList {
			placeholders[i] = "?"
			args[i] = table
		}

		query = fmt.Sprintf(`
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' 
			AND name NOT LIKE 'sqlite_%%'
			AND name IN (%s)
		`, strings.Join(placeholders, ","))

	} else {
		query = `
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' 
			AND name NOT LIKE 'sqlite_%'
		`
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []types.Table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		columns, err := c.loadColumns(ctx, tx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to load columns for table %s: %w", tableName, err)
		}

		tables = append(tables, types.Table{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

// Query
func (c *SQLiteConnector) Query(ctx context.Context, sqlQuery string) ([]map[string]any, error) {
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("BeginTx failed with error: %w", err)
	}
	defer tx.Commit()

	rows, err := tx.QueryxContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("unable to query db: %w", err)
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		results = append(results, row)
	}

	return results, nil
}

// Sample
func (c *SQLiteConnector) Sample(ctx context.Context, table string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", table, limit)
	return c.Query(ctx, query)
}

func (c *SQLiteConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *SQLiteConnector) loadColumns(ctx context.Context, tx *sqlx.Tx, tableName string) ([]types.Column, error) {
	query := fmt.Sprintf("PRAGMA table_info('%s')", tableName)

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []types.Column
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue *string
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, types.Column{
			Name:     name,
			Type:     dataType,
			Nullable: notNull == 0,
		})
	}

	return columns, nil
}
