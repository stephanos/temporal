package installation

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

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
	config, err := wasi.TemporalConfig(root, path.Join("/workspace", clean))
	if err != nil {
		return wasi.Config{}, err
	}
	config.NetworkV2 = wasi.TemporalNetwork()
	return config, nil
}
