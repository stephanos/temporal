package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
)

const campaignPlanMountDirectory = "mounts"

func canonicalCampaignPlanMounts(mappings []deterministicio.Mapping) ([]deterministicio.Mapping, []deterministicio.Mapping) {
	canonical := append([]deterministicio.Mapping(nil), mappings...)
	sort.Slice(canonical, func(left, right int) bool { return canonical[left].Target < canonical[right].Target })
	portable := make([]deterministicio.Mapping, len(canonical))
	for index, mapping := range canonical {
		portable[index] = deterministicio.Mapping{Source: campaignPlanMountSource(index), Target: mapping.Target}
	}
	return canonical, portable
}

func campaignPlanMountSource(index int) string {
	return path.Join(campaignPlanMountDirectory, fmt.Sprintf("%06d", index))
}

func campaignPlanMountValue(index int, target string) string {
	return campaignPlanMountSource(index) + "=" + strings.TrimPrefix(target, "/")
}

func campaignPlanRuntimeMountValues(mappings []deterministicio.Mapping) []string {
	values := make([]string, len(mappings))
	for index, mapping := range mappings {
		values[index] = mapping.Source + "=" + strings.TrimPrefix(mapping.Target, "/")
	}
	return values
}

func captureCampaignPlanMounts(mappings []deterministicio.Mapping, limits deterministicio.Limits) (*campaignPlanMountIdentity, *deterministicio.CapturedInputs, error) {
	if len(mappings) == 0 {
		return nil, nil, nil
	}
	captured, err := deterministicio.CaptureReadOnlyMountInputs(mappings, limits)
	if err != nil {
		return nil, nil, err
	}
	manifest := captured.Manifest
	return &campaignPlanMountIdentity{
		Schema: manifest.Schema, SHA256: manifest.SHA256, Bytes: evidence.Uint64String(manifest.Bytes), Entries: evidence.Uint64String(manifest.Entries), TotalBytes: evidence.Uint64String(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: manifest.Limits,
	}, &captured, nil
}

func materializeCampaignPlanMounts(bundle string, captured *deterministicio.CapturedInputs) error {
	if captured == nil {
		return nil
	}
	mappings, _, snapshot, err := deterministicio.DecodeCapturedInputs(captured.Manifest, captured.Descriptor, func(name string, maximum uint64) ([]byte, error) {
		data, found := captured.Payloads[name]
		if !found || uint64(len(data)) > maximum {
			return nil, os.ErrNotExist
		}
		return append([]byte(nil), data...), nil
	})
	if err != nil {
		return err
	}
	mountRoot := filepath.Join(bundle, campaignPlanMountDirectory)
	if err := os.Mkdir(mountRoot, 0o700); err != nil {
		return err
	}
	type directoryMode struct {
		path string
		mode os.FileMode
	}
	directories := make([]directoryMode, 0, len(snapshot.Entries))
	rootSeen := make([]bool, len(mappings))
	for _, entry := range snapshot.Entries {
		if entry.Kind != deterministicio.KindDirectory {
			continue
		}
		destination, mappingIndex, err := campaignPlanMountDestination(mountRoot, mappings, entry.Path)
		if err != nil {
			return err
		}
		if entry.Path == mappings[mappingIndex].Target {
			rootSeen[mappingIndex] = true
		}
		if err := os.MkdirAll(destination, 0o700); err != nil {
			return fmt.Errorf("create captured mount directory %q: %w", entry.Path, err)
		}
		directories = append(directories, directoryMode{path: destination, mode: entry.Mode.Perm()})
	}
	for index, seen := range rootSeen {
		if !seen {
			return fmt.Errorf("captured mount %q is missing its root directory", mappings[index].Target)
		}
	}
	for _, entry := range snapshot.Entries {
		if entry.Kind != deterministicio.KindFile {
			continue
		}
		destination, _, err := campaignPlanMountDestination(mountRoot, mappings, entry.Path)
		if err != nil {
			return err
		}
		if err := writeCampaignPlanMountFile(destination, entry.Data, entry.Mode.Perm()); err != nil {
			return fmt.Errorf("write captured mount file %q: %w", entry.Path, err)
		}
	}
	sort.Slice(directories, func(left, right int) bool {
		leftDepth := strings.Count(filepath.Clean(directories[left].path), string(filepath.Separator))
		rightDepth := strings.Count(filepath.Clean(directories[right].path), string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return directories[left].path > directories[right].path
	})
	for _, directory := range directories {
		if err := chmodAndSyncCampaignPlanDirectory(directory.path, directory.mode); err != nil {
			return err
		}
	}
	return syncDirectory(mountRoot)
}

func campaignPlanMountDestination(root string, mappings []deterministicio.Mapping, targetPath string) (string, int, error) {
	for index, mapping := range mappings {
		if targetPath != mapping.Target && !strings.HasPrefix(targetPath, mapping.Target+"/") {
			continue
		}
		relative := strings.TrimPrefix(strings.TrimPrefix(targetPath, mapping.Target), "/")
		destination := filepath.Join(root, fmt.Sprintf("%06d", index), filepath.FromSlash(relative))
		contained, err := filepath.Rel(root, destination)
		if err != nil || contained == ".." || filepath.IsAbs(contained) || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
			return "", 0, errors.Join(fmt.Errorf("captured mount path %q escapes its bundle", targetPath), err)
		}
		return destination, index, nil
	}
	return "", 0, fmt.Errorf("captured mount path %q is not mapped", targetPath)
}

func writeCampaignPlanMountFile(path string, contents []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(contents)
	if writeErr == nil && written != len(contents) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		return errors.Join(writeErr, file.Close())
	}
	if err := file.Chmod(mode); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	return file.Close()
}

func chmodAndSyncCampaignPlanDirectory(path string, mode os.FileMode) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Chmod(mode); err != nil {
		return errors.Join(err, directory.Close())
	}
	if err := directory.Sync(); err != nil {
		return errors.Join(err, directory.Close())
	}
	return directory.Close()
}

func validateCampaignPlanBundleInventory(bundle string, mountCount int) error {
	entries, err := os.ReadDir(bundle)
	wantEntries := 1
	if mountCount != 0 {
		wantEntries++
	}
	if err != nil || len(entries) != wantEntries {
		return errors.Join(errors.New("campaign plan bundle inventory is invalid"), err)
	}
	foundTarget := false
	foundMounts := false
	for _, entry := range entries {
		switch entry.Name() {
		case campaignPlanTargetFile:
			foundTarget = entry.Type().IsRegular()
		case campaignPlanMountDirectory:
			foundMounts = entry.IsDir() && entry.Type()&os.ModeSymlink == 0
		default:
			return errors.New("campaign plan bundle inventory is invalid")
		}
	}
	if !foundTarget || foundMounts != (mountCount != 0) {
		return errors.New("campaign plan bundle inventory is invalid")
	}
	if mountCount == 0 {
		return nil
	}
	mountRoot := filepath.Join(bundle, campaignPlanMountDirectory)
	info, err := os.Lstat(mountRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return errors.Join(errors.New("campaign plan mount bundle is not a private directory"), err)
	}
	mounts, err := os.ReadDir(mountRoot)
	if err != nil || len(mounts) != mountCount {
		return errors.Join(errors.New("campaign plan mount bundle inventory is invalid"), err)
	}
	for index, mount := range mounts {
		if mount.Name() != fmt.Sprintf("%06d", index) || !mount.IsDir() || mount.Type()&os.ModeSymlink != 0 {
			return errors.New("campaign plan mount bundle inventory is invalid")
		}
	}
	return nil
}
