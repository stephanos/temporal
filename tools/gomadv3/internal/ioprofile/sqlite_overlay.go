package ioprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const (
	sqliteModuleVersion = "v1.51.0"
	sqliteSourceSHA256  = "sha256:49e9d6f24ca24c12a0cd99593655d5236eaf88214d7b1e0fc94a5262c44e5180"
	maximumModuleFiles  = 4096
	maximumModuleBytes  = 512 << 20
)

type BuildOverlay struct {
	Path              string
	Source            string
	Replacement       string
	SourceSHA256      string
	ReplacementSHA256 string
}

func (profile Profile) PrepareBuildOverlay(spec target.Spec, moduleCache string) (target.Spec, BuildOverlay, error) {
	if _, found := profileArgument(profile.Name); !found {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("unknown I/O profile %q", profile.Name)
	}
	if moduleCache == "" || spec.PreparationRoot == "" {
		return target.Spec{}, BuildOverlay{}, errors.New("I/O profile build overlay requires module cache and preparation root")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "modernc.org", "sqlite@"+sqliteModuleVersion))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve pinned SQLite module: %w", err)
	}
	source := filepath.Join(moduleSource, "lib", "sqlite_darwin_arm64.go")
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() {
		return target.Spec{}, BuildOverlay{}, errors.New("pinned SQLite source is not a regular file")
	}
	contents, err := os.ReadFile(source)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("read pinned SQLite source: %w", err)
	}
	if digestBytes(contents) != sqliteSourceSHA256 {
		return target.Spec{}, BuildOverlay{}, errors.New("pinned SQLite source identity mismatch")
	}
	replacementContents, err := rewriteSQLiteSource(contents)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	preparationRoot, err := filepath.Abs(spec.PreparationRoot)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve I/O profile preparation root: %w", err)
	}
	root := filepath.Join(preparationRoot, ".io-profile-overlay")
	if err := os.Mkdir(root, 0o700); err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("create I/O profile overlay directory: %w", err)
	}
	moduleReplacement := filepath.Join(root, "sqlite")
	replacement, err := copySQLiteModule(moduleSource, moduleReplacement, replacementContents)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	workingDirectory, err := filepath.Abs(spec.WorkingDir)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve target working directory: %w", err)
	}
	moduleFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.mod"))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("read target module file: %w", err)
	}
	if bytes.Contains(moduleFile, []byte("replace modernc.org/sqlite")) {
		return target.Spec{}, BuildOverlay{}, errors.New("target module already replaces modernc.org/sqlite")
	}
	moduleFile = append(moduleFile, []byte("\nreplace modernc.org/sqlite => "+moduleReplacement+"\n")...)
	modFilePath := filepath.Join(root, "gomad.mod")
	if err := writeExclusive(modFilePath, moduleFile); err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	sumFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.sum"))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("read target module sums: %w", err)
	}
	if err := writeExclusive(filepath.Join(root, "gomad.sum"), sumFile); err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	spec.BuildModFile = modFilePath
	if profile.Name == TemporalActivityAPIBatchSecurity {
		spec.BuildOverlay, err = prepareTemporalSQLiteOverlay(root, workingDirectory)
		if err != nil {
			return target.Spec{}, BuildOverlay{}, err
		}
	}
	return spec, BuildOverlay{
		Path: modFilePath, Source: source, Replacement: replacement, SourceSHA256: sqliteSourceSHA256,
		ReplacementSHA256: digestBytes(replacementContents),
	}, nil
}

func prepareTemporalSQLiteOverlay(root, workingDirectory string) (string, error) {
	source := filepath.Join(workingDirectory, "tests", "testcore", "functional_test_base.go")
	contents, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read Temporal SQLite profile source: %w", err)
	}
	const original = `		// Use file-based SQLite for shared clusters to support parallel test access.
		return *persistencetests.GetSQLiteFileTestClusterOption()`
	const replacement = `		// Use named in-memory SQLite for Gomad; schema contents still come from the explicit read-only mount.
		options := *persistencetests.GetSQLiteMemoryTestClusterOption()
		options.SchemaDir = "schema/sqlite/v3"
		return options`
	if bytes.Count(contents, []byte(original)) != 1 {
		return "", errors.New("Temporal SQLite profile rewrite anchor mismatch")
	}
	replacementContents := bytes.Replace(contents, []byte(original), []byte(replacement), 1)
	replacementPath := filepath.Join(root, "temporal", "tests", "testcore", "functional_test_base.go")
	if err := os.MkdirAll(filepath.Dir(replacementPath), 0o700); err != nil {
		return "", fmt.Errorf("create Temporal SQLite overlay directory: %w", err)
	}
	if err := writeExclusive(replacementPath, replacementContents); err != nil {
		return "", err
	}
	overlayPath := filepath.Join(root, "overlay.json")
	overlayJSON := []byte(fmt.Sprintf("{\"Replace\":{%q:%q}}", source, replacementPath))
	if err := writeExclusive(overlayPath, overlayJSON); err != nil {
		return "", err
	}
	return overlayPath, nil
}

func copySQLiteModule(source, destination string, replacement []byte) (string, error) {
	if err := os.Mkdir(destination, 0o700); err != nil {
		return "", fmt.Errorf("create SQLite overlay module: %w", err)
	}
	files := 0
	bytesCopied := int64(0)
	replacementPath := filepath.Join(destination, "lib", "sqlite_darwin_arm64.go")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("invalid SQLite module path")
		}
		targetPath := filepath.Join(destination, relative)
		if entry.IsDir() {
			if relative == "." {
				return nil
			}
			return os.Mkdir(targetPath, 0o700)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("SQLite module contains a non-regular file")
		}
		files++
		bytesCopied += info.Size()
		if files > maximumModuleFiles || bytesCopied > maximumModuleBytes {
			return errors.New("SQLite module exceeds overlay bounds")
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if targetPath == replacementPath {
			contents = replacement
		}
		return writeExclusive(targetPath, contents)
	})
	if err != nil {
		return "", fmt.Errorf("copy SQLite overlay module: %w", err)
	}
	return replacementPath, nil
}

func rewriteSQLiteSource(source []byte) ([]byte, error) {
	rewrites := [][2][]byte{
		{
			[]byte("var _ unsafe.Pointer\n"),
			[]byte("var _ unsafe.Pointer\n\n//go:linkname gomadSQLiteEnabled internal/gomadio.Enabled\nfunc gomadSQLiteEnabled() bool\n\n//go:linkname gomadSQLiteRandomness internal/gomadio.SQLiteRandomness\nfunc gomadSQLiteRandomness(address uintptr, size int32) int32\n\n//go:linkname gomadSQLiteCurrentTime internal/gomadio.SQLiteCurrentTime\nfunc gomadSQLiteCurrentTime() int64\n"),
		},
		{
			[]byte("func _unixRandomness(tls *libc.TLS, NotUsed uintptr, nBuf int32, zBuf uintptr) (r int32) {\n"),
			[]byte("func _unixRandomness(tls *libc.TLS, NotUsed uintptr, nBuf int32, zBuf uintptr) (r int32) {\n\tif gomadSQLiteEnabled() {\n\t\treturn gomadSQLiteRandomness(zBuf, nBuf)\n\t}\n"),
		},
		{
			[]byte("func _unixCurrentTimeInt64(tls *libc.TLS, NotUsed uintptr, piNow uintptr) (r int32) {\n"),
			[]byte("func _unixCurrentTimeInt64(tls *libc.TLS, NotUsed uintptr, piNow uintptr) (r int32) {\n\tif gomadSQLiteEnabled() {\n\t\t*(*Tsqlite3_int64)(unsafe.Pointer(piNow)) = gomadSQLiteCurrentTime()\n\t\treturn SQLITE_OK\n\t}\n"),
		},
	}
	replacement := append([]byte(nil), source...)
	for _, rewrite := range rewrites {
		if bytes.Count(replacement, rewrite[0]) != 1 {
			return nil, errors.New("pinned SQLite rewrite anchor mismatch")
		}
		replacement = bytes.Replace(replacement, rewrite[0], rewrite[1], 1)
	}
	return replacement, nil
}

func writeExclusive(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return fmt.Errorf("create I/O profile overlay file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		return errors.Join(fmt.Errorf("write I/O profile overlay file: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close I/O profile overlay file: %w", err)
	}
	return nil
}

func digestBytes(contents []byte) string {
	digest := sha256.Sum256(contents)
	return "sha256:" + hex.EncodeToString(digest[:])
}
