// Package recovery is the canary's in-job recovery record: the invocation, the lease it took or
// found, the Run in flight and its phase, and each iteration's Run with whether its receipt was
// published. It lives only for its job, in one mode-0600 file that `run` creates before it reads
// the lease and rewrites at each phase, and that `reconcile` reads to act on exactly the lease its
// own job recorded. It holds identities only: no credential, coordinate or Run content.
package recovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Version is the record format this package reads.
const Version = 1

// fileMode is the record's only mode.
const fileMode fs.FileMode = 0o600

// maxRecordBytes bounds a read: a record holds a few identities per iteration.
const maxRecordBytes = 64 << 10

// How the job came to hold the lease it records.
const (
	// HeldTook is a lease this job started; its process is the one that ends it.
	HeldTook = "took"
	// HeldFound is an unreconciled lease this job found and refused to run past.
	HeldFound = "found"
)

// The phases a record moves through.
const (
	PhaseStarted    = "started"
	PhaseLeasing    = "leasing"
	PhaseRunning    = "running"
	PhaseCleaning   = "cleaning"
	PhaseReleased   = "released"
	PhaseUncertain  = "uncertain"
	PhaseRefused    = "refused"
	PhasePublishing = "publishing"
	PhaseFinished   = "finished"
)

var (
	phases = []string{PhaseStarted, PhaseLeasing, PhaseRunning, PhaseCleaning, PhaseReleased, PhaseUncertain, PhaseRefused, PhasePublishing, PhaseFinished}
	helds  = []string{HeldTook, HeldFound}
)

// Lease is the lease workflow's ID and the run ID the job took or found; the run ID is the fence.
type Lease struct {
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	Held       string `json:"held"`
}

// Iteration is one Run this job made and whether its receipt was published.
type Iteration struct {
	RunID     string `json:"runId"`
	Published bool   `json:"published"`
}

// Record is one job's recovery record.
type Record struct {
	Version      int         `json:"version"`
	InvocationID string      `json:"invocationId"`
	Phase        string      `json:"phase"`
	Lease        *Lease      `json:"lease"`
	CurrentRunID string      `json:"currentRunId"`
	Iterations   []Iteration `json:"iterations"`
}

func (r *Record) validate() error {
	if r.Version != Version {
		return fmt.Errorf("recovery record format version %d, not %d", r.Version, Version)
	}
	if r.InvocationID == "" {
		return errors.New("the recovery record names no invocation")
	}
	if !slices.Contains(phases, r.Phase) {
		return fmt.Errorf("recovery record phase %q is not one of the canary's", r.Phase)
	}
	if r.Lease != nil && (r.Lease.WorkflowID == "" || r.Lease.RunID == "" || !slices.Contains(helds, r.Lease.Held)) {
		return errors.New("the recovery record's lease needs a workflow ID, a run ID and whether it was took or found")
	}
	for _, iteration := range r.Iterations {
		if iteration.RunID == "" {
			return errors.New("a recovery record iteration names no Run")
		}
	}
	return nil
}

// Encode renders a valid record canonically: two-space indented, with a final newline.
func Encode(record *Record) ([]byte, error) {
	if record == nil {
		return nil, errors.New("no recovery record")
	}
	if err := record.validate(); err != nil {
		return nil, err
	}
	canonical := *record
	if canonical.Iterations == nil {
		canonical.Iterations = []Iteration{}
	}
	encoded, err := json.MarshalIndent(&canonical, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// Decode reads a record strictly: every field valid and the bytes exactly its canonical rendering,
// so an unknown, repeated or case-folded key, another order or other spacing is refused.
func Decode(encoded []byte) (*Record, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return nil, fmt.Errorf("decode the recovery record: %w", err)
	}
	canonical, err := Encode(&record)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, encoded) {
		return nil, errors.New("the recovery record is not in its canonical form")
	}
	return &record, nil
}

// Read reads the record at path, refusing one whose mode is not 0600 or that is not a regular
// file. A missing record is fs.ErrNotExist, which reconcile reports as nothing to reconcile.
func Read(path string) (*Record, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("the recovery record %s is not a regular file", filepath.Base(path))
	}
	if info.Mode().Perm() != fileMode {
		return nil, fmt.Errorf("the recovery record's mode is %#o, not %#o", info.Mode().Perm(), fileMode)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	encoded, err := io.ReadAll(io.LimitReader(file, maxRecordBytes+1))
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxRecordBytes {
		return nil, fmt.Errorf("the recovery record exceeds %d bytes", maxRecordBytes)
	}
	return Decode(encoded)
}

// Store is the record a running job owns: every update is written whole before it returns.
type Store struct {
	mu     sync.Mutex
	path   string
	record Record
}

// Create writes a new record for the invocation, in phase started. A record already at path
// refuses: a job writes one record, once.
func Create(path, invocationID string) (*Store, error) {
	store := &Store{path: path, record: Record{Version: Version, InvocationID: invocationID, Phase: PhaseStarted}}
	encoded, err := Encode(&store.record)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return nil, fmt.Errorf("create the recovery record: %w", err)
	}
	_, writeErr := file.Write(encoded)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return nil, fmt.Errorf("write the recovery record: %w", err)
	}
	return store, nil
}

// Snapshot is a copy of the record as last written.
func (s *Store) Snapshot() Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.record)
}

// Update applies change to a copy of the record and writes it; the record changes only when the
// write succeeds.
func (s *Store) Update(change func(*Record)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.record)
	change(&next)
	encoded, err := Encode(&next)
	if err != nil {
		return err
	}
	if err := replace(s.path, encoded); err != nil {
		return err
	}
	s.record = next
	return nil
}

func clone(record Record) Record {
	if record.Lease != nil {
		lease := *record.Lease
		record.Lease = &lease
	}
	record.Iterations = slices.Clone(record.Iterations)
	return record
}

// replace writes contents beside path at mode 0600 and renames it over path, so a reader sees the
// old record or the new one, never a torn one.
func replace(path string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("update the recovery record: %w", err)
	}
	name := temporary.Name()
	_, writeErr := temporary.Write(contents)
	chmodErr := temporary.Chmod(fileMode)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if err := errors.Join(writeErr, chmodErr, syncErr, closeErr); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("update the recovery record: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("update the recovery record: %w", err)
	}
	return nil
}
