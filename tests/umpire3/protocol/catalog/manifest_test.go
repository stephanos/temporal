package catalog_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestEmptyManifestRecordsSchemaAndLeanVersions(t *testing.T) {
	manifest := catalog.NewEmptyManifest("4.33.0")

	require.Equal(t, catalog.FormatVersion, manifest.FormatVersion)
	require.Equal(t, "4.33.0", manifest.Toolchain.Lean)
}

func TestWriteManifestIsCanonical(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, catalog.WriteManifest(&output, catalog.NewEmptyManifest("4.33.0")))
	require.JSONEq(t, "{\n  \"formatVersion\": \"umpire3/v2\",\n  \"toolchain\": {\n    \"lean\": \"4.33.0\"\n  }\n}\n", output.String())
}
