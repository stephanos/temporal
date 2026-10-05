package target

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/capabilityreview"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

// AdapterPreparedSourceSetSHA256 computes the prepared source-set identity
// that capability review pins for an adapter's prepared package when a target
// builds for goos/goarch. The go command selects the package's Go and foreign
// files for that platform under the target build environment (cgo disabled,
// the target experiment), and the files are hashed the way review hashes them.
// File selection reads only build constraints, so any host computes every
// platform's identity; the go command must be the pinned Go release, whose
// release tags decide version constraints.
//
// The package is listed by directory outside module mode, so neither the
// adapter module's requirements nor a network connection are needed. The
// import comment check that mode applies is the only error accepted, and only
// when the comment names importPath.
func AdapterPreparedSourceSetSHA256(ctx context.Context, goCommand, packageDirectory, importPath, goos, goarch string) (digest string, retErr error) {
	return adapterPreparedSourceSetSHA256With(ctx, goCommand, packageDirectory, importPath, goos, goarch, gocommand.Default())
}

const maximumAdapterListingBytes = 4 << 20

func adapterPreparedSourceSetSHA256With(ctx context.Context, goCommand, packageDirectory, importPath, goos, goarch string, runner gocommand.Runner) (digest string, retErr error) {
	gopath, err := os.MkdirTemp("", "gomad3-source-set-gopath-")
	if err != nil {
		return "", fmt.Errorf("create source-set GOPATH: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(gopath); cleanupErr != nil {
			digest = ""
			if retErr == nil {
				retErr = cleanupErr
			} else {
				retErr = errors.Join(retErr, cleanupErr)
			}
		}
	}()
	result, err := runner.Compatibility(ctx, gocommand.Request{
		Command: []string{goCommand, "list", "-e", "-find", "-json", "."}, Dir: packageDirectory,
		Env: append(targetbuild.Environment(), "GO111MODULE=off", "GOPATH="+gopath, "GOOS="+goos, "GOARCH="+goarch), OutputLimit: maximumAdapterListingBytes,
	})
	if err != nil {
		return "", fmt.Errorf("list prepared package %s for %s/%s: %w: %s", importPath, goos, goarch, err, strings.TrimSpace(string(result.Stderr)))
	}
	var listed struct {
		capabilityreview.Package
		Error *struct{ Err string }
	}
	if err := json.Unmarshal(result.Stdout, &listed); err != nil {
		return "", fmt.Errorf("decode prepared package %s listing: %w", importPath, err)
	}
	if listed.Error != nil && !strings.HasSuffix(listed.Error.Err, " expects import "+strconv.Quote(importPath)) {
		return "", fmt.Errorf("list prepared package %s for %s/%s: %s", importPath, goos, goarch, listed.Error.Err)
	}
	if listed.Dir == "" || listed.Name == "" {
		return "", errors.New("prepared package listing has no directory or name")
	}
	listed.ImportPath = importPath
	projected := CapabilityPackage{ImportPath: importPath, Name: listed.Name, Sources: []CapabilitySource{}}
	for _, name := range packageSourceFiles(listed.Package) {
		source, err := projectCapabilitySource(listed.Package, nil, name)
		if err != nil {
			return "", err
		}
		projected.Sources = append(projected.Sources, source)
	}
	projected.ForeignSources, err = projectForeignSources(listed.Package, nil)
	if err != nil {
		return "", err
	}
	return capabilityCompatibilityPackage(projected).SourceSetSHA256, nil
}
