package measure

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

type retainedRun struct {
	store, root *os.Root
	lease       *hostfs.Lock
	path, name  string
}

func (scope *runScope) beginRetention(ctx context.Context) (_ *retainedRun, retErr error) {
	// Samples write their parent's authenticated index paths and share its lease.
	if scope.name == "namespace-sample" {
		return nil, nil
	}
	variable := ""
	for _, declared := range measurement.Inputs(scope.name) {
		if declared.Required && strings.HasSuffix(declared.Variable, "_ROOT") {
			variable = declared.Variable
			break
		}
	}
	if variable == "" && scope.name == "matched-suite" {
		variable = "GOMAD_WASM_PROTOTYPE_ROOT"
	}
	base := scope.options.Inputs[variable]
	if base == "" {
		return nil, nil
	}
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) {
		return nil, fmt.Errorf("%w: measurement output root must be an absolute clean non-root path", ErrRefused)
	}
	if insideRetentionStore(base) {
		return nil, fmt.Errorf("%w: measurement output root %s is inside a retained measurement store; pass the originally requested root", ErrRefused, base)
	}
	baseRoot, err := retentionBase(base, true)
	if err != nil {
		return nil, err
	}
	if insideRetentionStore(baseRoot.Name()) {
		return nil, errors.Join(fmt.Errorf("%w: measurement output root %s is inside a retained measurement store", ErrRefused, base), baseRoot.Close())
	}
	store, openErr := retentionDirectory(filepath.Join(baseRoot.Name(), ".gomad-measure", scope.name), true)
	err = errors.Join(openErr, baseRoot.Close())
	if err != nil {
		if store != nil {
			err = errors.Join(err, store.Close())
		}
		return nil, err
	}
	run := &retainedRun{store: store, name: "run-" + rand.Text()}
	run.path = filepath.Join(store.Name(), run.name)
	initializing, err := retentionStoreLock(ctx, store)
	if err != nil {
		return nil, errors.Join(err, store.Close())
	}
	defer func() { retErr = errors.Join(retErr, initializing.Release()) }()
	directoryName := ".stage-" + run.name
	created, owned := false, false
	defer func() {
		if retErr != nil {
			if owned {
				retErr = errors.Join(retErr, run.finish(ctx, false))
			} else {
				if created {
					retErr = errors.Join(retErr, store.RemoveAll(directoryName))
				}
				retErr = errors.Join(retErr, run.close())
			}
		}
	}()
	if err := store.Mkdir(directoryName, 0700); err != nil {
		return nil, err
	}
	created = true
	run.root, err = store.OpenRoot(directoryName)
	if err != nil {
		return nil, err
	}
	file, _, err := hostfs.CreateRootExclusive(run.root, ".lease", 0600)
	if err != nil {
		return nil, err
	}
	run.lease, err = hostfs.TryDescriptor(file)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if err := retentionWrite(run.root, ".pin", []byte(run.path)); err != nil {
		return nil, err
	}
	if scope.options.Keep {
		if err := retentionWrite(run.root, ".keep", nil); err != nil {
			return nil, err
		}
	}
	if _, err := store.Lstat(run.name); err == nil {
		return nil, errors.New("measurement run path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := store.Rename(directoryName, run.name); err != nil {
		return nil, err
	}
	owned = true
	if err := initializing.Release(); err != nil {
		return nil, err
	}
	if err := run.pinInputs(scope.options.Inputs, variable); err != nil {
		return nil, fmt.Errorf("%w: retain measurement inputs: %w", ErrRefused, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope.outputVariable, scope.requestedRoot = variable, base
	scope.options.Inputs = maps.Clone(scope.options.Inputs)
	scope.options.Inputs[variable] = run.path
	logf(ctx, "staged output=%s", run.path)
	return run, nil
}

// insideRetentionStore reports whether path, or its nearest existing ancestor's
// canonical form, lies in a .gomad-measure store, where new runs would nest
// inside immutable results.
func insideRetentionStore(path string) bool {
	for existing := path; ; existing = filepath.Dir(existing) {
		if slices.Contains(strings.Split(existing, string(filepath.Separator)), ".gomad-measure") {
			return true
		}
		if canonical, err := filepath.EvalSymlinks(existing); err == nil {
			return slices.Contains(strings.Split(canonical, string(filepath.Separator)), ".gomad-measure") ||
				slices.Contains(strings.Split(path, string(filepath.Separator)), ".gomad-measure")
		}
		if existing == filepath.Dir(existing) {
			return false
		}
	}
}

func retentionBase(path string, create bool) (*os.Root, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) && create {
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("measurement output root is not a real directory")
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	root, err := retentionDirectory(canonical, false)
	if err != nil {
		return nil, err
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.Join(errors.New("measurement output root changed"), err, root.Close())
	}
	return root, nil
}

func retentionDirectory(path string, create bool) (_ *os.Root, retErr error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("absolute clean retention directory required")
	}
	root, err := os.OpenRoot(string(filepath.Separator))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, root.Close())
		}
	}()
	for _, component := range strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator)) {
		info, err := root.Lstat(component)
		if errors.Is(err, os.ErrNotExist) && create {
			if err := root.Mkdir(component, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, err
			}
			info, err = root.Lstat(component)
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, errors.New("retention directory is not a real directory")
		}
		next, err := root.OpenRoot(component)
		if err != nil {
			return nil, err
		}
		opened, statErr := next.Stat(".")
		if statErr != nil || !os.SameFile(info, opened) {
			return nil, errors.Join(errors.New("retention directory changed"), statErr, next.Close())
		}
		if err := root.Close(); err != nil {
			return nil, errors.Join(err, next.Close())
		}
		root = next
	}
	return root, nil
}

func retentionWrite(root *os.Root, name string, data []byte) (retErr error) {
	stage := ".stage-" + rand.Text()
	file, _, err := hostfs.CreateRootExclusive(root, stage, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err := root.Remove(stage); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, err)
		}
	}()
	_, writeErr := file.Write(data)
	err = errors.Join(writeErr, file.Sync(), file.Close())
	if err != nil {
		return err
	}
	if err := root.Rename(stage, name); err != nil {
		return err
	}
	return hostfs.SyncRoot(root)
}

func managedRunPath(path string) bool {
	return strings.HasPrefix(filepath.Base(path), "run-") && filepath.Base(filepath.Dir(filepath.Dir(path))) == ".gomad-measure"
}

func (run *retainedRun) pinInputs(inputs map[string]string, output string) error {
	for key, value := range inputs {
		if key == output || !filepath.IsAbs(value) {
			continue
		}
		for path := filepath.Clean(value); path != string(filepath.Separator); path = filepath.Dir(path) {
			if !managedRunPath(path) {
				continue
			}
			if err := run.pinInput(path); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (run *retainedRun) pinInput(path string) (retErr error) {
	root, err := retentionBase(path, false)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	file, _, err := hostfs.OpenRootWritable(root, ".lease")
	if err != nil {
		return err
	}
	lease, err := hostfs.TrySharedDescriptor(file)
	if err != nil {
		return errors.Join(err, file.Close())
	}
	defer func() { retErr = errors.Join(retErr, lease.Release()) }()
	if err := validateRetentionPin(root, root.Name()); err != nil {
		return err
	}
	// Collection renames runs aside under their exclusive lease; reject a run moved before we leased it.
	current, err := os.Lstat(root.Name())
	if err != nil {
		return err
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(current, opened) {
		return errors.Join(errors.New("retained measurement input was collected"), err)
	}
	if err := run.root.Mkdir(".pins", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	digest := sha256.Sum256([]byte(root.Name()))
	name := filepath.Join(".pins", hex.EncodeToString(digest[:]))
	if _, err := run.root.Lstat(name); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := linkRetentionPin(root, run.root, name); err != nil {
		return err
	}
	return hostfs.SyncRoot(run.root)
}

func validateRetentionPin(root *os.Root, path string) error {
	data, err := retentionPin(root, ".pin")
	if err != nil {
		return err
	}
	if data != path {
		return errors.New("retention run identity changed")
	}
	return nil
}

func retentionPin(root *os.Root, name string) (_ string, retErr error) {
	info, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 8192 {
		return "", errors.New("invalid retention pin")
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", errors.Join(errors.New("retention pin changed"), err)
	}
	data, err := io.ReadAll(io.LimitReader(file, 8193))
	if err != nil || len(data) > 8192 {
		return "", errors.Join(errors.New("invalid retention pin size"), err)
	}
	return string(data), nil
}

func (run *retainedRun) close() error {
	var errs []error
	if run.lease != nil {
		errs = append(errs, run.lease.Release())
	}
	if run.root != nil {
		errs = append(errs, run.root.Close())
	}
	if run.store != nil {
		errs = append(errs, run.store.Close())
	}
	return errors.Join(errs...)
}

func (run *retainedRun) finish(ctx context.Context, success bool) (retErr error) {
	defer func() { retErr = errors.Join(retErr, run.close()) }()
	if success {
		if err := run.publish(ctx); err != nil {
			return err
		}
	}
	if err := run.lease.Release(); err != nil {
		return err
	}
	reclaimCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), retentionLockTimeout)
	defer cancel()
	// Publication (or failure) is already final; a reclaim failure leaves extra runs for the next pass.
	if err := reclaimRetention(reclaimCtx, run.store.Name()); err != nil {
		logf(ctx, "retention reclaim deferred: %v", err)
	}
	return nil
}

// retentionLockTimeout bounds waiting for the store lock, which is only held for
// pointer reads/writes and renames; tree syncs and deletions run outside it.
const retentionLockTimeout = time.Minute

func (run *retainedRun) publish(ctx context.Context) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The run's exclusive lease keeps collectors away while the tree is synced.
	if err := syncRetentionTree(ctx, run.root); err != nil {
		return err
	}
	if err := retentionWrite(run.root, ".complete", nil); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			// An unpublished attempt is a failed diagnostic, not a superseded success.
			if err := run.root.Remove(".complete"); err != nil && !errors.Is(err, os.ErrNotExist) {
				retErr = errors.Join(retErr, err)
			}
		}
	}()
	lockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), retentionLockTimeout)
	defer cancel()
	lock, err := retentionStoreLock(lockCtx, run.store)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := latestRetentionRun(run.store); err != nil {
		return err
	}
	encoded, err := json.Marshal(run.name)
	if err != nil {
		return err
	}
	return retentionWrite(run.store, "latest.json", append(encoded, '\n'))
}

func syncRetentionTree(ctx context.Context, root *os.Root) error {
	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return errors.New("nonregular measurement output")
		}
		file, err := root.Open(path)
		if err != nil {
			return err
		}
		return errors.Join(file.Sync(), file.Close())
	})
}

func retentionStoreLock(ctx context.Context, root *os.Root) (*hostfs.Lock, error) {
	file, _, err := hostfs.CreateRootExclusive(root, ".lock", 0600)
	if errors.Is(err, os.ErrExist) {
		file, _, err = hostfs.OpenRootWritable(root, ".lock")
	}
	if err != nil {
		return nil, err
	}
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		lock, err := hostfs.TryDescriptor(file)
		if err == nil {
			return lock, nil
		}
		if !errors.Is(err, hostfs.ErrContended) {
			return nil, errors.Join(err, file.Close())
		}
		select {
		case <-ctx.Done():
			return nil, errors.Join(ctx.Err(), file.Close())
		case <-tick.C:
		}
	}
}

func latestRetentionRun(root *os.Root) (string, error) {
	file, _, err := hostfs.OpenRoot(root, "latest.json")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 1024))
	err = errors.Join(readErr, file.Close())
	var name string
	if err == nil {
		err = json.Unmarshal(data, &name)
	}
	if err != nil || filepath.Base(name) != name || !strings.HasPrefix(name, "run-") {
		return "", errors.Join(errors.New("invalid latest measurement pointer"), err)
	}
	return name, nil
}

func reclaimRetention(ctx context.Context, path string) error {
	pending := []string{path}
	for len(pending) > 0 {
		path, pending = pending[0], pending[1:]
		store, err := retentionDirectory(path, false)
		if err != nil {
			return err
		}
		lock, err := retentionStoreLock(ctx, store)
		if err != nil {
			return errors.Join(err, store.Close())
		}
		trash, released, collectErr := collectRetention(store)
		err = errors.Join(collectErr, lock.Release())
		// Trash names are claimed by atomic rename, so deletion needs no store lock.
		for _, name := range trash {
			err = errors.Join(err, store.RemoveAll(name))
		}
		if err = errors.Join(err, store.Close()); err != nil {
			return err
		}
		pending = append(pending, released...)
	}
	return nil
}

func retentionTrashName() string { return ".trash-" + rand.Text() }

func collectRetention(store *os.Root) (trash, released []string, retErr error) {
	defer func() {
		if retErr == nil && len(trash) > 0 {
			retErr = hostfs.SyncRoot(store)
		}
	}()
	latest, err := latestRetentionRun(store)
	if err != nil {
		return nil, nil, err
	}
	directory, err := store.Open(".")
	if err != nil {
		return nil, nil, err
	}
	entries, readErr := directory.ReadDir(-1)
	if err := errors.Join(readErr, directory.Close()); err != nil {
		return nil, nil, err
	}
	modified := make(map[string]time.Time, len(entries))
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil {
			modified[entry.Name()] = info.ModTime()
		}
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		if order := modified[b.Name()].Compare(modified[a.Name()]); order != 0 {
			return order
		}
		return strings.Compare(a.Name(), b.Name())
	})
	failedKept := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() && (strings.HasPrefix(name, ".stage-run-") || strings.HasPrefix(name, ".trash-")) {
			// Claim abandoned staging or partially deleted trash for this pass.
			claimed := retentionTrashName()
			if err := store.Rename(name, claimed); err != nil {
				return trash, released, err
			}
			trash = append(trash, claimed)
			continue
		}
		if !entry.IsDir() && strings.HasPrefix(name, ".stage-") {
			if err := store.Remove(name); err != nil {
				return trash, released, err
			}
			continue
		}
		if !entry.IsDir() || !strings.HasPrefix(name, "run-") || name == latest {
			continue
		}
		claimed, inputs, err := collectRetentionRun(store, name, &failedKept)
		if err != nil {
			return trash, released, err
		}
		if claimed != "" {
			trash = append(trash, claimed)
			released = append(released, inputs...)
		}
	}
	return trash, released, nil
}

func collectRetentionRun(store *os.Root, name string, failedKept *bool) (_ string, _ []string, retErr error) {
	root, err := store.OpenRoot(name)
	if err != nil {
		return "", nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	path := filepath.Join(store.Name(), name)
	if err := validateRetentionPin(root, path); err != nil {
		return "", nil, nil
	}
	file, _, err := hostfs.OpenRootWritable(root, ".lease")
	if err != nil {
		return "", nil, err
	}
	lease, err := hostfs.TryDescriptor(file)
	if err != nil {
		closeErr := file.Close()
		if errors.Is(err, hostfs.ErrContended) {
			return "", nil, closeErr
		}
		return "", nil, errors.Join(err, closeErr)
	}
	defer func() { retErr = errors.Join(retErr, lease.Release()) }()
	if _, err := root.Lstat(".keep"); err == nil {
		return "", nil, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, err
	}
	info, err := root.Lstat(".pin")
	if err != nil {
		return "", nil, err
	}
	links, known := hostfs.LinkCount(info)
	if !known || links != 1 {
		return "", nil, nil
	}
	if _, err := root.Lstat(".complete"); errors.Is(err, os.ErrNotExist) {
		if !*failedKept {
			*failedKept = true
			return "", nil, nil
		}
	} else if err != nil {
		return "", nil, err
	}
	dependencies, err := retentionDependencies(root)
	if err != nil {
		return "", nil, err
	}
	// Move the run out of its authenticated path while holding its lease; deletion follows outside the store lock.
	claimed := retentionTrashName()
	if err := store.Rename(name, claimed); err != nil {
		return "", nil, err
	}
	released := []string{store.Name()}
	for _, path := range dependencies {
		released = append(released, filepath.Dir(path))
	}
	return claimed, released, nil
}

func retentionDependencies(root *os.Root) (_ []string, retErr error) {
	directory, err := root.Open(".pins")
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, directory.Close()) }()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		path, err := retentionPin(root, filepath.Join(".pins", entry.Name()))
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || !managedRunPath(path) {
			return nil, errors.New("invalid retained measurement input path")
		}
		paths = append(paths, path)
	}
	return paths, nil
}
