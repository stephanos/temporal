package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	ErrCapacity = errors.New("store capacity exhausted")
	ErrCorrupt  = errors.New("store state is corrupt")
	ErrLocked   = errors.New("store run is active")
)

type Store struct {
	root string
}

type Run struct {
	store    *Store
	id       string
	dir      string
	lock     lockRecord
	mu       sync.Mutex
	manifest Manifest
	closed   bool
}

type Manifest struct {
	Schema           string    `json:"schema"`
	RunID            string    `json:"run_id"`
	State            string    `json:"state"`
	Phase            string    `json:"phase,omitempty"`
	Outcome          string    `json:"outcome,omitempty"`
	StartedAt        time.Time `json:"started_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	RequestPath      string    `json:"request_path"`
	RequestBytes     int64     `json:"request_bytes"`
	RequestDigest    string    `json:"request_digest"`
	CheckpointPath   string    `json:"checkpoint_path,omitempty"`
	CheckpointBytes  int64     `json:"checkpoint_bytes,omitempty"`
	CheckpointDigest string    `json:"checkpoint_digest,omitempty"`
	Generation       int       `json:"generation"`
	AttemptCount     int       `json:"attempt_count"`
	ResultPath       string    `json:"result_path,omitempty"`
	ResultBytes      int64     `json:"result_bytes,omitempty"`
	ResultDigest     string    `json:"result_digest,omitempty"`
}

type Inspection struct {
	Manifest    Manifest
	Recoverable bool
	Attempts    []AttemptManifest
}

type AttemptManifest struct {
	Schema       string    `json:"schema"`
	RunID        string    `json:"run_id"`
	Attempt      int       `json:"attempt"`
	Stage        string    `json:"stage"`
	Status       string    `json:"status"`
	Session      string    `json:"session,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	EventPath    string    `json:"event_path"`
	EventCount   int       `json:"event_count"`
	EventBytes   int64     `json:"event_bytes"`
	EventDigest  string    `json:"event_digest,omitempty"`
	OutputPath   string    `json:"output_path,omitempty"`
	OutputBytes  int64     `json:"output_bytes,omitempty"`
	OutputDigest string    `json:"output_digest,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type Recorder struct {
	run         *Run
	directory   string
	manifest    AttemptManifest
	events      *os.File
	eventHasher hashWriter
	maxBytes    int64
	maxEvents   int
	closed      bool
}

type lockRecord struct {
	Schema   string `json:"schema"`
	PID      int    `json:"pid"`
	Host     string `json:"host"`
	Token    string `json:"token"`
	Acquired string `json:"acquired"`
}

func Open(root string) (*Store, error) {
	root, err := filepath.Abs(root)
	if err != nil || root == string(filepath.Separator) {
		return nil, errors.Join(errors.New("store root must be a non-root directory"), err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.Join(errors.New("store root is not a directory"), err)
	}
	return &Store{root: root}, nil
}

func (store *Store) Root() string {
	return store.root
}

func (store *Store) Create(id string, request []byte, now time.Time) (_ *Run, returnedErr error) {
	if err := validComponent(id); err != nil {
		return nil, err
	}
	if !json.Valid(request) {
		return nil, errors.New("store request is not valid JSON")
	}
	directory := filepath.Join(store.root, id)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create run directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed && returnedErr != nil {
			returnedErr = errors.Join(returnedErr, os.RemoveAll(directory))
		}
	}()
	for _, name := range []string{"attempts", "checkpoints", "workspaces"} {
		if err := os.Mkdir(filepath.Join(directory, name), 0o700); err != nil {
			return nil, fmt.Errorf("create run %s directory: %w", name, err)
		}
	}
	run := &Run{store: store, id: id, dir: directory}
	if err := run.acquire(false); err != nil {
		return nil, err
	}
	defer func() {
		if returnedErr != nil {
			returnedErr = errors.Join(returnedErr, run.Close())
		}
	}()
	requestPath := "request.json"
	if err := atomicWrite(filepath.Join(directory, requestPath), request); err != nil {
		return nil, err
	}
	run.manifest = Manifest{
		Schema: "agentworkflow.run/v2", RunID: id, State: "declared", StartedAt: now.UTC(), UpdatedAt: now.UTC(),
		RequestPath: requestPath, RequestBytes: int64(len(request)), RequestDigest: digest("agentworkflow.request/v2", request),
	}
	if err := run.writeManifest(); err != nil {
		return nil, err
	}
	committed = true
	return run, nil
}

func (store *Store) Acquire(id string) (*Run, error) {
	if err := validComponent(id); err != nil {
		return nil, err
	}
	directory := filepath.Join(store.root, id)
	manifest, err := readManifest(directory)
	if err != nil {
		return nil, err
	}
	run := &Run{store: store, id: id, dir: directory, manifest: manifest}
	if err := run.acquire(true); err != nil {
		return nil, err
	}
	return run, nil
}

func (store *Store) Inspect(id string, maxBytes int64) (Inspection, error) {
	if err := validComponent(id); err != nil {
		return Inspection{}, err
	}
	directory := filepath.Join(store.root, id)
	manifest, err := readManifest(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return inspectLegacyRun(directory, id, maxBytes)
		}
		return Inspection{}, err
	}
	if err := verifyRunArtifacts(directory, manifest, maxBytes); err != nil {
		return Inspection{}, err
	}
	inspection := Inspection{Manifest: manifest}
	entries, err := os.ReadDir(filepath.Join(directory, "attempts"))
	if err != nil {
		return Inspection{}, fmt.Errorf("read attempt directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return Inspection{}, fmt.Errorf("%w: unexpected attempt entry %q", ErrCorrupt, entry.Name())
		}
		attempt, err := inspectAttempt(directory, id, entry.Name(), maxBytes)
		if err != nil {
			return Inspection{}, err
		}
		if attempt.Status == "running" {
			inspection.Recoverable = true
		}
		inspection.Attempts = append(inspection.Attempts, attempt)
	}
	slices.SortFunc(inspection.Attempts, func(left, right AttemptManifest) int { return left.Attempt - right.Attempt })
	return inspection, nil
}

func verifyRunArtifacts(directory string, manifest Manifest, maxBytes int64) error {
	if err := verifyArtifact(directory, manifest.RequestPath, manifest.RequestBytes, manifest.RequestDigest, "agentworkflow.request/v2", maxBytes); err != nil {
		return err
	}
	if manifest.CheckpointPath != "" {
		if err := verifyArtifact(directory, manifest.CheckpointPath, manifest.CheckpointBytes, manifest.CheckpointDigest, "agentworkflow.checkpoint/v2", maxBytes); err != nil {
			return err
		}
	}
	if manifest.ResultPath == "" {
		return nil
	}
	return verifyArtifact(directory, manifest.ResultPath, manifest.ResultBytes, manifest.ResultDigest, "agentworkflow.result/v2", maxBytes)
}

func inspectAttempt(directory, runID, name string, maxBytes int64) (AttemptManifest, error) {
	attemptDirectory := filepath.Join(directory, "attempts", name)
	encoded, err := readBounded(filepath.Join(attemptDirectory, "attempt.json"), maxBytes)
	if err != nil {
		return AttemptManifest{}, fmt.Errorf("read attempt %q: %w", name, err)
	}
	var attempt AttemptManifest
	if err := strictDecode(encoded, &attempt); err != nil {
		return AttemptManifest{}, fmt.Errorf("%w: decode attempt %q: %v", ErrCorrupt, name, err)
	}
	if attempt.Schema != "agentworkflow.attempt/v2" || attempt.RunID != runID {
		return AttemptManifest{}, fmt.Errorf("%w: inconsistent attempt %q", ErrCorrupt, name)
	}
	if attempt.Status == "running" {
		return attempt, nil
	}
	if err := verifyArtifact(attemptDirectory, attempt.EventPath, attempt.EventBytes, attempt.EventDigest, "agentworkflow.events/v2", maxBytes); err != nil {
		return AttemptManifest{}, err
	}
	if attempt.OutputPath != "" {
		if err := verifyArtifact(attemptDirectory, attempt.OutputPath, attempt.OutputBytes, attempt.OutputDigest, "agentworkflow.output/v2", maxBytes); err != nil {
			return AttemptManifest{}, err
		}
	}
	return attempt, nil
}

func (run *Run) ID() string {
	return run.id
}

func (run *Run) Directory() string {
	return run.dir
}

func (run *Run) Manifest() Manifest {
	run.mu.Lock()
	defer run.mu.Unlock()
	return run.manifest
}

func (run *Run) ReadRequest(maxBytes int64) ([]byte, error) {
	run.mu.Lock()
	manifest := run.manifest
	run.mu.Unlock()
	if err := verifyArtifact(run.dir, manifest.RequestPath, manifest.RequestBytes, manifest.RequestDigest, "agentworkflow.request/v2", maxBytes); err != nil {
		return nil, err
	}
	return readBounded(filepath.Join(run.dir, manifest.RequestPath), maxBytes)
}

func (run *Run) WriteCheckpoint(data []byte, state, phase, outcome string, now time.Time) error {
	if !json.Valid(data) {
		return errors.New("store checkpoint is not valid JSON")
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return errors.New("store run is closed")
	}
	generation := run.manifest.Generation + 1
	path := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("%06d.json", generation)))
	if err := atomicWrite(filepath.Join(run.dir, filepath.FromSlash(path)), data); err != nil {
		return err
	}
	run.manifest.State = state
	run.manifest.Phase = phase
	run.manifest.Outcome = outcome
	run.manifest.CheckpointPath = path
	run.manifest.CheckpointBytes = int64(len(data))
	run.manifest.CheckpointDigest = digest("agentworkflow.checkpoint/v2", data)
	run.manifest.Generation = generation
	run.manifest.UpdatedAt = now.UTC()
	return run.writeManifest()
}

func (run *Run) ReadCheckpoint(maxBytes int64) ([]byte, error) {
	run.mu.Lock()
	manifest := run.manifest
	run.mu.Unlock()
	if manifest.CheckpointPath == "" {
		return nil, os.ErrNotExist
	}
	if err := verifyArtifact(run.dir, manifest.CheckpointPath, manifest.CheckpointBytes, manifest.CheckpointDigest, "agentworkflow.checkpoint/v2", maxBytes); err != nil {
		return nil, err
	}
	return readBounded(filepath.Join(run.dir, filepath.FromSlash(manifest.CheckpointPath)), maxBytes)
}

func (run *Run) PublishResult(data []byte, outcome, phase string, now time.Time) error {
	if !json.Valid(data) {
		return errors.New("store result is not valid JSON")
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return errors.New("store run is closed")
	}
	path := "result.json"
	if err := atomicWrite(filepath.Join(run.dir, path), data); err != nil {
		return err
	}
	run.manifest.State = "terminal"
	run.manifest.Phase = phase
	run.manifest.Outcome = outcome
	run.manifest.ResultPath = path
	run.manifest.ResultBytes = int64(len(data))
	run.manifest.ResultDigest = digest("agentworkflow.result/v2", data)
	run.manifest.UpdatedAt = now.UTC()
	return run.writeManifest()
}

func (run *Run) ReadResult(maxBytes int64) ([]byte, error) {
	run.mu.Lock()
	manifest := run.manifest
	run.mu.Unlock()
	if manifest.ResultPath == "" {
		return nil, os.ErrNotExist
	}
	if err := verifyArtifact(run.dir, manifest.ResultPath, manifest.ResultBytes, manifest.ResultDigest, "agentworkflow.result/v2", maxBytes); err != nil {
		return nil, err
	}
	return readBounded(filepath.Join(run.dir, manifest.ResultPath), maxBytes)
}

func (run *Run) StartAttempt(stage string, maxBytes int64, maxEvents int, now time.Time) (*Recorder, error) {
	if err := validComponent(stage); err != nil {
		return nil, err
	}
	if maxBytes <= 0 || maxEvents <= 0 {
		return nil, errors.New("attempt bounds must be positive")
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return nil, errors.New("store run is closed")
	}
	attempt := run.manifest.AttemptCount + 1
	directory := filepath.Join(run.dir, "attempts", fmt.Sprintf("%06d-%s", attempt, stage))
	if err := os.Mkdir(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create attempt directory: %w", err)
	}
	events, err := os.OpenFile(filepath.Join(directory, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create attempt event stream: %w", err)
	}
	recorder := &Recorder{
		run: run, directory: directory, events: events, maxBytes: maxBytes, maxEvents: maxEvents,
		eventHasher: newHashWriter("agentworkflow.events/v2"),
		manifest: AttemptManifest{
			Schema: "agentworkflow.attempt/v2", RunID: run.id, Attempt: attempt, Stage: stage,
			Status: "running", StartedAt: now.UTC(), EventPath: "events.jsonl",
		},
	}
	if err := recorder.writeManifest(); err != nil {
		_ = events.Close()
		return nil, err
	}
	run.manifest.AttemptCount = attempt
	run.manifest.UpdatedAt = now.UTC()
	if err := run.writeManifest(); err != nil {
		_ = events.Close()
		return nil, err
	}
	return recorder, nil
}

func (recorder *Recorder) Emit(event []byte) error {
	if recorder.closed {
		return errors.New("attempt recorder is closed")
	}
	if !json.Valid(event) {
		return errors.New("attempt event is not valid JSON")
	}
	if recorder.manifest.EventCount+1 > recorder.maxEvents || recorder.manifest.EventBytes+int64(len(event))+1 > recorder.maxBytes {
		return ErrCapacity
	}
	data := append(append([]byte(nil), event...), '\n')
	if _, err := recorder.events.Write(data); err != nil {
		return fmt.Errorf("write attempt event: %w", err)
	}
	if _, err := recorder.eventHasher.Write(data); err != nil {
		return fmt.Errorf("hash attempt event: %w", err)
	}
	if err := recorder.events.Sync(); err != nil {
		return fmt.Errorf("sync attempt event: %w", err)
	}
	recorder.manifest.EventCount++
	recorder.manifest.EventBytes += int64(len(data))
	return nil
}

func (recorder *Recorder) SetSession(session string) error {
	if recorder.closed {
		return errors.New("attempt recorder is closed")
	}
	if strings.TrimSpace(session) == "" {
		return errors.New("attempt session is empty")
	}
	if recorder.manifest.Session != "" && recorder.manifest.Session != session {
		return errors.New("attempt session identity changed")
	}
	recorder.manifest.Session = session
	return recorder.writeManifest()
}

func (recorder *Recorder) Finish(status, session string, output []byte, failure error, now time.Time) error {
	if recorder.closed {
		return errors.New("attempt recorder is closed")
	}
	if status != "completed" && status != "failed" && status != "interrupted" {
		return fmt.Errorf("attempt terminal status %q is invalid", status)
	}
	recorder.closed = true
	if err := recorder.events.Sync(); err != nil {
		_ = recorder.events.Close()
		return fmt.Errorf("sync attempt event stream: %w", err)
	}
	if err := recorder.events.Close(); err != nil {
		return fmt.Errorf("close attempt event stream: %w", err)
	}
	recorder.manifest.Status = status
	recorder.manifest.Session = session
	recorder.manifest.FinishedAt = now.UTC()
	recorder.manifest.EventDigest = recorder.eventHasher.Digest()
	if failure != nil {
		recorder.manifest.Error = failure.Error()
	}
	capacityExceeded := false
	if output != nil {
		if int64(len(output)) > recorder.maxBytes {
			recorder.manifest.Status = "failed"
			recorder.manifest.Error = ErrCapacity.Error()
			output = nil
			capacityExceeded = true
		} else {
			recorder.manifest.OutputPath = "output.json"
			recorder.manifest.OutputBytes = int64(len(output))
			recorder.manifest.OutputDigest = digest("agentworkflow.output/v2", output)
			if err := atomicWrite(filepath.Join(recorder.directory, recorder.manifest.OutputPath), output); err != nil {
				return err
			}
		}
	}
	writeErr := recorder.writeManifest()
	if capacityExceeded {
		return errors.Join(ErrCapacity, writeErr)
	}
	return writeErr
}

func (run *Run) Close() error {
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return nil
	}
	run.closed = true
	path := filepath.Join(run.dir, "running.lock")
	encoded, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read run lock during release: %w", err)
	}
	var record lockRecord
	if err := strictDecode(encoded, &record); err != nil {
		return fmt.Errorf("decode run lock during release: %w", err)
	}
	if record.Token != run.lock.Token {
		return errors.New("run lock ownership changed")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("release run lock: %w", err)
	}
	return syncDirectory(run.dir)
}

func (run *Run) acquire(recoverStale bool) error {
	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("resolve hostname for run lock: %w", err)
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	record := lockRecord{
		Schema: "agentworkflow.lock/v1", PID: os.Getpid(), Host: host, Token: token,
		Acquired: time.Now().UTC().Format(time.RFC3339Nano),
	}
	path := filepath.Join(run.dir, "running.lock")
	if err := createExclusiveJSON(path, record); err == nil {
		run.lock = record
		return nil
	} else if !errors.Is(err, os.ErrExist) {
		return err
	}
	if !recoverStale {
		return ErrLocked
	}
	existingData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read existing run lock: %w", err)
	}
	var existing lockRecord
	if err := strictDecode(existingData, &existing); err != nil {
		return fmt.Errorf("%w: existing run lock is invalid: %v", ErrLocked, err)
	}
	if existing.Host != host || processAlive(existing.PID) {
		return ErrLocked
	}
	stale := filepath.Join(run.dir, fmt.Sprintf("stale-lock-%s.json", existing.Token))
	if err := os.Rename(path, stale); err != nil {
		return fmt.Errorf("retain stale run lock: %w", err)
	}
	if err := createExclusiveJSON(path, record); err != nil {
		return fmt.Errorf("acquire recovered run lock: %w", err)
	}
	run.lock = record
	return nil
}

func (run *Run) writeManifest() error {
	encoded, err := json.Marshal(run.manifest)
	if err != nil {
		return fmt.Errorf("encode run manifest: %w", err)
	}
	return atomicWrite(filepath.Join(run.dir, "run.json"), encoded)
}

func (recorder *Recorder) writeManifest() error {
	encoded, err := json.Marshal(recorder.manifest)
	if err != nil {
		return fmt.Errorf("encode attempt manifest: %w", err)
	}
	return atomicWrite(filepath.Join(recorder.directory, "attempt.json"), encoded)
}

func readManifest(directory string) (Manifest, error) {
	encoded, err := readBounded(filepath.Join(directory, "run.json"), 1<<20)
	if err != nil {
		return Manifest{}, fmt.Errorf("read run manifest: %w", err)
	}
	var manifest Manifest
	if err := strictDecode(encoded, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: decode run manifest: %v", ErrCorrupt, err)
	}
	if manifest.Schema != "agentworkflow.run/v2" || manifest.RunID != filepath.Base(directory) {
		return Manifest{}, fmt.Errorf("%w: run manifest identity is inconsistent", ErrCorrupt)
	}
	return manifest, nil
}

func verifyArtifact(root, relative string, size int64, expectedDigest, domain string, maxBytes int64) error {
	if relative == "" || filepath.IsAbs(relative) {
		return fmt.Errorf("%w: artifact path %q is invalid", ErrCorrupt, relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: artifact path %q escapes run", ErrCorrupt, relative)
	}
	if size < 0 || size > maxBytes {
		return fmt.Errorf("%w: artifact %q has invalid declared size", ErrCorrupt, relative)
	}
	data, err := readBounded(filepath.Join(root, clean), maxBytes)
	if err != nil {
		return fmt.Errorf("%w: read artifact %q: %v", ErrCorrupt, relative, err)
	}
	if int64(len(data)) != size || digest(domain, data) != expectedDigest {
		return fmt.Errorf("%w: artifact %q failed integrity validation", ErrCorrupt, relative)
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".write-*")
	if err != nil {
		return fmt.Errorf("create temporary artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set artifact permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write artifact: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish artifact: %w", err)
	}
	return syncDirectory(directory)
}

func createExclusiveJSON(path string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	syncErr := handle.Sync()
	closeErr := handle.Close()
	return errors.Join(syncErr, closeErr)
}

func readBounded(path string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("read bound must be positive")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrCapacity
	}
	return data, nil
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validComponent(value string) error {
	if value == "" || value == "." || value == ".." {
		return errors.New("store identity is invalid")
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return fmt.Errorf("store identity %q contains an invalid character", value)
	}
	return nil
}

func randomToken() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate store lock token: %w", err)
	}
	return hex.EncodeToString(data), nil
}

func digest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, domain)
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

type hashWriter struct {
	hasher io.Writer
	sum    interface{ Sum([]byte) []byte }
}

func newHashWriter(domain string) hashWriter {
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, domain)
	_, _ = hasher.Write([]byte{0})
	return hashWriter{hasher: hasher, sum: hasher}
}

func (writer hashWriter) Write(data []byte) (int, error) {
	return writer.hasher.Write(data)
}

func (writer hashWriter) Digest() string {
	return "sha256:" + hex.EncodeToString(writer.sum.Sum(nil))
}
