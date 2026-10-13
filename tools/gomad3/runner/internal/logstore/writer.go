package logstore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

// Writer is the exclusive appender of one store. Its methods are safe for
// concurrent use.
type Writer struct {
	mu      sync.Mutex
	store   *store
	kinds   map[Kind]uint32
	hook    Hook
	offset  int64
	records uint64
	poison  error
	closed  bool
}

// OpenWriter takes the store's exclusive lock, replays its log to consume,
// and truncates and syncs a recognized torn tail before returning a Writer.
// A second writer on the same store, through any accepted path alias, fails
// with ErrInUse. Corruption and invalid metadata fail before any mutation.
func OpenWriter(path string, options Options, consume func(Record) error) (_ *Writer, _ Report, retErr error) {
	kinds, err := options.validate()
	if err != nil {
		return nil, Report{}, err
	}
	hook := options.Hook
	createdRoot := false
	if options.Create {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			if err := hook.observe(PointCreate, "root"); err != nil {
				return nil, Report{}, err
			}
			if err := os.Mkdir(path, storeMode.Perm()); err != nil {
				return nil, Report{}, fmt.Errorf("create log store: %w", err)
			}
			createdRoot = true
		}
	}
	root, rootInfo, err := pinRoot(path, hook, createdRoot)
	if err != nil {
		return nil, Report{}, err
	}
	s := &store{path: path, root: root, rootInfo: rootInfo}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, s.close())
		}
	}()
	missing, err := inspectMembers(root, options.Create)
	if err != nil {
		return nil, Report{}, err
	}
	if s.lock, err = openMember(root, LockFile, missing[LockFile], hook); err != nil {
		return nil, Report{}, err
	}
	if s.held, err = hostfs.TryDescriptor(s.lock.file); err != nil {
		if errors.Is(err, hostfs.ErrContended) {
			return nil, Report{}, fmt.Errorf("%w: %w", ErrInUse, err)
		}
		return nil, Report{}, fmt.Errorf("lock log store: %w", err)
	}
	if err := s.verify(); err != nil {
		return nil, Report{}, err
	}
	if s.log, err = openMember(root, LogFile, missing[LogFile], hook); err != nil {
		return nil, Report{}, err
	}
	if err := syncCreated(s, hook, missing[LockFile] || missing[LogFile], createdRoot); err != nil {
		return nil, Report{}, err
	}
	if err := s.verify(); err != nil {
		return nil, Report{}, err
	}
	report, err := scan(s.log.file, s.log.info.Size(), kinds, consume)
	if errors.Is(err, errShortRead) {
		return nil, report, invalidStore("log shrank under the writer lock")
	}
	if err != nil {
		return nil, report, err
	}
	if report.Tail != TailNone {
		if err := repair(s, hook, report.ValidBytes); err != nil {
			return nil, report, err
		}
		report.RepairedBytes = report.ObservedTailBytes
	}
	return &Writer{store: s, kinds: kinds, hook: hook, offset: report.ValidBytes, records: report.Records}, report, nil
}

// inspectMembers validates every existing member before anything is created,
// and reports which members are missing.
func inspectMembers(root *os.Root, create bool) (map[string]bool, error) {
	missing := map[string]bool{}
	for _, name := range []string{LockFile, LogFile} {
		info, err := root.Lstat(name)
		switch {
		case errors.Is(err, os.ErrNotExist) && create:
			missing[name] = true
		case errors.Is(err, os.ErrNotExist):
			return nil, fmt.Errorf("open log store %s: %w", name, err)
		case err != nil:
			return nil, invalidStore("%s: %v", name, err)
		default:
			if err := validateMemberInfo(name, info); err != nil {
				return nil, err
			}
		}
	}
	return missing, nil
}

func openMember(root *os.Root, name string, create bool, hook Hook) (member, error) {
	var file *os.File
	var info os.FileInfo
	var err error
	if create {
		if err := hook.observe(PointCreate, name); err != nil {
			return member{}, err
		}
		file, info, err = hostfs.CreateRootExclusive(root, name, memberMode)
	} else {
		file, info, err = hostfs.OpenRootWritable(root, name)
	}
	if err != nil {
		return member{}, fmt.Errorf("%w: open %s: %w", ErrInvalidStore, name, err)
	}
	if err := validateMemberInfo(name, info); err != nil {
		return member{}, errors.Join(err, file.Close())
	}
	if err := hook.observe(PointOpen, name); err != nil {
		return member{}, errors.Join(err, file.Close())
	}
	return member{name: name, file: file, info: info}, nil
}

func syncCreated(s *store, hook Hook, createdMember, createdRoot bool) error {
	if createdMember || createdRoot {
		if err := hook.observe(PointDirectorySync, "store"); err != nil {
			return err
		}
		if err := hostfs.SyncRoot(s.root); err != nil {
			return fmt.Errorf("sync log store: %w", err)
		}
	}
	if createdRoot {
		if err := hook.observe(PointDirectorySync, "parent"); err != nil {
			return err
		}
		if err := hostfs.SyncDirectory(filepath.Dir(s.path)); err != nil {
			return fmt.Errorf("sync log store parent: %w", err)
		}
	}
	return nil
}

func repair(s *store, hook Hook, validBytes int64) error {
	if err := hook.observe(PointTruncate, "repair"); err != nil {
		return err
	}
	if err := s.verify(); err != nil {
		return err
	}
	if err := s.log.file.Truncate(validBytes); err != nil {
		return fmt.Errorf("truncate torn log tail: %w", err)
	}
	if err := hook.observe(PointSync, "repair"); err != nil {
		return err
	}
	if err := s.log.file.Sync(); err != nil {
		return fmt.Errorf("sync repaired log: %w", err)
	}
	return nil
}

// Append writes and syncs one record as one transaction. It returns nil only
// after the frame is durable; the caller advances committed state only then.
// Any write, sync, or identity failure poisons the writer: a complete frame
// whose call failed may survive, and the next OpenWriter replay decides it.
func (w *Writer) Append(kind Kind, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrClosed
	}
	if w.poison != nil {
		return fmt.Errorf("%w: %w", ErrPoisoned, w.poison)
	}
	limit, known := w.kinds[kind]
	if !known {
		return fmt.Errorf("append log record: kind %d is unknown", kind)
	}
	if uint64(len(payload)) > uint64(limit) {
		return fmt.Errorf("append log record: kind %d payload length %d exceeds its bound %d", kind, len(payload), limit)
	}
	return w.commit(encodeFrame(kind, payload))
}

// AppendJSON encodes value with EncodeJSON and appends it.
func (w *Writer) AppendJSON(kind Kind, value any) error {
	payload, err := EncodeJSON(value)
	if err != nil {
		return err
	}
	return w.Append(kind, payload)
}

func (w *Writer) commit(frame []byte) error {
	written := len(frame)
	var injected error
	if err := w.hook.observe(PointWrite, "append"); err != nil {
		var partial *PartialWrite
		if !errors.As(err, &partial) {
			return w.fail(err)
		}
		written, injected = max(0, min(partial.Bytes, len(frame))), err
	}
	if err := w.verifyAppend(); err != nil {
		return w.fail(err)
	}
	count, err := w.store.log.file.WriteAt(frame[:written], w.offset)
	if err == nil && count != written {
		err = io.ErrShortWrite
	}
	if err = errors.Join(err, injected); err != nil {
		return w.fail(err)
	}
	if err := w.hook.observe(PointSync, "append"); err != nil {
		return w.fail(err)
	}
	if err := w.store.log.file.Sync(); err != nil {
		return w.fail(err)
	}
	w.offset += int64(len(frame))
	w.records++
	if err := w.hook.observe(PointSynced, "append"); err != nil {
		return w.fail(err)
	}
	return nil
}

func (w *Writer) verifyAppend() error {
	if err := w.store.verify(); err != nil {
		return err
	}
	info, err := w.store.log.file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != w.offset {
		return invalidStore("log size %d differs from the committed offset %d", info.Size(), w.offset)
	}
	return nil
}

func (w *Writer) fail(err error) error {
	w.poison = err
	return fmt.Errorf("%w: %w", ErrPoisoned, err)
}

// Offset returns the byte length of the acknowledged log.
func (w *Writer) Offset() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.offset
}

// Records returns the number of acknowledged records, including replayed ones.
func (w *Writer) Records() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.records
}

// Close releases the lock and every held descriptor. It is idempotent.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	return w.store.close()
}
