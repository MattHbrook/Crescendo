package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open creates or opens the SQLite database at {dataPath}/crescendo.db and
// configures it for concurrent access with WAL journaling.
func Open(dataPath string) (*sql.DB, error) {
	dsn := dataPath + "/crescendo.db?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"

	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", dsn, err)
	}

	if err := handle.PingContext(context.Background()); err != nil {
		if closeErr := handle.Close(); closeErr != nil {
			return nil, fmt.Errorf("db: ping: %w (close: %w)", err, closeErr)
		}
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return handle, nil
}

// Migrate applies all pending SQL migrations from the embedded migrations/
// directory. Each migration runs inside a transaction and is recorded in the
// schema_migrations table so it is never applied twice.
func Migrate(handle *sql.DB) error {
	ctx := context.Background()

	const createMeta = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := handle.ExecContext(ctx, createMeta); err != nil {
		return fmt.Errorf("db: create schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("db: read embedded migrations: %w", err)
	}

	// Sort by filename to guarantee deterministic ordering.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		var exists int
		if err := handle.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name).Scan(&exists); err != nil {
			return fmt.Errorf("db: check migration %q: %w", name, err)
		}
		if exists > 0 {
			continue
		}

		body, err := fs.ReadFile(migrationFS, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("db: read migration %q: %w", name, err)
		}

		tx, err := handle.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("db: begin tx for %q: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("db: exec migration %q: %w (rollback: %w)", name, err, rbErr)
			}
			return fmt.Errorf("db: exec migration %q: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", name); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("db: record migration %q: %w (rollback: %w)", name, err, rbErr)
			}
			return fmt.Errorf("db: record migration %q: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit migration %q: %w", name, err)
		}
	}

	return nil
}
