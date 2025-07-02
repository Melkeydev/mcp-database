package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/melkeydev/mcp-database/types"
)

type MySQLConnector struct {
	db *sqlx.DB
}

func NewMySQLConnector(connectionString string) (*MySQLConnector, error) {
	_, err := mysql.ParseDSN(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Open the database connection
	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	connector := &MySQLConnector{
		db: db,
	}

	if err := connector.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return connector, nil
}

func (c *MySQLConnector) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Discover
func (c *MySQLConnector) Scan(ctx context.Context, tablesList []string) ([]types.Table, error) {
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
			SELECT table_name, table_schema
			FROM information_schema.tables 
			WHERE table_type = 'BASE TABLE'
			AND table_schema = DATABASE()
			AND table_name IN (%s)
		`, strings.Join(placeholders, ","))

	} else {
		// Query all tables in the current database
		query = `
			SELECT table_name, table_schema
			FROM information_schema.tables 
			WHERE table_type = 'BASE TABLE'
			AND table_schema = DATABASE()
		`
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []types.Table
	for rows.Next() {
		var tableName, tableSchema string
		if err := rows.Scan(&tableName, &tableSchema); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		columns, err := c.loadColumns(ctx, tx, tableName, tableSchema)
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
func (c *MySQLConnector) Query(ctx context.Context, sqlQuery string) ([]map[string]any, error) {
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
func (c *MySQLConnector) Sample(ctx context.Context, table string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d", table, limit)
	return c.Query(ctx, query)
}

func (c *MySQLConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *MySQLConnector) loadColumns(ctx context.Context, tx *sqlx.Tx, tableName, tableSchema string) ([]types.Column, error) {
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = ? AND table_schema = ?
		ORDER BY ordinal_position
	`

	rows, err := tx.QueryContext(ctx, query, tableName, tableSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []types.Column
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, types.Column{
			Name:     name,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
	}

	return columns, nil
}
