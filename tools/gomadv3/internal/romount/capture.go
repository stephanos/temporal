package romount

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

type Kind uint8

const (
	KindFile Kind = iota + 1
	KindDirectory
)

var ErrCapacity = errors.New("read-only mount capacity exceeded")

var ErrReplayDivergence = errors.New("read-only mount replay lookup was not captured")

type Limits struct {
	PathBytes        uint64
	Requests         uint64
	Files            uint64
	DirectoryEntries uint64
	SingleFileBytes  uint64
	TotalBytes       uint64
}

func DefaultLimits() Limits {
	return Limits{
		PathBytes: 4096, Requests: 100_000, Files: 10_000,
		DirectoryEntries: 100_000, SingleFileBytes: 16 << 20, TotalBytes: 64 << 20,
	}
}

type Child struct {
	Name string
	Mode os.FileMode
	Kind Kind
}

type Entry struct {
	Path     string
	Mode     os.FileMode
	Kind     Kind
	Data     []byte
	Children []Child
}

type Snapshot struct {
	Entries    []Entry
	NotExist   []string
	Requests   uint64
	TotalBytes uint64
}

type preparedMapping struct {
	mapping Mapping
	root    *os.Root
}

type Broker struct {
	mu               sync.Mutex
	mappings         []preparedMapping
	limits           Limits
	entries          map[string]Entry
	notExist         map[string]struct{}
	requests         uint64
	totalBytes       uint64
	directoryEntries uint64
	closed           bool
	replay           bool
}

func Prepare(mappings []Mapping, limits Limits) (*Broker, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	broker := &Broker{limits: limits, entries: make(map[string]Entry), notExist: make(map[string]struct{}), mappings: make([]preparedMapping, 0, len(mappings))}
	for _, mapping := range mappings {
		root, err := os.OpenRoot(mapping.Source)
		if err != nil {
			_ = broker.Close()
			return nil, fmt.Errorf("open read-only mount source %q: %w", mapping.Source, err)
		}
		broker.mappings = append(broker.mappings, preparedMapping{mapping: mapping, root: root})
	}
	return broker, nil
}

func PrepareReplay(mappings []Mapping, limits Limits, snapshot Snapshot) (*Broker, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	broker := &Broker{
		limits: limits, entries: make(map[string]Entry, len(snapshot.Entries)), notExist: make(map[string]struct{}, len(snapshot.NotExist)), mappings: make([]preparedMapping, len(mappings)), replay: true,
	}
	for index, mapping := range mappings {
		broker.mappings[index] = preparedMapping{mapping: mapping}
	}
	for _, entry := range snapshot.Entries {
		if _, found := broker.entries[entry.Path]; found || !withinTargets(entry.Path, mappingTargets(mappings)) {
			return nil, fmt.Errorf("invalid replay read-only mount path %q", entry.Path)
		}
		broker.entries[entry.Path] = cloneEntry(entry)
		broker.totalBytes += uint64(len(entry.Data))
		broker.directoryEntries += uint64(len(entry.Children))
	}
	for _, name := range snapshot.NotExist {
		if _, found := broker.notExist[name]; found || !withinTargets(name, mappingTargets(mappings)) {
			return nil, fmt.Errorf("invalid replay missing read-only mount path %q", name)
		}
		if _, found := broker.entries[name]; found {
			return nil, fmt.Errorf("conflicting replay read-only mount path %q", name)
		}
		broker.notExist[name] = struct{}{}
	}
	if broker.totalBytes != snapshot.TotalBytes || uint64(len(broker.entries)) > limits.Files || broker.directoryEntries > limits.DirectoryEntries {
		return nil, errors.New("invalid replay read-only mount snapshot totals")
	}
	return broker, nil
}

func validateLimits(limits Limits) error {
	if limits.PathBytes == 0 || limits.Requests == 0 || limits.Files == 0 || limits.DirectoryEntries == 0 || limits.SingleFileBytes == 0 || limits.TotalBytes == 0 {
		return errors.New("read-only mount limits must be positive")
	}
	if limits.SingleFileBytes > limits.TotalBytes {
		return errors.New("read-only mount single-file limit exceeds aggregate limit")
	}
	return nil
}

func (broker *Broker) Lookup(name string) (Entry, error) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return Entry{}, errors.New("read-only mount broker is closed")
	}
	if broker.requests == broker.limits.Requests {
		return Entry{}, ErrCapacity
	}
	broker.requests++
	normalized, err := normalizeLookup(name, broker.limits.PathBytes)
	if err != nil {
		return Entry{}, err
	}
	if entry, found := broker.entries[normalized]; found {
		return cloneEntry(entry), nil
	}
	if _, found := broker.notExist[normalized]; found {
		return Entry{}, os.ErrNotExist
	}
	mapping, relative, found := broker.resolve(normalized)
	if !found {
		return Entry{}, os.ErrNotExist
	}
	if broker.replay {
		return Entry{}, errors.Join(os.ErrNotExist, ErrReplayDivergence)
	}
	entry, err := broker.capture(mapping, normalized, relative)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			broker.notExist[normalized] = struct{}{}
		}
		return Entry{}, err
	}
	broker.entries[normalized] = entry
	return cloneEntry(entry), nil
}

func normalizeLookup(name string, pathLimit uint64) (string, error) {
	if name == "" || name[0] != '/' || strings.IndexByte(name, 0) >= 0 || uint64(len(name)) > pathLimit {
		return "", fmt.Errorf("invalid read-only mount path %q", name)
	}
	for _, component := range strings.Split(name, "/") {
		if component == ".." {
			return "", fmt.Errorf("invalid read-only mount path %q", name)
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "/" {
		return "", fmt.Errorf("invalid read-only mount path %q", name)
	}
	return cleaned, nil
}

func (broker *Broker) resolve(name string) (*preparedMapping, string, bool) {
	for index := range broker.mappings {
		mapping := &broker.mappings[index]
		if name == mapping.mapping.Target {
			return mapping, ".", true
		}
		if strings.HasPrefix(name, mapping.mapping.Target+"/") {
			return mapping, strings.TrimPrefix(name, mapping.mapping.Target+"/"), true
		}
	}
	return nil, "", false
}

func (broker *Broker) capture(mapping *preparedMapping, targetPath, relative string) (Entry, error) {
	info, err := mapping.root.Lstat(relative)
	if err != nil {
		return Entry{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Entry{}, fmt.Errorf("read-only mount path %q is a symbolic link", targetPath)
	}
	if err := rejectSymlinkParents(mapping.root, relative); err != nil {
		return Entry{}, err
	}
	switch {
	case info.Mode().IsRegular():
		return broker.captureFile(mapping.root, targetPath, relative, info)
	case info.IsDir():
		return broker.captureDirectory(mapping.root, targetPath, relative, info)
	default:
		return Entry{}, fmt.Errorf("read-only mount path %q has unsupported mode %v", targetPath, info.Mode())
	}
}

func rejectSymlinkParents(root *os.Root, relative string) error {
	components := strings.Split(relative, "/")
	for index := 1; index < len(components); index++ {
		info, err := root.Lstat(strings.Join(components[:index], "/"))
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("read-only mount path traverses a symbolic link")
		}
	}
	return nil
}

func (broker *Broker) captureFile(root *os.Root, targetPath, relative string, before os.FileInfo) (Entry, error) {
	if hardLinked(before) {
		return Entry{}, fmt.Errorf("read-only mount path %q is hard linked", targetPath)
	}
	if uint64(before.Size()) > broker.limits.SingleFileBytes || broker.totalBytes > broker.limits.TotalBytes-uint64(before.Size()) || uint64(len(broker.entries)) == broker.limits.Files {
		return Entry{}, ErrCapacity
	}
	file, err := root.Open(relative)
	if err != nil {
		return Entry{}, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, int64(broker.limits.SingleFileBytes)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return Entry{}, errors.Join(readErr, closeErr)
	}
	if uint64(len(data)) > broker.limits.SingleFileBytes {
		return Entry{}, ErrCapacity
	}
	after, err := root.Lstat(relative)
	if err != nil {
		return Entry{}, err
	}
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || int64(len(data)) != before.Size() {
		return Entry{}, fmt.Errorf("read-only mount path %q changed during capture", targetPath)
	}
	broker.totalBytes += uint64(len(data))
	return Entry{Path: targetPath, Mode: before.Mode().Perm(), Kind: KindFile, Data: data}, nil
}

func (broker *Broker) captureDirectory(root *os.Root, targetPath, relative string, before os.FileInfo) (Entry, error) {
	file, err := root.Open(relative)
	if err != nil {
		return Entry{}, err
	}
	entries, readErr := file.ReadDir(-1)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return Entry{}, errors.Join(readErr, closeErr)
	}
	if uint64(len(entries)) > broker.limits.DirectoryEntries-broker.directoryEntries || uint64(len(broker.entries)) == broker.limits.Files {
		return Entry{}, ErrCapacity
	}
	children := make([]Child, 0, len(entries))
	for _, directoryEntry := range entries {
		childRelative := directoryEntry.Name()
		if relative != "." {
			childRelative = relative + "/" + childRelative
		}
		info, err := root.Lstat(childRelative)
		if err != nil {
			return Entry{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
			return Entry{}, fmt.Errorf("read-only mount directory %q contains unsupported entry %q", targetPath, directoryEntry.Name())
		}
		kind := KindFile
		if info.IsDir() {
			kind = KindDirectory
		}
		children = append(children, Child{Name: directoryEntry.Name(), Mode: info.Mode().Perm(), Kind: kind})
	}
	sort.Slice(children, func(left, right int) bool { return children[left].Name < children[right].Name })
	after, err := root.Lstat(relative)
	if err != nil {
		return Entry{}, err
	}
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		return Entry{}, fmt.Errorf("read-only mount directory %q changed during capture", targetPath)
	}
	broker.directoryEntries += uint64(len(children))
	return Entry{Path: targetPath, Mode: before.Mode().Perm(), Kind: KindDirectory, Children: children}, nil
}

func (broker *Broker) Captured() Snapshot {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	paths := make([]string, 0, len(broker.entries))
	for path := range broker.entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	missing := make([]string, 0, len(broker.notExist))
	for name := range broker.notExist {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	snapshot := Snapshot{Entries: make([]Entry, 0, len(paths)), NotExist: missing, Requests: broker.requests, TotalBytes: broker.totalBytes}
	for _, path := range paths {
		snapshot.Entries = append(snapshot.Entries, cloneEntry(broker.entries[path]))
	}
	return snapshot
}

func cloneEntry(entry Entry) Entry {
	entry.Data = append([]byte(nil), entry.Data...)
	entry.Children = append([]Child(nil), entry.Children...)
	return entry
}

func (broker *Broker) Close() error {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return nil
	}
	broker.closed = true
	var result error
	for index := range broker.mappings {
		if broker.mappings[index].root != nil {
			result = errors.Join(result, broker.mappings[index].root.Close())
		}
	}
	return result
}
