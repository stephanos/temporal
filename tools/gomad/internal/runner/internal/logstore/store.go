package logstore

import (
	"errors"
	"fmt"
	"os"

	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

const (
	// LogFile names the record log inside a store directory.
	LogFile = "log"
	// LockFile names the writer lock inside a store directory.
	LockFile = "lock"

	storeMode  = os.ModeDir | 0o700
	memberMode = os.FileMode(0o600)
)

// Options configures a store open.
type Options struct {
	// Kinds maps every record kind the owner accepts to its maximum payload
	// length. Replay rejects any other kind as corruption, and the bound is
	// validated before any payload is allocated.
	Kinds map[Kind]uint32
	// Create lets OpenWriter create the final store directory component and
	// any missing log or lock file with private modes. Replay ignores it.
	Create bool
	// Hook observes each mutation and identity boundary; a non-nil error
	// aborts that operation. It exists for deterministic fault injection.
	Hook Hook
}

// PointKind names a class of observed store boundary.
type PointKind string

const (
	// PointCreate precedes creating the store directory, lock, or log.
	PointCreate PointKind = "create"
	// PointDirectorySync precedes syncing the store or its parent directory.
	PointDirectorySync PointKind = "directory-sync"
	// PointOpen follows pinning the root or opening the lock or log, before
	// their identity is bound.
	PointOpen PointKind = "open"
	// PointTruncate precedes a writer's torn-tail truncation.
	PointTruncate PointKind = "truncate"
	// PointWrite precedes writing an appended frame.
	PointWrite PointKind = "write"
	// PointSync precedes syncing an appended frame or a repair.
	PointSync PointKind = "sync"
	// PointSynced follows a successful append sync, before Append returns.
	PointSynced PointKind = "synced"
)

// Point is one observed boundary; Operation names the object or step.
type Point struct {
	Kind      PointKind
	Operation string
}

// Hook observes a Point. Returning *PartialWrite at PointWrite writes that
// many leading frame bytes before failing the append.
type Hook func(Point) error

// PartialWrite makes a PointWrite hook simulate an interrupted write.
type PartialWrite struct {
	Bytes int
}

func (partial *PartialWrite) Error() string {
	return fmt.Sprintf("injected partial write of %d bytes", partial.Bytes)
}

func (options Options) validate() (map[Kind]uint32, error) {
	if len(options.Kinds) == 0 {
		return nil, errors.New("log store record kinds are required")
	}
	kinds := make(map[Kind]uint32, len(options.Kinds))
	for kind, limit := range options.Kinds {
		if kind == 0 || limit == 0 || limit > MaxPayloadBytes {
			return nil, fmt.Errorf("log store record kind %d bound %d is invalid", kind, limit)
		}
		kinds[kind] = limit
	}
	return kinds, nil
}

func (hook Hook) observe(kind PointKind, operation string) error {
	if hook == nil {
		return nil
	}
	return hook(Point{Kind: kind, Operation: operation})
}

func invalidStore(format string, arguments ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidStore}, arguments...)...)
}

func validateRootInfo(info os.FileInfo, checkMode bool) error {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return invalidStore("store directory is a symbolic link")
	case !info.IsDir():
		return invalidStore("store path is not a directory")
	case checkMode && info.Mode() != storeMode:
		return invalidStore("store directory mode is %v, want %v", info.Mode(), storeMode)
	}
	return nil
}

func validateMemberInfo(name string, info os.FileInfo) error {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return invalidStore("%s is a symbolic link", name)
	case !info.Mode().IsRegular():
		return invalidStore("%s is not a regular file", name)
	case info.Mode() != memberMode:
		return invalidStore("%s mode is %v, want %v", name, info.Mode(), memberMode)
	}
	if err := hostfs.SingleLink(info); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidStore, err)
	}
	return nil
}

// member is a validated lock or log file. file is nil for a reader's lock,
// which is only inspected.
type member struct {
	name string
	file *os.File
	info os.FileInfo
}

// store is a pinned store directory and its bound members.
type store struct {
	path     string
	root     *os.Root
	rootInfo os.FileInfo
	log      member
	lock     member
	held     *hostfs.Lock
}

// pinRoot opens path as a root and binds it to the validated directory. A
// directory the caller just created is chmodded through its pinned
// descriptor before validation.
func pinRoot(path string, hook Hook, created bool) (*os.Root, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open log store: %w", err)
	}
	if err := validateRootInfo(info, !created); err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, nil, fmt.Errorf("pin log store: %w", err)
	}
	pinned, err := root.Stat(".")
	if err == nil && !os.SameFile(info, pinned) {
		err = invalidStore("store directory changed while opening")
	}
	if err == nil && created {
		err = chmodRoot(root)
	}
	if err == nil {
		err = hook.observe(PointOpen, "root")
	}
	if err == nil {
		info, err = bindRoot(path, pinned)
	}
	if err != nil {
		return nil, nil, errors.Join(err, root.Close())
	}
	return root, info, nil
}

func chmodRoot(root *os.Root) (retErr error) {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, directory.Close()) }()
	return directory.Chmod(storeMode.Perm())
}

// bindRoot checks that path still names the pinned, valid store directory.
func bindRoot(path string, pinned os.FileInfo) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, invalidStore("store directory changed: %v", err)
	}
	if err := validateRootInfo(info, true); err != nil {
		return nil, err
	}
	if !os.SameFile(info, pinned) {
		return nil, invalidStore("store directory was replaced")
	}
	return info, nil
}

// verify checks that the path, root and every bound member still name the
// objects this store bound, before any creation, append, or repair and after
// replay.
func (s *store) verify() error {
	if _, err := bindRoot(s.path, s.rootInfo); err != nil {
		return err
	}
	for _, bound := range []member{s.lock, s.log} {
		if bound.info == nil {
			continue
		}
		current, err := s.root.Lstat(bound.name)
		if err != nil {
			return invalidStore("%s changed: %v", bound.name, err)
		}
		if err := validateMemberInfo(bound.name, current); err != nil {
			return err
		}
		if !os.SameFile(current, bound.info) {
			return invalidStore("%s was replaced", bound.name)
		}
		if bound.file == nil {
			continue
		}
		held, err := bound.file.Stat()
		if err != nil {
			return invalidStore("%s descriptor: %v", bound.name, err)
		}
		if err := validateMemberInfo(bound.name, held); err != nil {
			return err
		}
	}
	return nil
}

func (s *store) close() error {
	var errs []error
	if s.held != nil {
		errs = append(errs, s.held.Release())
		s.held = nil
		s.lock.file = nil
	}
	for _, file := range []*os.File{s.lock.file, s.log.file} {
		if file != nil {
			errs = append(errs, file.Close())
		}
	}
	s.lock.file, s.log.file = nil, nil
	if s.root != nil {
		errs = append(errs, s.root.Close())
		s.root = nil
	}
	return errors.Join(errs...)
}
