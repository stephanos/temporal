package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/umpire/artifact"
)

const (
	exitSuccess     = 0
	exitFailure     = 1
	exitUsage       = 2
	maximumSetFiles = 7
)

type usageError struct {
	cause error
}

func (e usageError) Error() string {
	return e.cause.Error()
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, _ io.Writer, stderr io.Writer) int {
	err := dispatch(arguments)
	if err == nil {
		return exitSuccess
	}
	if _, writeErr := fmt.Fprintf(stderr, "umpire-artifact: %v\n", err); writeErr != nil {
		return exitFailure
	}
	var invalid usageError
	if errors.As(err, &invalid) {
		return exitUsage
	}
	return exitFailure
}

func dispatch(arguments []string) error {
	if len(arguments) == 0 {
		return invalidUsage("expected check or check-set subcommand")
	}
	switch arguments[0] {
	case "check":
		return checkArtifact(arguments[1:])
	case "check-set":
		return checkArtifactSet(arguments[1:])
	default:
		return invalidUsage("expected check or check-set subcommand")
	}
}

func checkArtifact(arguments []string) error {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	family := flags.String("family", "", "exact Artifact formatVersion")
	path := flags.String("artifact", "", "Artifact JSON path")
	if err := flags.Parse(arguments); err != nil {
		return usageError{cause: err}
	}
	if flags.NArg() != 0 {
		return invalidUsage("unexpected positional arguments")
	}
	if *family == "" {
		return invalidUsage("--family is required")
	}
	if !supportedFamily(*family) {
		return invalidUsage("unsupported --family %q", *family)
	}
	if *path == "" {
		return invalidUsage("--artifact is required")
	}
	encoded, err := readRegularFile(*path)
	if err != nil {
		return fmt.Errorf("read Artifact: %w", err)
	}
	return admitArtifact(*family, encoded)
}

func checkArtifactSet(arguments []string) error {
	flags := flag.NewFlagSet("check-set", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("set", "", "complete Artifact set directory")
	if err := flags.Parse(arguments); err != nil {
		return usageError{cause: err}
	}
	if flags.NArg() != 0 {
		return invalidUsage("unexpected positional arguments")
	}
	if *path == "" {
		return invalidUsage("--set is required")
	}
	files, err := readSetFiles(*path)
	if err != nil {
		return fmt.Errorf("read Artifact set: %w", err)
	}
	_, err = artifact.AdmitSetFiles(files)
	return err
}

func supportedFamily(family string) bool {
	switch family {
	case "umpire-experiment/v2",
		"umpire-runtime-configuration/v2",
		"umpire-experiment-run/v2",
		"umpire-raw-evidence/v2",
		"umpire-evidence/v2",
		"umpire-result/v2":
		return true
	default:
		return false
	}
}

func admitArtifact(family string, encoded []byte) error {
	switch family {
	case "umpire-experiment/v2":
		_, err := artifact.DecodeExperimentV2(encoded)
		return err
	case "umpire-runtime-configuration/v2":
		_, err := artifact.DecodeRuntimeConfigurationV2(encoded)
		return err
	case "umpire-experiment-run/v2":
		_, err := artifact.DecodeExperimentRunV2(encoded)
		return err
	case "umpire-raw-evidence/v2":
		_, err := artifact.DecodeRawEvidenceV2(encoded)
		return err
	case "umpire-evidence/v2":
		_, err := artifact.DecodeEvidenceV2(encoded)
		return err
	case "umpire-result/v2":
		_, err := artifact.DecodeResultV2(encoded)
		return err
	default:
		return invalidUsage("unsupported --family %q", family)
	}
}

func readSetFiles(root string) (map[string][]byte, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("set path is not a regular directory")
	}
	files := make(map[string][]byte)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative != "." && relative != "artifacts" {
				return fmt.Errorf("unexpected directory %q", relative)
			}
			return nil
		}
		if !supportedSetFilePath(relative) {
			return fmt.Errorf("unexpected file %q", relative)
		}
		if len(files) >= maximumSetFiles {
			return fmt.Errorf("artifact set has more than %d files", maximumSetFiles)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q is not a regular file", relative)
		}
		encoded, err := readRegularFile(path)
		if err != nil {
			return fmt.Errorf("%q: %w", relative, err)
		}
		files[relative] = encoded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func supportedSetFilePath(path string) bool {
	switch path {
	case "manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"artifacts/experiment-run.json",
		"artifacts/raw-evidence.json",
		"artifacts/evidence.json",
		"artifacts/result.json":
		return true
	default:
		return false
	}
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("is not a regular file")
	}
	if info.Size() > artifact.MaximumDocumentBytes {
		return nil, fmt.Errorf("exceeds the %d-byte limit", artifact.MaximumDocumentBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, errors.New("changed while it was opened")
	}
	encoded, err := io.ReadAll(io.LimitReader(file, artifact.MaximumDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(encoded) > artifact.MaximumDocumentBytes {
		return nil, fmt.Errorf("exceeds the %d-byte limit", artifact.MaximumDocumentBytes)
	}
	return encoded, nil
}

func invalidUsage(format string, arguments ...any) error {
	return usageError{cause: fmt.Errorf(format, arguments...)}
}
