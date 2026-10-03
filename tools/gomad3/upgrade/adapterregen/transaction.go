package adapterregen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"golang.org/x/mod/modfile"
)

// The transaction keeps its lock and journal under the checkout's private
// toolchain directory, which version control ignores.
const (
	stateDirectory   = ".toolchain/adapter-regeneration"
	journalDirectory = "journal"
	journalManifest  = "manifest.json"
	journalSchema    = "gomad3.adapter-regeneration-journal/v1"
)

// skippedDirectories are checkout directories the staged copy omits: built
// toolchains, binaries, and caches that no generator reads.
var skippedDirectories = map[string]bool{".toolchain": true, ".bin": true}

// DefaultGenerators are the gomad3-local steps of make generate. The
// qualification manifest step reads the repository outside the Gomad module
// and does not consume adapter pins, so the staged copy omits it.
var DefaultGenerators = [][]string{
	{"{go}", "run", "./cmd/gomadtool", "version-generate"},
	{"{go}", "run", "./cmd/gomadtool", "protocol-generate"},
	{"{go}", "run", "./cmd/gomadtool", "boundary-generate"},
	{"{go}", "run", "./cmd/gomadtool", "boundary-generate", "-check-compiler-tests"},
	{"{go}", "run", "./cmd/gomadtool", "compatibility-pack", "generate", "--root={root}"},
}

// DefaultVerifiers check the staged set before publication: it builds, the
// commands build, the adapter tests compile, generated files agree with their inputs, and the
// staged adapter accepts the candidate module under every pin. The copy omits
// the repository around the module, so the pack check does not require a
// go.mod in the working directories the table maps outside it.
var DefaultVerifiers = [][]string{
	{"{go}", "build", "./cmd/..."},
	{"{go}", "vet", "-tags", "test_dep", "./deterministicio/...", "./toolchain/version/..."},
	{"{go}", "run", "./cmd/gomadtool", "version-generate", "-check"},
	{"{go}", "run", "./cmd/gomadtool", "boundary-generate", "-check"},
	{"{go}", "run", "./cmd/gomadtool", "compatibility-pack", "check", "--root={root}", "--staged-copy"},
	{"{go}", "run", "./cmd/gomadtool", "adapter-regenerate", "--verify", "--module={module}", "--module-dir={candidate}", "--go={go}"},
}

type journalEntry struct {
	Path string `json:"path"`
	// Previous and Next are content digests; "" means the file is absent.
	Previous string `json:"previous"`
	Next     string `json:"next"`
	Mode     uint32 `json:"mode"`
	Blob     string `json:"blob,omitempty"`
}

type journal struct {
	Schema  string         `json:"schema"`
	Entries []journalEntry `json:"entries"`
}

func apply(ctx context.Context, spec Spec, regeneration deterministicio.AdapterRegeneration, candidate string) (publication, error) {
	state := filepath.Join(spec.Root, filepath.FromSlash(stateDirectory))
	if err := os.MkdirAll(state, 0o700); err != nil {
		return publication{}, err
	}
	lock, err := hostfs.Try(filepath.Join(state, "lock"))
	if err != nil {
		if errors.Is(err, hostfs.ErrContended) {
			return publication{}, fmt.Errorf("another adapter regeneration holds %s: %w", filepath.Join(stateDirectory, "lock"), err)
		}
		return publication{}, err
	}
	defer lock.Release()
	if spec.StageOnly {
		if pending, err := publicationPending(spec.Root); err != nil || pending {
			return publication{}, errors.Join(err, &BlockedError{Err: errors.New("an interrupted adapter publication is pending; run with --recover to complete it")})
		}
	} else if completed, err := recoverPublication(spec.Root); err != nil {
		return publication{}, err
	} else if completed {
		return publication{}, &BlockedError{Err: errors.New("completed an interrupted adapter publication; run the dry run again before applying")}
	}
	work, err := os.MkdirTemp("", "gomad3-adapter-stage-")
	if err != nil {
		return publication{}, err
	}
	defer removeWritable(work)
	stage := filepath.Join(work, "root")
	read, err := copyCheckout(spec.Root, stage)
	if err != nil {
		return publication{}, err
	}
	if err := stageRegeneration(ctx, spec, regeneration, stage, candidate); err != nil {
		return publication{}, &BlockedError{Err: fmt.Errorf("staged regeneration failed; nothing was published: %w", err)}
	}
	if spec.afterStage != nil {
		if err := spec.afterStage(); err != nil {
			return publication{}, err
		}
	}
	entries, err := stagedChanges(stage, read)
	if err != nil {
		return publication{}, err
	}
	staged, err := summarizeStaged(spec.Root, stage, entries)
	if err != nil {
		return publication{}, err
	}
	if spec.StageOnly {
		return publication{staged: staged}, nil
	}
	if err := revalidate(spec.Root, read); err != nil {
		return publication{}, &BlockedError{Err: fmt.Errorf("the checkout changed after staging; nothing was published: %w", err)}
	}
	if err := writeJournal(state, stage, entries); err != nil {
		return publication{}, err
	}
	if err := completeJournal(spec.Root, spec.beforeApplyFile); err != nil {
		return publication{}, err
	}
	result := publication{staged: staged, published: make([]string, len(entries))}
	for index, entry := range entries {
		result.published[index] = entry.Path
	}
	// The publication is complete; a failed scan for leftover references
	// must not report it as failed.
	scan := spec.residualScan
	if scan == nil {
		scan = residualReferences
	}
	if result.residual, err = scan(spec.Root, regeneration); err != nil {
		result.warnings = append(result.warnings, fmt.Sprintf("published, but scanning for references to %s@%s failed: %v", regeneration.Module, regeneration.Previous.Version, err))
		result.residual = nil
	}
	return result, nil
}

// publication is what an apply staged and, unless it only staged, published.
type publication struct {
	staged    []StagedFile
	published []string
	residual  []string
	warnings  []string
}

// summarizeStaged lists every staged change with the module-file diffs a
// person reviews before approving: tidy can move requirements beyond the
// adapted module.
func summarizeStaged(root, stage string, entries []journalEntry) ([]StagedFile, error) {
	staged := make([]StagedFile, len(entries))
	for index, entry := range entries {
		file := StagedFile{Path: entry.Path, Change: "changed"}
		switch {
		case entry.Previous == "":
			file.Change = "added"
		case entry.Next == "":
			file.Change = "removed"
		}
		if name := path.Base(entry.Path); name == "go.mod" || name == "go.sum" {
			previous, err := readOptional(filepath.Join(root, filepath.FromSlash(entry.Path)))
			if err != nil {
				return nil, err
			}
			next, err := readOptional(filepath.Join(stage, filepath.FromSlash(entry.Path)))
			if err != nil {
				return nil, err
			}
			file.Diff = unifiedDiff(entry.Path, previous, next)
		}
		staged[index] = file
	}
	return staged, nil
}

func readOptional(path string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return contents, err
}

// Recover completes an interrupted publication under the transaction lock.
func Recover(root string) error {
	state := filepath.Join(root, filepath.FromSlash(stateDirectory))
	if err := os.MkdirAll(state, 0o700); err != nil {
		return err
	}
	lock, err := hostfs.Try(filepath.Join(state, "lock"))
	if err != nil {
		return err
	}
	defer lock.Release()
	_, err = recoverPublication(root)
	return err
}

func publicationPending(root string) (bool, error) {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(stateDirectory), journalDirectory, journalManifest))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// recoverPublication completes a committed journal, or discards an
// uncommitted one, which no checkout file has seen yet. A committed journal
// whose files no longer hold their previous or next contents is left for a
// person.
func recoverPublication(root string) (bool, error) {
	state := filepath.Join(root, filepath.FromSlash(stateDirectory))
	pending, err := filepath.Glob(filepath.Join(state, "pending-*"))
	if err != nil {
		return false, err
	}
	retired, err := filepath.Glob(filepath.Join(state, "done-*"))
	if err != nil {
		return false, err
	}
	for _, directory := range append(pending, retired...) {
		if err := os.RemoveAll(directory); err != nil {
			return false, err
		}
	}
	committed, err := publicationPending(root)
	if err != nil {
		return false, err
	}
	if !committed {
		// A journal directory without its manifest is the remainder of a
		// completed publication whose removal was interrupted: the manifest
		// is written before the commit rename, so no uncommitted or
		// incomplete journal lacks it.
		if err := os.RemoveAll(filepath.Join(state, journalDirectory)); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, completeJournal(root, nil)
}

func completeJournal(root string, beforeApplyFile func(int) error) error {
	directory := filepath.Join(root, filepath.FromSlash(stateDirectory), journalDirectory)
	contents, err := os.ReadFile(filepath.Join(directory, journalManifest))
	if err != nil {
		return err
	}
	var recorded journal
	if err := json.Unmarshal(contents, &recorded); err != nil || recorded.Schema != journalSchema {
		return fmt.Errorf("adapter publication journal is unreadable: %v", err)
	}
	for _, entry := range recorded.Entries {
		current, err := fileDigest(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			return err
		}
		if current != entry.Previous && current != entry.Next {
			return &BlockedError{Err: fmt.Errorf("adapter publication journal %s cannot complete: %s changed after the publication started; restore it or remove the journal by hand", filepath.Join(stateDirectory, journalDirectory), entry.Path)}
		}
	}
	for index, entry := range recorded.Entries {
		if beforeApplyFile != nil {
			if err := beforeApplyFile(index); err != nil {
				return err
			}
		}
		path := filepath.Join(root, filepath.FromSlash(entry.Path))
		// A file already holding its next contents was published before an
		// interruption; its blob may be gone with a partially removed journal.
		if current, err := fileDigest(path); err != nil {
			return err
		} else if current == entry.Next {
			continue
		}
		if entry.Next == "" {
			if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, entry.Blob))
		if err != nil {
			return err
		}
		if digest(contents) != entry.Next {
			return fmt.Errorf("adapter publication journal blob for %s is corrupt", entry.Path)
		}
		if err := hostfs.Replace(path, contents, fs.FileMode(entry.Mode)); err != nil {
			return fmt.Errorf("publish %s: %w", entry.Path, err)
		}
	}
	return retireJournal(filepath.Dir(directory), directory)
}

// retireJournal removes a completed journal by first renaming it out of the
// way, so an interrupted removal never leaves a partial journal/ behind; the
// next run sweeps the retired directory.
func retireJournal(state, directory string) error {
	retired, err := os.MkdirTemp(state, "done-")
	if err != nil {
		return err
	}
	if err := os.Remove(retired); err != nil {
		return err
	}
	if err := os.Rename(directory, retired); err != nil {
		return fmt.Errorf("retire adapter publication journal: %w", err)
	}
	if err := syncDirectory(state); err != nil {
		return err
	}
	return os.RemoveAll(retired)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

// writeJournal records every staged change and its contents in a pending
// directory, then commits it with one rename; before the rename no checkout
// file has changed, after it the next run can always finish publication.
func writeJournal(state, stage string, entries []journalEntry) error {
	pending, err := os.MkdirTemp(state, "pending-")
	if err != nil {
		return err
	}
	for index := range entries {
		if entries[index].Next == "" {
			continue
		}
		entries[index].Blob = "blob-" + strconv.Itoa(index)
		contents, err := os.ReadFile(filepath.Join(stage, filepath.FromSlash(entries[index].Path)))
		if err != nil {
			return err
		}
		if err := hostfs.Replace(filepath.Join(pending, entries[index].Blob), contents, 0o600); err != nil {
			return err
		}
	}
	encoded, err := json.MarshalIndent(journal{Schema: journalSchema, Entries: entries}, "", "  ")
	if err != nil {
		return err
	}
	if err := hostfs.Replace(filepath.Join(pending, journalManifest), encoded, 0o600); err != nil {
		return err
	}
	if err := os.Rename(pending, filepath.Join(state, journalDirectory)); err != nil {
		return fmt.Errorf("commit adapter publication journal: %w", err)
	}
	return syncDirectory(state)
}

// copyCheckout copies root into stage and returns the digest of every file it
// read, keyed by slash path.
func copyCheckout(root, stage string) (map[string]string, error) {
	read := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(stage, relative)
		if entry.IsDir() {
			if skippedDirectories[relative] {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("checkout file %s is not a regular file", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		read[filepath.ToSlash(relative)] = digest(contents)
		return os.WriteFile(target, contents, info.Mode().Perm())
	})
	if err != nil {
		return nil, fmt.Errorf("stage the checkout: %w", err)
	}
	// Generators share the checkout's build cache; nothing reads it as input.
	if cache := filepath.Join(root, ".toolchain", "generator-cache"); isDirectory(cache) {
		if err := os.MkdirAll(filepath.Join(stage, ".toolchain"), 0o700); err != nil {
			return nil, err
		}
		if err := os.Symlink(cache, filepath.Join(stage, ".toolchain", "generator-cache")); err != nil {
			return nil, err
		}
	}
	return read, nil
}

func stageRegeneration(ctx context.Context, spec Spec, regeneration deterministicio.AdapterRegeneration, stage, candidate string) error {
	edits, err := regeneration.SourceEdits(stage)
	if err != nil {
		return err
	}
	for relative, contents := range edits {
		if err := writeStaged(stage, relative, contents); err != nil {
			return err
		}
	}
	if err := editDescriptor(stage, regeneration); err != nil {
		return err
	}
	fixtures, err := editFixtures(stage, regeneration)
	if err != nil {
		return err
	}
	tidy := spec.Tidy
	if tidy == nil {
		tidy = []string{"{go}", "mod", "tidy"}
	}
	for _, fixture := range fixtures {
		if err := runStaged(ctx, spec, filepath.Join(stage, filepath.FromSlash(fixture)), stage, candidate, tidy, "GOFLAGS=-mod=mod"); err != nil {
			return fmt.Errorf("tidy test fixture %s: %w", fixture, err)
		}
	}
	generators, verifiers := spec.Generators, spec.Verifiers
	if generators == nil {
		generators = DefaultGenerators
	}
	if verifiers == nil {
		verifiers = DefaultVerifiers
	}
	for _, command := range generators {
		if err := runStaged(ctx, spec, stage, stage, candidate, command); err != nil {
			return fmt.Errorf("generate: %w", err)
		}
	}
	for _, command := range verifiers {
		if err := runStaged(ctx, spec, stage, stage, candidate, command); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
	}
	return nil
}

func runStaged(ctx context.Context, spec Spec, directory, stage, candidate string, command []string, extra ...string) error {
	expanded := make([]string, len(command))
	replacer := strings.NewReplacer("{root}", stage, "{go}", spec.GoCommand, "{module}", spec.Module, "{candidate}", candidate)
	for index, argument := range command {
		expanded[index] = replacer.Replace(argument)
	}
	environment := goEnvironment(spec.Environment, append([]string{"GOCACHE=" + filepath.Join(stage, ".toolchain", "generator-cache")}, extra...)...)
	if !isDirectory(filepath.Join(stage, ".toolchain", "generator-cache")) {
		environment = goEnvironment(spec.Environment, extra...)
	}
	result, err := hostexec.Run(ctx, hostexec.Request{
		Command: expanded, Dir: directory, Env: environment,
		Timeout: commandTimeout, TerminateGrace: time.Second, OutputLimit: commandOutputLimit,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", strings.Join(command, " "), err)
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || result.WatchdogTimeout {
		output := strings.TrimSpace(string(result.Stdout.RawBytes) + string(result.Stderr.RawBytes))
		return fmt.Errorf("%s exited %d: %s", strings.Join(command, " "), result.ExitCode, output)
	}
	return nil
}

// editDescriptor moves the module's version descriptor entry to the
// candidate identity. The entry is matched exactly once, like a rewrite
// anchor, so a reformatted descriptor fails instead of being guessed at.
func editDescriptor(stage string, regeneration deterministicio.AdapterRegeneration) error {
	path := filepath.Join(stage, "toolchain", "version", "version.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	entry := func(version, sum string) []byte {
		return []byte(fmt.Sprintf("\"module\": %q,\n      \"version\": %q,\n      \"sum\": %q", regeneration.Module, version, sum))
	}
	previous := entry(regeneration.Previous.Version, regeneration.Previous.Sum)
	if count := bytes.Count(contents, previous); count != 1 {
		return fmt.Errorf("version descriptor entry for %s@%s occurs %d times, want exactly once", regeneration.Module, regeneration.Previous.Version, count)
	}
	return writeStaged(stage, "toolchain/version/version.json", bytes.Replace(contents, previous, entry(regeneration.Proposed.Version, regeneration.Proposed.Sum), 1))
}

// editFixtures moves every test fixture module that requires the previous
// version to the candidate and returns the fixtures it changed. The go.sum
// lines for the module are replaced; tidy then completes the graph.
func editFixtures(stage string, regeneration deterministicio.AdapterRegeneration) ([]string, error) {
	var changed []string
	err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Type()&fs.ModeSymlink == 0 && skippedDirectories[entry.Name()] {
			return filepath.SkipDir
		}
		if entry.Name() != "go.mod" || !strings.Contains(filepath.ToSlash(path), "/testdata/") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parsed, err := modfile.Parse(path, contents, nil)
		if err != nil {
			return err
		}
		requires := false
		for _, requirement := range parsed.Require {
			requires = requires || requirement.Mod.Path == regeneration.Module && requirement.Mod.Version == regeneration.Previous.Version
		}
		if !requires {
			return nil
		}
		if err := parsed.AddRequire(regeneration.Module, regeneration.Proposed.Version); err != nil {
			return err
		}
		edited, err := parsed.Format()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(stage, filepath.Dir(path))
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := writeStaged(stage, relative+"/go.mod", edited); err != nil {
			return err
		}
		sums, err := os.ReadFile(filepath.Join(filepath.Dir(path), "go.sum"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		var kept []string
		for _, line := range strings.SplitAfter(string(sums), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 3 && fields[0] == regeneration.Module && (fields[1] == regeneration.Previous.Version || fields[1] == regeneration.Previous.Version+"/go.mod") {
				continue
			}
			kept = append(kept, line)
		}
		kept = append(kept, regeneration.Module+" "+regeneration.Proposed.Version+" "+regeneration.Proposed.Sum+"\n")
		if regeneration.Proposed.GoModSum != "" {
			kept = append(kept, regeneration.Module+" "+regeneration.Proposed.Version+"/go.mod "+regeneration.Proposed.GoModSum+"\n")
		}
		if err := writeStaged(stage, relative+"/go.sum", []byte(strings.Join(kept, ""))); err != nil {
			return err
		}
		changed = append(changed, relative)
		return nil
	})
	sort.Strings(changed)
	return changed, err
}

func writeStaged(stage, relative string, contents []byte) error {
	path := filepath.Join(stage, filepath.FromSlash(relative))
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, contents, mode)
}

// stagedChanges compares the staged copy with the files read from the
// checkout and returns every addition, change, and removal.
func stagedChanges(stage string, read map[string]string) ([]journalEntry, error) {
	var entries []journalEntry
	seen := map[string]bool{}
	err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirectories[relative] {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("staged file %s is not a regular file", relative)
		}
		relative = filepath.ToSlash(relative)
		seen[relative] = true
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if next := digest(contents); next != read[relative] {
			entries = append(entries, journalEntry{Path: relative, Previous: read[relative], Next: next, Mode: uint32(info.Mode().Perm())})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for relative, previous := range read {
		if !seen[relative] {
			entries = append(entries, journalEntry{Path: relative, Previous: previous})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// revalidate checks that every checkout file the stage read still holds the
// contents it had, and that no file appeared in its place.
func revalidate(root string, read map[string]string) error {
	current := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirectories[relative] {
				return filepath.SkipDir
			}
			return nil
		}
		relative = filepath.ToSlash(relative)
		current[relative] = true
		want, found := read[relative]
		if !found {
			return fmt.Errorf("%s appeared", relative)
		}
		got, err := fileDigest(path)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("%s changed", relative)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for relative := range read {
		if !current[relative] {
			return fmt.Errorf("%s was removed", relative)
		}
	}
	return nil
}

// residualReferences lists published files that still name the previous
// module identity, for a person to review.
func residualReferences(root string, regeneration deterministicio.AdapterRegeneration) ([]string, error) {
	needles := [][]byte{
		[]byte(regeneration.Module + " " + regeneration.Previous.Version),
		[]byte(regeneration.Module + "@" + regeneration.Previous.Version),
		[]byte(regeneration.Previous.Sum),
	}
	var residual []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirectories[relative] || filepath.Base(relative) == "packs" || filepath.Base(relative) == "requests" {
				return filepath.SkipDir
			}
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for number, line := range bytes.Split(contents, []byte("\n")) {
			for _, needle := range needles {
				if bytes.Contains(line, needle) && !versionContinues(line, needle) {
					residual = append(residual, filepath.ToSlash(relative)+":"+strconv.Itoa(number+1))
					break
				}
			}
		}
		return nil
	})
	return residual, err
}

func versionContinues(line, needle []byte) bool {
	index := bytes.Index(line, needle)
	end := index + len(needle)
	return end < len(line) && bytes.ContainsRune([]byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.+-"), rune(line[end]))
}

func fileDigest(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return digest(contents), nil
}

func digest(contents []byte) string {
	sum := sha256.Sum256(contents)
	return fmt.Sprintf("sha256:%x", sum)
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
