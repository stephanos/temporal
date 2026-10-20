package installation

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/temporalinput"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

// TemporalConfig derives a guest directory from one checkout-relative package.
func TemporalConfig(checkout, pkg string) (wasi.Config, error) {
	clean := path.Clean(pkg)
	if checkout == "" || pkg == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(pkg, "\\\x00") || strings.Contains(pkg, "...") {
		return wasi.Config{}, fmt.Errorf("temporal setup requires a checkout and one relative package directory")
	}
	root, err := filepath.Abs(checkout)
	if err != nil {
		return wasi.Config{}, err
	}
	inputs, err := temporalinput.CaptureSchema(root)
	if err != nil {
		return wasi.Config{}, fmt.Errorf("capture Temporal schema/sqlite: %w", err)
	}
	return wasi.TemporalConfig(inputs, path.Join("/workspace", clean), wasi.TemporalInstallation)
}
