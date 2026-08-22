package command

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestRunDeveloperDispatchesManifestGeneration(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, RunDeveloper(context.Background(), []string{
		"manifest", "-lean-version", protocolcatalog.LeanVersion,
	}, &output))
	require.JSONEq(t, `{
		"formatVersion": "umpire3/v2",
		"toolchain": {"lean": "4.28.0"}
	}`, output.String())
}

func TestRunDeveloperRejectsUnknownBuildOperation(t *testing.T) {
	err := RunDeveloper(context.Background(), []string{"unknown"}, &bytes.Buffer{})
	require.ErrorContains(t, err, `unknown developer operation "unknown"`)
}
