package romount

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const descriptorPath = "io/mounts.json"

type ArtifactRecord struct {
	Manifest   record.ReadOnlyMounts
	Descriptor []byte
	Payloads   map[string][]byte
}

type PayloadReader func(string, uint64) ([]byte, error)

func EncodeArtifact(mappings []Mapping, limits Limits, snapshot Snapshot) (ArtifactRecord, error) {
	if err := validateLimits(limits); err != nil {
		return ArtifactRecord{}, err
	}
	targets := mappingTargets(mappings)
	entries := append([]Entry(nil), snapshot.Entries...)
	sort.Slice(entries, func(left, right int) bool { return entries[left].Path < entries[right].Path })
	notExist := append([]string(nil), snapshot.NotExist...)
	sort.Strings(notExist)
	descriptor := record.ReadOnlyMountDescriptor{
		Schema: "gomadv3.io-read-only-mounts/v1", Mappings: targets, Limits: recordLimits(limits),
		Requests: record.Uint64String(snapshot.Requests), TotalBytes: record.Uint64String(snapshot.TotalBytes),
		NotExist: notExist,
		Entries:  make([]record.ReadOnlyMountEntry, 0, len(entries)),
	}
	payloads := make(map[string][]byte)
	var totalBytes uint64
	previous := ""
	for index, entry := range entries {
		if index > 0 && entry.Path == previous {
			return ArtifactRecord{}, fmt.Errorf("duplicate captured read-only mount path %q", entry.Path)
		}
		previous = entry.Path
		if !withinTargets(entry.Path, targets) {
			return ArtifactRecord{}, fmt.Errorf("captured read-only mount path %q is outside its mappings", entry.Path)
		}
		encoded := record.ReadOnlyMountEntry{
			Path: entry.Path, Mode: formatEntryMode(entry.Mode), Kind: kindName(entry.Kind),
			Children: make([]record.ReadOnlyMountChild, 0, len(entry.Children)),
		}
		switch entry.Kind {
		case KindFile:
			digest := record.HashBytes(entry.Data)
			payload := "io/mounts/files/" + strings.TrimPrefix(string(digest), "sha256:")
			encoded.Size = record.Uint64String(len(entry.Data))
			encoded.SHA256 = digest
			encoded.Payload = payload
			payloads[payload] = append([]byte(nil), entry.Data...)
			totalBytes += uint64(len(entry.Data))
		case KindDirectory:
			children := append([]Child(nil), entry.Children...)
			sort.Slice(children, func(left, right int) bool { return children[left].Name < children[right].Name })
			for _, child := range children {
				encoded.Children = append(encoded.Children, record.ReadOnlyMountChild{
					Name: child.Name, Mode: formatEntryMode(child.Mode), Kind: kindName(child.Kind),
				})
			}
		default:
			return ArtifactRecord{}, fmt.Errorf("captured read-only mount path %q has invalid kind %d", entry.Path, entry.Kind)
		}
		descriptor.Entries = append(descriptor.Entries, encoded)
	}
	previous = ""
	for index, name := range descriptor.NotExist {
		if index > 0 && name == previous || !withinTargets(name, targets) {
			return ArtifactRecord{}, fmt.Errorf("missing read-only mount paths must be sorted, unique, and mapped")
		}
		previous = name
		for _, entry := range entries {
			if entry.Path == name {
				return ArtifactRecord{}, fmt.Errorf("conflicting captured read-only mount path %q", name)
			}
		}
	}
	if totalBytes != snapshot.TotalBytes {
		return ArtifactRecord{}, fmt.Errorf("captured read-only mount byte count is %d, want %d", totalBytes, snapshot.TotalBytes)
	}
	encoded, err := record.CanonicalJSON(descriptor)
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("encode read-only mount descriptor: %w", err)
	}
	return ArtifactRecord{
		Manifest: record.ReadOnlyMounts{
			Schema: descriptor.Schema, File: descriptorPath, SHA256: record.HashBytes(encoded), Bytes: record.Uint64String(len(encoded)),
			Entries: record.Uint64String(len(descriptor.Entries)), NotExist: record.Uint64String(len(descriptor.NotExist)), TotalBytes: descriptor.TotalBytes, Mappings: targets, Limits: descriptor.Limits,
		},
		Descriptor: encoded, Payloads: payloads,
	}, nil
}

func DecodeArtifact(manifest record.ReadOnlyMounts, descriptorBytes []byte, readPayload PayloadReader) ([]Mapping, Limits, Snapshot, error) {
	if manifest.Schema != "gomadv3.io-read-only-mounts/v1" || manifest.File != descriptorPath {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("invalid read-only mount artifact identity")
	}
	if uint64(len(descriptorBytes)) != uint64(manifest.Bytes) || record.HashBytes(descriptorBytes) != manifest.SHA256 {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount descriptor identity mismatch")
	}
	var descriptor record.ReadOnlyMountDescriptor
	if err := record.StrictDecode(descriptorBytes, &descriptor); err != nil {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("decode read-only mount descriptor: %w", err)
	}
	canonical, err := record.CanonicalJSON(descriptor)
	if err != nil || !bytes.Equal(canonical, descriptorBytes) {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount descriptor is not canonical")
	}
	limits, err := decodeLimits(descriptor.Limits)
	if err != nil {
		return nil, Limits{}, Snapshot{}, err
	}
	if descriptor.Schema != manifest.Schema || !equalTargets(descriptor.Mappings, manifest.Mappings) || descriptor.Limits != manifest.Limits || record.Uint64String(len(descriptor.Entries)) != manifest.Entries || record.Uint64String(len(descriptor.NotExist)) != manifest.NotExist || descriptor.TotalBytes != manifest.TotalBytes {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount descriptor does not match its manifest")
	}
	if !sortedTargets(descriptor.Mappings) {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount mappings must be sorted and unique")
	}
	mappings := make([]Mapping, len(descriptor.Mappings))
	for index, target := range descriptor.Mappings {
		mappings[index] = Mapping{Target: target}
	}
	snapshot := Snapshot{Requests: uint64(descriptor.Requests), TotalBytes: uint64(descriptor.TotalBytes), NotExist: append([]string(nil), descriptor.NotExist...), Entries: make([]Entry, 0, len(descriptor.Entries))}
	previous := ""
	for index, name := range snapshot.NotExist {
		if index > 0 && name <= previous || !withinTargets(name, descriptor.Mappings) {
			return nil, Limits{}, Snapshot{}, fmt.Errorf("missing read-only mount paths must be sorted, unique, and mapped")
		}
		previous = name
	}
	var totalBytes uint64
	previous = ""
	for index, encoded := range descriptor.Entries {
		if index > 0 && encoded.Path <= previous || !withinTargets(encoded.Path, descriptor.Mappings) {
			return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount entries must be sorted, unique, and mapped")
		}
		previous = encoded.Path
		for _, missing := range snapshot.NotExist {
			if missing == encoded.Path {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("conflicting captured read-only mount path %q", encoded.Path)
			}
		}
		mode, err := parseEntryMode(encoded.Mode)
		if err != nil {
			return nil, Limits{}, Snapshot{}, err
		}
		kind, err := parseKind(encoded.Kind)
		if err != nil {
			return nil, Limits{}, Snapshot{}, err
		}
		entry := Entry{Path: encoded.Path, Mode: mode, Kind: kind, Children: make([]Child, 0, len(encoded.Children))}
		switch kind {
		case KindFile:
			if encoded.Payload == "" || encoded.SHA256 == "" || len(encoded.Children) != 0 || uint64(encoded.Size) > limits.SingleFileBytes {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("invalid captured file %q", encoded.Path)
			}
			if readPayload == nil {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("read-only mount payload reader is required")
			}
			data, err := readPayload(encoded.Payload, uint64(encoded.Size))
			if err != nil {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("read captured file %q: %w", encoded.Path, err)
			}
			if uint64(len(data)) != uint64(encoded.Size) || record.HashBytes(data) != encoded.SHA256 {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("captured file %q identity mismatch", encoded.Path)
			}
			entry.Data = data
			totalBytes += uint64(len(data))
		case KindDirectory:
			if encoded.Size != 0 || encoded.SHA256 != "" || encoded.Payload != "" {
				return nil, Limits{}, Snapshot{}, fmt.Errorf("invalid captured directory %q", encoded.Path)
			}
			childPrevious := ""
			for childIndex, encodedChild := range encoded.Children {
				if childIndex > 0 && encodedChild.Name <= childPrevious || encodedChild.Name == "" || path.Base(encodedChild.Name) != encodedChild.Name {
					return nil, Limits{}, Snapshot{}, fmt.Errorf("invalid captured directory child in %q", encoded.Path)
				}
				childPrevious = encodedChild.Name
				childMode, err := parseEntryMode(encodedChild.Mode)
				if err != nil {
					return nil, Limits{}, Snapshot{}, err
				}
				childKind, err := parseKind(encodedChild.Kind)
				if err != nil {
					return nil, Limits{}, Snapshot{}, err
				}
				entry.Children = append(entry.Children, Child{Name: encodedChild.Name, Mode: childMode, Kind: childKind})
			}
		}
		snapshot.Entries = append(snapshot.Entries, entry)
	}
	if totalBytes != snapshot.TotalBytes || totalBytes > limits.TotalBytes || uint64(len(snapshot.Entries)) > limits.Files {
		return nil, Limits{}, Snapshot{}, fmt.Errorf("captured read-only mount limits or totals do not match")
	}
	return mappings, limits, snapshot, nil
}

func mappingTargets(mappings []Mapping) []string {
	targets := make([]string, len(mappings))
	for index, mapping := range mappings {
		targets[index] = mapping.Target
	}
	sort.Strings(targets)
	return targets
}

func recordLimits(limits Limits) record.ReadOnlyMountLimits {
	return record.ReadOnlyMountLimits{
		PathBytes: record.Uint64String(limits.PathBytes), Requests: record.Uint64String(limits.Requests), Files: record.Uint64String(limits.Files),
		DirectoryEntries: record.Uint64String(limits.DirectoryEntries), SingleFileBytes: record.Uint64String(limits.SingleFileBytes), TotalBytes: record.Uint64String(limits.TotalBytes),
	}
}

func decodeLimits(limits record.ReadOnlyMountLimits) (Limits, error) {
	decoded := Limits{
		PathBytes: uint64(limits.PathBytes), Requests: uint64(limits.Requests), Files: uint64(limits.Files), DirectoryEntries: uint64(limits.DirectoryEntries),
		SingleFileBytes: uint64(limits.SingleFileBytes), TotalBytes: uint64(limits.TotalBytes),
	}
	return decoded, validateLimits(decoded)
}

func formatEntryMode(mode os.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

func parseEntryMode(value string) (os.FileMode, error) {
	if len(value) != 4 {
		return 0, fmt.Errorf("invalid captured entry mode %q", value)
	}
	parsed, err := strconv.ParseUint(value, 8, 12)
	if err != nil {
		return 0, fmt.Errorf("invalid captured entry mode %q", value)
	}
	return os.FileMode(parsed), nil
}

func kindName(kind Kind) string {
	if kind == KindFile {
		return "file"
	}
	if kind == KindDirectory {
		return "directory"
	}
	return ""
}

func parseKind(value string) (Kind, error) {
	switch value {
	case "file":
		return KindFile, nil
	case "directory":
		return KindDirectory, nil
	default:
		return 0, fmt.Errorf("invalid captured entry kind %q", value)
	}
}

func withinTargets(name string, targets []string) bool {
	for _, target := range targets {
		if name == target || strings.HasPrefix(name, target+"/") {
			return true
		}
	}
	return false
}

func sortedTargets(targets []string) bool {
	for index, target := range targets {
		if target == "" || target[0] != '/' || path.Clean(target) != target || target == "/" || index > 0 && (target <= targets[index-1] || overlaps(target, targets[index-1])) {
			return false
		}
	}
	return true
}

func equalTargets(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
