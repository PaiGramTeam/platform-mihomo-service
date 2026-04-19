//go:build integration

package integration

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestMigrationsCreateCoreTables(t *testing.T) {
	stack := newIntegrationStack(t)
	t.Cleanup(stack.cleanup)

	requireMigrationsApplied(t, stack.SQLDB)
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "credential_records")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "device_records")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "account_profiles")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "runtime_artifacts")
}

func TestBindingMigrationRejectsUnknownLegacyPlatformAccountIDs(t *testing.T) {
	stack := newIntegrationStack(t)
	t.Cleanup(stack.cleanup)

	applyMigrationFile(t, stack.SQLDB, migrationPath(t, "000001_create_credential_records.up.sql"))
	applyMigrationFile(t, stack.SQLDB, migrationPath(t, "000003_create_account_profiles.up.sql"))

	_, err := stack.SQLDB.Exec(`
		INSERT INTO credential_records (
			platform_account_id, platform, account_id, region, credential_blob, credential_version, status
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "legacy_10001", "mihomo", "10001", "cn_gf01", "{}", "v1", "active")
	if err != nil {
		t.Fatalf("insert legacy credential row: %v", err)
	}

	_, err = stack.SQLDB.Exec(`
		INSERT INTO account_profiles (
			platform_account_id, game_biz, region, player_id, nickname, level, is_default
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "legacy_10001", "hk4e_cn", "cn_gf01", "1008611", "Traveler", 1, true)
	if err != nil {
		t.Fatalf("insert legacy profile row: %v", err)
	}

	err = execMigrationFile(stack.SQLDB, migrationPath(t, "000005_add_binding_id_to_credentials_and_profiles.up.sql"))
	if err == nil {
		t.Fatal("expected binding migration to fail for unknown legacy platform_account_id")
	}
}

func requireMigrationsApplied(t *testing.T, db *sql.DB) {
	t.Helper()

	pattern := filepath.Join(repoRoot(t), "initialize", "migrate", "sql", "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migration files found for pattern %q", pattern)
	}
	sort.Strings(files)

	for _, path := range files {
		if err := execMigrationFile(db, path); err != nil {
			t.Fatalf("apply migration %q: %v", path, err)
		}
	}
}

func migrationPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "initialize", "migrate", "sql", name)
}

func applyMigrationFile(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	if err := execMigrationFile(db, path); err != nil {
		t.Fatalf("apply migration %q: %v", path, err)
	}
}

func execMigrationFile(db *sql.DB, path string) error {
	statement, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %q: %w", path, err)
	}
	if _, err := db.Exec(string(statement)); err != nil {
		return err
	}
	return nil
}

func requireTableExists(t *testing.T, db *sql.DB, schema string, table string) {
	t.Helper()

	const query = `
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name = ?
		LIMIT 1
	`

	var exists int
	err := db.QueryRow(query, schema, table).Scan(&exists)
	if err == nil {
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("table %q does not exist in schema %q", table, schema)
	}
	t.Fatalf("query table %q existence: %v", table, err)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("stat repo root %q: %v", root, err)
	}

	return root
}
