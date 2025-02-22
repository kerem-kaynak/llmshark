// internal/postgres/client.go
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kerem-kaynak/llmshark/internal/storage"
)

type Schema struct {
	Name     string
	Tables   []Table
	Selected bool
	Expanded bool
}

type Table struct {
	Name        string
	Description string
	Columns     []Column
	Selected    bool
	Expanded    bool
}

type Column struct {
	Name        string
	Type        string
	Description string
	IsNullable  bool
	HasDefault  bool
	Default     string
	IsPrimary   bool
	IsUnique    bool
	Constraints []string
	Selected    bool
}

type SchemaFilter struct {
	ExcludeSchemas []string
	IncludeSchemas []string
}

var DefaultSchemaFilter = SchemaFilter{
	ExcludeSchemas: []string{"pg_catalog", "information_schema"},
}

type Client struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, creds *storage.Credentials) (*Client, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		creds.User, creds.Password, creds.Host, creds.Port, creds.Database)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}

	config.MaxConns = 4
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	return &Client{pool: pool}, nil
}

func (c *Client) GetSchemas(ctx context.Context, filter SchemaFilter) ([]Schema, error) {
	query := `
        WITH RECURSIVE 
        schemas AS (
            SELECT n.nspname, n.oid
            FROM pg_namespace n
            WHERE n.nspname = ANY($1::text[])
            OR (
                n.nspname != ALL($2::text[])
                AND n.nspname NOT LIKE 'pg_%'
                AND n.nspname != 'information_schema'
                AND array_length($1::text[], 1) IS NULL
            )
        ),
        base_tables AS (
            SELECT 
                s.nspname as schema_name,
                c.relname as table_name,
                c.oid as table_oid,
                obj_description(c.oid, 'pg_class') as table_description
            FROM schemas s
            JOIN pg_class c ON c.relnamespace = s.oid
            WHERE c.relkind = 'r'
            AND NOT c.relispartition
        ),
        columns AS (
            SELECT 
                t.schema_name,
                t.table_name,
                t.table_description,
                a.attname as column_name,
                pg_catalog.format_type(a.atttypid, a.atttypmod) as column_type,
                col_description(t.table_oid, a.attnum) as column_description,
                a.attnotnull as not_null,
                a.atthasdef as has_default,
                pg_get_expr(d.adbin, d.adrelid) as column_default,
                EXISTS (
                    SELECT 1 FROM pg_constraint c 
                    WHERE c.conrelid = t.table_oid 
                    AND c.contype = 'p' 
                    AND a.attnum = ANY(c.conkey)
                ) as is_primary,
                EXISTS (
                    SELECT 1 FROM pg_constraint c 
                    WHERE c.conrelid = t.table_oid 
                    AND c.contype = 'u' 
                    AND a.attnum = ANY(c.conkey)
                ) as is_unique,
                (
                    SELECT array_agg(c.conname) 
                    FROM pg_constraint c 
                    WHERE c.conrelid = t.table_oid 
                    AND a.attnum = ANY(c.conkey)
                ) as constraints
            FROM base_tables t
            JOIN pg_attribute a ON a.attrelid = t.table_oid
            LEFT JOIN pg_attrdef d ON d.adrelid = t.table_oid AND d.adnum = a.attnum
            WHERE a.attnum > 0 
            AND NOT a.attisdropped
            ORDER BY t.schema_name, t.table_name, a.attnum
        )
        SELECT * FROM columns;
    `

	includeSchemas := filter.IncludeSchemas
	excludeSchemas := filter.ExcludeSchemas
	if len(excludeSchemas) == 0 {
		excludeSchemas = DefaultSchemaFilter.ExcludeSchemas
	}

	rows, err := c.pool.Query(ctx, query, includeSchemas, excludeSchemas)
	if err != nil {
		return nil, fmt.Errorf("schema query failed: %w", err)
	}
	defer rows.Close()

	schemaMap := make(map[string]*Schema)

	for rows.Next() {
		var (
			schemaName, tableName                string
			tableDesc, colName, colType, colDesc sql.NullString
			notNull, hasDefault                  bool
			colDefault                           sql.NullString
			isPrimary, isUnique                  bool
			constraints                          []string
		)

		if err := rows.Scan(
			&schemaName, &tableName, &tableDesc,
			&colName, &colType, &colDesc,
			&notNull, &hasDefault, &colDefault,
			&isPrimary, &isUnique, &constraints,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Get or create schema
		schema, ok := schemaMap[schemaName]
		if !ok {
			schema = &Schema{
				Name:     schemaName,
				Tables:   make([]Table, 0),
				Expanded: true,
			}
			schemaMap[schemaName] = schema
		}

		// Find or create table
		var table *Table
		for i := range schema.Tables {
			if schema.Tables[i].Name == tableName {
				table = &schema.Tables[i]
				break
			}
		}
		if table == nil {
			schema.Tables = append(schema.Tables, Table{
				Name:        tableName,
				Description: tableDesc.String,
				Columns:     make([]Column, 0),
				Expanded:    true,
			})
			table = &schema.Tables[len(schema.Tables)-1]
		}

		// Add column
		column := Column{
			Name:        colName.String,
			Type:        colType.String,
			Description: colDesc.String,
			IsNullable:  !notNull,
			HasDefault:  hasDefault,
			Default:     colDefault.String,
			IsPrimary:   isPrimary,
			IsUnique:    isUnique,
			Constraints: constraints,
		}
		table.Columns = append(table.Columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration failed: %w", err)
	}

	// Convert map to slice
	schemas := make([]Schema, 0, len(schemaMap))
	for _, schema := range schemaMap {
		schemas = append(schemas, *schema)
	}

	return schemas, nil
}

// Common error messages
var (
	ErrCommentTooLong   = errors.New("comment exceeds maximum length of 1000 characters")
	ErrCommentEmpty     = errors.New("comment cannot be empty")
	ErrCommentMalicious = errors.New("comment contains potentially malicious content")
)

// sanitizeComment cleans and validates the comment input
func sanitizeComment(comment string) (string, error) {
	// Trim spaces
	comment = strings.TrimSpace(comment)

	// Check if empty after trimming
	if comment == "" {
		return "", ErrCommentEmpty
	}

	// Check length (PostgreSQL has a limit, but we'll be conservative)
	if len(comment) > 1000 {
		return "", ErrCommentTooLong
	}

	// Check for potential SQL injection patterns
	sqlInjectionPatterns := []string{
		"--;",
		"/*",
		"*/",
		"@@",
		"EXEC",
		"EXECUTE",
		"UNION",
		"SELECT",
		"DELETE",
		"DROP",
		"UPDATE",
		"INSERT",
	}

	commentUpper := strings.ToUpper(comment)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(commentUpper, pattern) {
			return "", ErrCommentMalicious
		}
	}

	// Remove or replace potentially problematic characters
	// Keep alphanumeric, basic punctuation, and common special characters
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r):
			return r
		case unicode.IsNumber(r):
			return r
		case unicode.IsSpace(r):
			return r
		case strings.ContainsRune(`.,!?()-_:;'"`, r):
			return r
		default:
			return -1 // Drop other characters
		}
	}, comment)

	// Escape single quotes for SQL
	sanitized = strings.ReplaceAll(sanitized, "'", "''")

	return sanitized, nil
}

func (c *Client) UpdateComment(ctx context.Context, schema, table, column, comment string) error {
	// Sanitize the comment (user input)
	sanitizedComment, err := sanitizeComment(comment)
	if err != nil {
		return fmt.Errorf("invalid comment: %w", err)
	}

	var query string
	if column == "" {
		query = fmt.Sprintf(`COMMENT ON TABLE "%s"."%s" IS '%s'`,
			schema,
			table,
			sanitizedComment)
	} else {
		query = fmt.Sprintf(`COMMENT ON COLUMN "%s"."%s"."%s" IS '%s'`,
			schema,
			table,
			column,
			sanitizedComment)
	}

	// Execute the query
	_, err = c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

func (c *Client) VerifyComment(ctx context.Context, schema, table, column string) (string, error) {
	var query string
	var args []interface{}

	if column == "" {
		query = `
            SELECT obj_description(
                (quote_ident($1) || '.' || quote_ident($2))::regclass
            );
        `
		args = []interface{}{schema, table}
	} else {
		query = `
            SELECT col_description(
                (quote_ident($1) || '.' || quote_ident($2))::regclass,
                (SELECT attnum FROM pg_attribute 
                 WHERE attrelid = (quote_ident($1) || '.' || quote_ident($2))::regclass 
                 AND attname = $3)
            );
        `
		args = []interface{}{schema, table, column}
	}

	var comment sql.NullString
	err := c.pool.QueryRow(ctx, query, args...).Scan(&comment)
	if err != nil {
		return "", fmt.Errorf("failed to verify comment: %w", err)
	}

	return comment.String, nil
}

func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}
