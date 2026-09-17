package destination

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/engine"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDestination struct {
	db *sql.DB
}

func NewMySQLDestination(cfg *config.DBConfig) (*MySQLDestination, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping destination db: %w", err)
	}

	return &MySQLDestination{db: db}, nil
}

func (m *MySQLDestination) SaveData(ctx context.Context, records []engine.Record) error {
	if len(records) == 0 {
		return nil
	}

	// Use a transaction for the batch
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, rec := range records {
		var cols []string
		var placeholders []string
		var args []interface{}

		// Dynamically build the insert query based on mapped fields
		for k, v := range rec {
			cols = append(cols, k)
			placeholders = append(placeholders, "?")
			args = append(args, v)
		}

		// Ensure we insert into destination_logs and ignore duplicates to prevent crashes
		query := fmt.Sprintf(
			"INSERT IGNORE INTO logs (%s) VALUES (%s)",
			strings.Join(cols, ", "),
			strings.Join(placeholders, ", "),
		)

		_, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute insert: %w", err)
		}
	}

	return tx.Commit()
}
