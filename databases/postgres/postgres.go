package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/melkeydev/mcp-database/types"
)

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

func (c *PostgresConnector) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Discover
func (c *PostgresConnector) Scan(ctx context.Context, tablesList []string) ([]types.Table, error) {
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

	var tables []types.Table
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
		tables = append(tables, types.Table{
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
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", table, limit)
	return c.Query(ctx, query)
}

func (c *PostgresConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *PostgresConnector) loadColumns(ctx context.Context, tx *sqlx.Tx, tableName, tableSchema string) ([]types.Column, error) {
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

// DescribeTable returns detailed information about a specific table
func (c *PostgresConnector) DescribeTable(ctx context.Context, table string) (*types.TableDescription, error) {
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Commit()

	// Parse table name to extract schema and table
	parts := strings.Split(table, ".")
	var tableSchema, tableName string
	if len(parts) == 2 {
		tableSchema = strings.Trim(parts[0], `"`)
		tableName = strings.Trim(parts[1], `"`)
	} else {
		tableSchema = "public"
		tableName = strings.Trim(table, `"`)
	}

	// Check if table exists
	var exists bool
	err = tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = $1 AND table_name = $2
		)`, tableSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to check table existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("table %s not found", table)
	}

	// Get columns
	columns, err := c.loadColumns(ctx, tx, tableName, tableSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to load columns: %w", err)
	}

	// Get row count
	var rowCount int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s"`, tableSchema, tableName)
	err = tx.GetContext(ctx, &rowCount, countQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get row count: %w", err)
	}

	// Get sample data
	sampleData, err := c.Sample(ctx, table, 5)
	if err != nil {
		// Non-critical error, continue without sample data
		sampleData = nil
	}

	// Get primary keys
	rows, err := tx.QueryContext(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE i.indisprimary
		AND n.nspname = $1
		AND c.relname = $2
		ORDER BY array_position(i.indkey, a.attnum)`, tableSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var pkColumn string
		if err := rows.Scan(&pkColumn); err != nil {
			return nil, fmt.Errorf("failed to scan primary key: %w", err)
		}
		primaryKeys = append(primaryKeys, pkColumn)
	}

	// Get indexes
	indexRows, err := tx.QueryContext(ctx, `
		SELECT 
			i.relname as index_name,
			array_agg(a.attname ORDER BY array_position(idx.indkey, a.attnum)) as columns,
			idx.indisunique as is_unique
		FROM pg_index idx
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_class c ON c.oid = idx.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = idx.indrelid AND a.attnum = ANY(idx.indkey)
		WHERE n.nspname = $1
		AND c.relname = $2
		AND NOT idx.indisprimary
		GROUP BY i.relname, idx.indisunique`, tableSchema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	defer indexRows.Close()

	var indexes []types.Index
	for indexRows.Next() {
		var indexName string
		var columnNames []string
		var isUnique bool
		if err := indexRows.Scan(&indexName, &columnNames, &isUnique); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}
		indexes = append(indexes, types.Index{
			Name:    indexName,
			Columns: columnNames,
			Unique:  isUnique,
		})
	}

	return &types.TableDescription{
		Name:        table,
		Columns:     columns,
		RowCount:    rowCount,
		SampleData:  sampleData,
		PrimaryKeys: primaryKeys,
		Indexes:     indexes,
	}, nil
}
