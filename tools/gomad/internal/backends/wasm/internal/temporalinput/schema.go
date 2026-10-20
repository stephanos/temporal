package temporalinput

import (
	"path/filepath"

	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
)

func CaptureSchema(root string) (captured.CapturedInputs, error) {
	return captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
}
