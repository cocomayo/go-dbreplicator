package source

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/engine"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLSource struct {
	db    *sql.DB
	query string
}

// NewMySQLSource initializes the connection pool to the source database.
func NewMySQLSource(cfg *config.DBConfig, query string) (*MySQLSource, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &MySQLSource{
		db:    db,
		query: query,
	}, nil
}

// FetchData runs the query using the high-water mark and extracts dynamic columns.
func (m *MySQLSource) FetchData(ctx context.Context, highWaterMark time.Time) ([]engine.Record, error) {
	// If it's the first run, we might pass an empty string, so fallback to an old date
	if highWaterMark.IsZero() {
		highWaterMark = time.Unix(0, 0).UTC()
	}

	rows, err := m.db.QueryContext(ctx, m.query, highWaterMark)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var records []engine.Record

	for rows.Next() {
		// Create a slice of interface{} to hold the dynamic column values
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Build the record map
		record := make(engine.Record)
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})

			// Handle byte slices (often returned by driver for strings/decimals)
			if b, ok := (*val).([]byte); ok {
				record[colName] = string(b)
			} else {
				record[colName] = *val
			}
		}
		records = append(records, record)
	}

	return records, nil
}
