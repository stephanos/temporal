package authoring

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

// WorkingDirectoriesFile is the checked-in table, below the pack authoring
// root, that names the module directory each request is discovered and
// qualified in. Requests name a module and package, not a directory, so
// compatibility-pack qualification and refresh both read this table.
const WorkingDirectoriesFile = "working-directories.json"

const WorkingDirectoriesSchema = "gomad3.compatibility-pack-working-directories/v1"

const maximumWorkingDirectoriesBytes = 1 << 20

type workingDirectoryTable struct {
	Schema   string                  `json:"schema"`
	Requests []workingDirectoryEntry `json:"requests"`
}

type workingDirectoryEntry struct {
	Request string `json:"request"`
	// Directory is slash-separated and relative to the pack authoring root.
	Directory string `json:"directory"`
}

// InputError reports authoring input the command cannot act on as given,
// such as a request without a working directory.
type InputError struct{ Err error }

func (err *InputError) Error() string { return err.Err.Error() }
func (err *InputError) Unwrap() error { return err.Err }

// IsInputError reports whether err is invalid authoring input.
func IsInputError(err error) bool {
	var input *InputError
	return errors.As(err, &input)
}

// LoadWorkingDirectories reads the table below root and returns each
// request's absolute working directory. Every request under root must have
// exactly one entry and every entry must name a request; anything else is
// invalid input.
func LoadWorkingDirectories(root string) (map[string]string, error) {
	requests, err := loadRequests(root, false)
	if err != nil {
		return nil, err
	}
	return workingDirectoriesFor(root, requests)
}

func workingDirectoriesFor(root string, requests map[string]Request) (map[string]string, error) {
	tablePath := filepath.Join(root, WorkingDirectoriesFile)
	info, err := os.Lstat(tablePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumWorkingDirectoriesBytes {
		return nil, &InputError{Err: fmt.Errorf("compatibility-pack working-directory table %s is not a bounded regular file", WorkingDirectoriesFile)}
	}
	contents, err := os.ReadFile(tablePath)
	if err != nil {
		return nil, fmt.Errorf("read compatibility-pack working-directory table: %w", err)
	}
	var table workingDirectoryTable
	if err := canonicaljson.StrictDecode(contents, &table); err != nil {
		return nil, &InputError{Err: fmt.Errorf("decode compatibility-pack working-directory table: %w", err)}
	}
	if table.Schema != WorkingDirectoriesSchema {
		return nil, &InputError{Err: errors.New("compatibility-pack working-directory table schema is unsupported")}
	}
	directories := make(map[string]string, len(table.Requests))
	for index, entry := range table.Requests {
		if index > 0 && table.Requests[index-1].Request >= entry.Request {
			return nil, &InputError{Err: errors.New("compatibility-pack working-directory table is not sorted by request with unique entries")}
		}
		if _, found := requests[entry.Request]; !found {
			return nil, &InputError{Err: fmt.Errorf("compatibility-pack working-directory table names %s, which has no request", entry.Request)}
		}
		directory := entry.Directory
		if directory == "" || path.IsAbs(directory) || path.Clean(directory) != directory || strings.ContainsAny(directory, "\\\x00") {
			return nil, &InputError{Err: fmt.Errorf("compatibility-pack working directory %q of %s must be a clean slash path relative to the pack root", directory, entry.Request)}
		}
		directories[entry.Request] = filepath.Join(root, filepath.FromSlash(directory))
	}
	var unmapped []string
	for id := range requests {
		if _, found := directories[id]; !found {
			unmapped = append(unmapped, id)
		}
	}
	if len(unmapped) != 0 {
		slices.Sort(unmapped)
		return nil, &InputError{Err: fmt.Errorf("compatibility-pack requests have no working directory in %s: %s", WorkingDirectoriesFile, strings.Join(unmapped, ", "))}
	}
	return directories, nil
}
