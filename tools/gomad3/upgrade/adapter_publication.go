package upgrade

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

type publicationContent struct {
	Present bool   `json:"present"`
	Bytes   []byte `json:"bytes,omitempty"`
}

type adapterPublicationFile struct {
	Path string             `json:"path"`
	Old  publicationContent `json:"old"`
	New  publicationContent `json:"new"`
}

type adapterPublication struct {
	Schema string                   `json:"schema"`
	Files  []adapterPublicationFile `json:"files"`
}

var adapterRegenerationInputTrees = []string{
	filepath.Join("tools", "gomad3"),
	filepath.Join("tools", "gomad3integration", "qualification"),
	"tests",
}

func adapterPublicationMarker(root string) string {
	return filepath.Join(root, "tools", "gomad3", ".toolchain", "adapter-regenerate", "transaction.json")
}

func withAdapterPublicationLock(root string, operation func() error) (retErr error) {
	lockPath := filepath.Join(root, "tools", "gomad3", ".toolchain", "adapter-regenerate.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return err
	}
	lock, err := hostfs.Try(lockPath)
	if err != nil {
		return fmt.Errorf("lock adapter regeneration: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	return operation()
}

func readPublicationFile(root, relative string) (publicationContent, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return publicationContent{}, fmt.Errorf("invalid publication path %q", relative)
	}
	path := filepath.Join(root, relative)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return publicationContent{}, nil
	}
	if err != nil {
		return publicationContent{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return publicationContent{}, fmt.Errorf("publication path is not a bounded regular file: %s", relative)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return publicationContent{}, err
	}
	return publicationContent{Present: true, Bytes: contents}, nil
}

func samePublicationContent(a, b publicationContent) bool {
	return a.Present == b.Present && bytes.Equal(a.Bytes, b.Bytes)
}

func writePublicationFile(root, relative string, content publicationContent) error {
	path := filepath.Join(root, relative)
	if !content.Present {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return syncPublicationDirectory(filepath.Dir(path))
	}
	mode := os.FileMode(0o644)
	if strings.Contains(relative, string(filepath.Separator)+"requests"+string(filepath.Separator)) {
		mode = 0o600
	}
	return hostfs.Replace(path, content.Bytes, mode)
}

func syncPublicationDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func recordAdapterPublication(root string, files []adapterPublicationFile) error {
	marker := adapterPublicationMarker(root)
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err != nil {
		return err
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		return errors.New("adapter regeneration transaction already exists")
	}
	encoded, err := json.Marshal(adapterPublication{Schema: "gomad3.adapter-regeneration-transaction/v1", Files: files})
	if err != nil {
		return err
	}
	return hostfs.Replace(marker, encoded, 0o600)
}

func clearAdapterPublication(root string) error {
	marker := adapterPublicationMarker(root)
	if err := os.Remove(marker); err != nil {
		return err
	}
	return syncPublicationDirectory(filepath.Dir(marker))
}

func recoverAdapterPublication(root string) error {
	marker := adapterPublicationMarker(root)
	info, err := os.Lstat(marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return errors.New("adapter regeneration marker is invalid")
	}
	encoded, err := os.ReadFile(marker)
	if err != nil {
		return err
	}
	var pending adapterPublication
	if err := json.Unmarshal(encoded, &pending); err != nil {
		return err
	}
	if pending.Schema != "gomad3.adapter-regeneration-transaction/v1" || len(pending.Files) == 0 {
		return errors.New("adapter regeneration marker has invalid schema or files")
	}
	seen := make(map[string]bool, len(pending.Files))
	for _, file := range pending.Files {
		if seen[file.Path] {
			return errors.New("adapter regeneration marker has duplicate paths")
		}
		seen[file.Path] = true
		current, err := readPublicationFile(root, file.Path)
		if err != nil {
			return err
		}
		if !samePublicationContent(current, file.Old) && !samePublicationContent(current, file.New) {
			return fmt.Errorf("adapter regeneration interrupted checkout changed externally: %s", file.Path)
		}
	}
	for _, file := range pending.Files {
		if err := writePublicationFile(root, file.Path, file.Old); err != nil {
			return fmt.Errorf("restore interrupted adapter regeneration %s: %w", file.Path, err)
		}
	}
	return clearAdapterPublication(root)
}

func publishAdapterFiles(root string, files []adapterPublicationFile) error {
	return publishAdapterFilesWithSnapshot(root, files, nil, nil)
}

func validateAdapterPublicationSnapshot(root string, files []adapterPublicationFile, snapshot map[string]publicationContent, inputTrees []string) error {
	for path, old := range snapshot {
		current, err := readPublicationFile(root, path)
		if err != nil {
			return err
		}
		if !samePublicationContent(current, old) {
			return fmt.Errorf("checkout changed since staging: %s", path)
		}
	}
	for _, file := range files {
		current, err := readPublicationFile(root, file.Path)
		if err != nil {
			return err
		}
		if !samePublicationContent(current, file.Old) {
			return fmt.Errorf("checkout changed since staging: %s", file.Path)
		}
	}
	for _, tree := range inputTrees {
		current, err := snapshotAdapterStageTree(filepath.Join(root, tree))
		if err != nil {
			return err
		}
		for relative, contents := range current {
			path := filepath.Join(tree, relative)
			old, found := snapshot[path]
			if !found || !samePublicationContent(contents, old) {
				return fmt.Errorf("checkout changed since staging: %s", path)
			}
		}
	}
	return nil
}

func publishAdapterFilesWithSnapshot(root string, files []adapterPublicationFile, snapshot map[string]publicationContent, inputTrees []string) error {
	if len(files) == 0 {
		return errors.New("adapter regeneration has no changed files")
	}
	slices.SortFunc(files, func(a, b adapterPublicationFile) int { return strings.Compare(a.Path, b.Path) })
	return withAdapterPublicationLock(root, func() error {
		if err := recoverAdapterPublication(root); err != nil {
			return err
		}
		if err := validateAdapterPublicationSnapshot(root, files, snapshot, inputTrees); err != nil {
			return err
		}
		if err := recordAdapterPublication(root, files); err != nil {
			return err
		}
		for _, file := range files {
			if err := writePublicationFile(root, file.Path, file.New); err != nil {
				return errors.Join(fmt.Errorf("publish adapter regeneration %s: %w", file.Path, err), recoverAdapterPublication(root))
			}
		}
		return clearAdapterPublication(root)
	})
}
