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

// DescribeTable returns detailed information about a specific table
func (c *MySQLConnector) DescribeTable(ctx context.Context, table string) (*types.TableDescription, error) {
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Commit()

	// Check if table exists
	var exists bool
	err = tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = DATABASE() AND table_name = ?
		)`, table)
	if err != nil {
		return nil, fmt.Errorf("failed to check table existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("table %s not found", table)
	}

	// Get current database name
	var dbName string
	err = tx.GetContext(ctx, &dbName, "SELECT DATABASE()")
	if err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}

	// Get columns
	columns, err := c.loadColumns(ctx, tx, table, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to load columns: %w", err)
	}

	// Get row count
	var rowCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)
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
		SELECT column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE()
		AND table_name = ?
		AND constraint_name = 'PRIMARY'
		ORDER BY ordinal_position`, table)
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
			index_name,
			GROUP_CONCAT(column_name ORDER BY seq_in_index) as columns,
			NOT non_unique as is_unique
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = ?
		AND index_name != 'PRIMARY'
		GROUP BY index_name, non_unique`, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	defer indexRows.Close()

	var indexes []types.Index
	for indexRows.Next() {
		var indexName string
		var columnNamesStr string
		var isUnique bool
		if err := indexRows.Scan(&indexName, &columnNamesStr, &isUnique); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}
		indexes = append(indexes, types.Index{
			Name:    indexName,
			Columns: strings.Split(columnNamesStr, ","),
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
