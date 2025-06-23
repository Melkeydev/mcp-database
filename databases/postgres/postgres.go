package databases

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

type Table struct {
	Name    string
	Columns []Column
}

type PostgresConnector struct {
	db *sqlx.DB
	// schema string
}

func NewPostgresConnector(connectionString string) (*PostgresConnector, error) {
	config, err := pgx.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	config.PreferSimpleProtocol = true

	db := sqlx.NewDb(stdlib.OpenDB(*config), "pgx")

	connector := &PostgresConnector{
		db: db,
		// schema: schema,
	}

	// Test the connection
	if err := connector.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return connector, nil
}

// TODO: continue this
func (c *PostgresConnector) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Discover
func (c *PostgresConnector) Scan(ctx context.Context, tablesList []string) ([]Table, error) {
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
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = table
		}

		query = fmt.Sprintf(`
			SELECT table_name, table_schema
			FROM information_schema.tables 
			WHERE table_type = 'BASE TABLE'
			AND table_name IN (%s)
		`, strings.Join(placeholders, ","))

	} else {
		// Query all tables
		query = `
			SELECT table_name, table_schema
			FROM information_schema.tables 
			WHERE table_type = 'BASE TABLE'
		`
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var tableName, tableSchema string
		if err := rows.Scan(&tableName, &tableSchema); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		columns, err := c.loadColumns(ctx, tx, tableName, tableSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to load columns: %w", err)
		}

		fqtn := fmt.Sprintf(`"%s"."%s"`, tableSchema, tableName)
		tables = append(tables, Table{
			Name:    fqtn,
			Columns: columns,
		})
	}

	return tables, nil
}

// Query
func (c *PostgresConnector) Query(ctx context.Context, sqlQuery string) ([]map[string]any, error) {
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
func (c *PostgresConnector) Sample(ctx context.Context, table string, limit int) ([]map[string]any, error) {
	var test []map[string]any
	return test, nil
}

func (c *PostgresConnector) loadColumns(ctx context.Context, tx *sqlx.Tx, tableName, tableSchema string) ([]Column, error) {
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema = $2
		ORDER BY ordinal_position
	`

	rows, err := tx.QueryContext(ctx, query, tableName, tableSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, Column{
			Name:     name,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
	}

	return columns, nil
}
