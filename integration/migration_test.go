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

	"github.com/go-sql-driver/mysql"
)

type integrationStack struct {
	SQLDB       *sql.DB
	DatabaseCfg databaseConfig
	cleanup     func()
}

type databaseConfig struct {
	Driver string `yaml:"driver"`
	Source string `yaml:"source"`
	Dbname string
}

func TestMigrationsCreateCoreTables(t *testing.T) {
	stack := newIntegrationStack(t)
	t.Cleanup(stack.cleanup)

	requireMigrationsApplied(t, stack.SQLDB)
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "credential_records")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "device_records")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "account_profiles")
	requireTableExists(t, stack.SQLDB, stack.DatabaseCfg.Dbname, "runtime_artifacts")
}

func newIntegrationStack(t *testing.T) *integrationStack {
	t.Helper()

	databaseCfg := loadDatabaseConfig(t)
	serverDB, err := openServerDB(databaseCfg.Source)
	if err != nil {
		t.Skipf("integration database unavailable: %v", err)
	}
	ensureDatabaseExists(t, serverDB, databaseCfg.Dbname)

	appDB, err := openDatabase(databaseCfg.Source)
	if err != nil {
		_ = serverDB.Close()
		t.Skipf("integration database unavailable: %v", err)
	}
	resetCoreTables(t, appDB)

	return &integrationStack{
		SQLDB:       appDB,
		DatabaseCfg: databaseCfg,
		cleanup: func() {
			resetCoreTables(t, appDB)
			_ = appDB.Close()
			_ = serverDB.Close()
		},
	}
}

func loadDatabaseConfig(t *testing.T) databaseConfig {
	t.Helper()

	source := os.Getenv("TEST_DATABASE_SOURCE")
	if source == "" {
		t.Skip("TEST_DATABASE_SOURCE is required for integration database tests")
	}
	cfg := parseDatabaseConfig(t, source)
	cfg.Source = source
	if cfg.Dbname == "" {
		t.Fatal("TEST_DATABASE_SOURCE must include a database name")
	}
	if cfg.Dbname == "platform_mihomo" {
		t.Fatal("TEST_DATABASE_SOURCE must point to a dedicated test database, not platform_mihomo")
	}
	return cfg
}

func parseDatabaseConfig(t *testing.T, source string) databaseConfig {
	t.Helper()

	parsed, err := mysql.ParseDSN(source)
	if err != nil {
		t.Fatalf("parse database source: %v", err)
	}
	if parsed.DBName == "" {
		t.Fatal("database source missing db name")
	}

	return databaseConfig{Dbname: parsed.DBName}
}

func openServerDB(source string) (*sql.DB, error) {
	parsed, err := mysql.ParseDSN(source)
	if err != nil {
		return nil, fmt.Errorf("parse server dsn: %w", err)
	}
	parsed.DBName = ""

	return openDatabase(parsed.FormatDSN())
}

func openDatabase(source string) (*sql.DB, error) {
	db, err := sql.Open("mysql", source)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func ensureDatabaseExists(t *testing.T, db *sql.DB, dbName string) {
	t.Helper()

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create database %q: %v", dbName, err)
	}
}

func resetCoreTables(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, table := range []string{"runtime_artifacts", "account_profiles", "device_records", "credential_records"} {
		if _, err := db.Exec("DROP TABLE IF EXISTS `" + table + "`"); err != nil {
			t.Fatalf("drop table %q: %v", table, err)
		}
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
		statement, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %q: %v", path, err)
		}
		if _, err := db.Exec(string(statement)); err != nil {
			t.Fatalf("apply migration %q: %v", path, err)
		}
	}
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
