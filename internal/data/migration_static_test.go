package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBindingFirstMigrationDoesNotDeleteRows(t *testing.T) {
	migration := readMigrationForStaticTest(t, "000006_binding_first_devices_profiles_and_grant_invalidations.up.sql")

	require.NotContains(t, migration, "DELETE D FROM DEVICE_RECORDS")
	require.NotContains(t, migration, "DELETE P FROM ACCOUNT_PROFILES")
}

func readMigrationForStaticTest(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "initialize", "migrate", "sql", name)
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.ToUpper(string(contents))
}
