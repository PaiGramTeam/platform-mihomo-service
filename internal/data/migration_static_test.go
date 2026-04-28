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

func TestBindingIDBackfillMigrationPrechecksLegacyAccountIDsBeforeDDL(t *testing.T) {
	migration := readMigrationForStaticTest(t, "000005_add_binding_id_to_credentials_and_profiles.up.sql")
	normalized := normalizeSQLForStaticTest(migration)

	credentialSignal := strings.Index(normalized, "SIGNAL SQLSTATE ''45000''")
	credentialAddColumn := strings.Index(normalized, "ALTER TABLE CREDENTIAL_RECORDS ADD COLUMN BINDING_ID")
	require.NotEqual(t, -1, credentialSignal)
	require.NotEqual(t, -1, credentialAddColumn)
	require.Less(t, credentialSignal, credentialAddColumn)

	profileSignal := strings.LastIndex(normalized, "SIGNAL SQLSTATE ''45000''")
	profileAddColumn := strings.Index(normalized, "ALTER TABLE ACCOUNT_PROFILES ADD COLUMN BINDING_ID")
	require.NotEqual(t, -1, profileSignal)
	require.NotEqual(t, -1, profileAddColumn)
	require.Less(t, profileSignal, profileAddColumn)
}

func TestDestructiveTableMutationPatternDetectsMultiTableDrop(t *testing.T) {
	profilePattern := destructiveTableMutationPattern("ACCOUNT_PROFILES")
	devicePattern := destructiveTableMutationPattern("DEVICE_RECORDS")

	require.Regexp(t, profilePattern, "DROP TABLE OTHER_TABLE, ACCOUNT_PROFILES")
	require.Regexp(t, devicePattern, "DROP TEMPORARY TABLE IF EXISTS DEVICE_RECORDS, OTHER_TABLE")
	require.NotRegexp(t, profilePattern, "DROP TABLE ACCOUNT_PROFILE_SNAPSHOTS")
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
	require.NotRegexp(t, destructiveTableMutationPattern(table), sql)
}

func destructiveTableMutationPattern(table string) *regexp.Regexp {
	identifier := regexp.QuoteMeta(table)
	return regexp.MustCompile(`(?i)\bDROP\b\s+(?:\bTEMPORARY\b\s+)?\bTABLE\b\s+(?:\bIF\b\s+\bEXISTS\b\s+)?[^;]*\b` + identifier + `\b`)
}
