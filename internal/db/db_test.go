package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestOpen(t *testing.T) {
	t.Run("opens database in temp dir", func(t *testing.T) {
		dir := t.TempDir()

		handle, err := Open(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer handle.Close()

		if err := handle.PingContext(context.Background()); err != nil {
			t.Fatalf("ping failed: %v", err)
		}
	})
}

func TestOpen_invalid_path(t *testing.T) {
	t.Run("returns error for non-existent nested path", func(t *testing.T) {
		_, err := Open("/no/such/deeply/nested/path")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMigrate(t *testing.T) {
	t.Run("creates all expected tables", func(t *testing.T) {
		handle := openTestDB(t)

		if err := Migrate(handle); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		wantTables := []string{"artist_mapping", "library_albums", "downloads"}
		for _, table := range wantTables {
			assertTableExists(t, handle, table)
		}
	})
}

func TestMigrate_idempotent(t *testing.T) {
	t.Run("running twice produces no error", func(t *testing.T) {
		handle := openTestDB(t)

		if err := Migrate(handle); err != nil {
			t.Fatalf("first migrate: %v", err)
		}
		if err := Migrate(handle); err != nil {
			t.Fatalf("second migrate: %v", err)
		}
	})
}

func TestMigrate_records_version(t *testing.T) {
	t.Run("migration version is recorded in schema_migrations", func(t *testing.T) {
		handle := openTestDB(t)

		if err := Migrate(handle); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		var count int
		err := handle.QueryRowContext(
			context.Background(),
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
			"001_create_tables.sql",
		).Scan(&count)
		if err != nil {
			t.Fatalf("query schema_migrations: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 row for 001_create_tables.sql, got %d", count)
		}
	})
}

// openTestDB is a helper that opens an isolated SQLite database in a temp dir
// and registers a cleanup to close it when the test finishes.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dir := t.TempDir()
	handle, err := Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { handle.Close() })

	return handle
}

// assertTableExists fails the test if the given table is not present in
// sqlite_master.
func assertTableExists(t *testing.T, handle *sql.DB, table string) {
	t.Helper()

	var count int
	err := handle.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for %q: %v", table, err)
	}
	if count == 0 {
		t.Errorf("table %q does not exist", table)
	}
}
