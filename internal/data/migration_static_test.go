package data

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBindingFirstMigrationDoesNotDeleteRows(t *testing.T) {
	migration := readMigrationForStaticTest(t, "000006_binding_first_devices_profiles_and_grant_invalidations.up.sql")
	normalized := normalizeSQLForStaticTest(migration)

	require.Regexp(t, regexp.MustCompile(`(?is)SIGNAL\s+SQLSTATE\s+''45000''[^;]*DEVICE_RECORDS`), migration)
	require.Regexp(t, regexp.MustCompile(`(?is)SIGNAL\s+SQLSTATE\s+''45000''[^;]*ACCOUNT_PROFILES`), migration)
	assertNoDestructiveTableMutation(t, normalized, "DEVICE_RECORDS")
	assertNoDestructiveTableMutation(t, normalized, "ACCOUNT_PROFILES")
}

func readMigrationForStaticTest(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "initialize", "migrate", "sql", name)
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.ToUpper(string(contents))
}

func normalizeSQLForStaticTest(sql string) string {
	normalized := strings.NewReplacer("`", "", "\"", "").Replace(sql)
	return strings.Join(strings.Fields(normalized), " ")
}

func assertNoDestructiveTableMutation(t *testing.T, sql, table string) {
	t.Helper()

	identifier := regexp.QuoteMeta(table)
	require.NotRegexp(t, regexp.MustCompile(`(?i)\bDELETE\b[^;]*\b`+identifier+`\b`), sql)
	require.NotRegexp(t, regexp.MustCompile(`(?i)\bTRUNCATE\b\s+(?:\bTABLE\b\s+)?\b`+identifier+`\b`), sql)
	require.NotRegexp(t, regexp.MustCompile(`(?i)\bDROP\b\s+(?:\bTEMPORARY\b\s+)?\bTABLE\b\s+(?:\bIF\b\s+\bEXISTS\b\s+)?\b`+identifier+`\b`), sql)
}
