package minimizer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

const CheckpointSchema = "gomad3.minimizer-checkpoint/v1"

const (
	workspaceStateRoot      = ".minimize"
	workspaceLockSuffix     = ".lock"
	workspaceCheckpointName = "state.json"
	workspaceAcceptedName   = "accepted"
	maximumCheckpointBytes  = 256 << 20
)

// ErrNoCheckpoint reports a resume of an output directory that holds no
// minimizer state for the parent artifact.
var ErrNoCheckpoint = errors.New("minimize output directory holds no minimizer state to resume for this parent artifact")

// ErrCheckpointExists reports an initial run on an output directory that still
// holds minimizer state for the parent artifact.
var ErrCheckpointExists = errors.New("minimize output directory already holds minimizer state for this parent artifact")

// Binding names the inputs a persisted state is only valid for.
type Binding struct {
	ParentRecordHash     record.SHA256 `json:"parent_record_hash"`
	ImplementationSHA256 record.SHA256 `json:"implementation_sha256"`
	ToolchainBuildKey    string        `json:"toolchain_build_key"`
}

// AcceptedArtifact references the retained artifact of the last accepted
// reduction, by its directory name under the workspace's accepted root.
type AcceptedArtifact struct {
	Directory          string        `json:"directory"`
	RecordHash         record.SHA256 `json:"record_hash"`
	ChoiceReplayStatus string        `json:"choice_replay_status"`
}

// PublishedArtifact references the final minimized artifact, by its directory
// name under the output root.
type PublishedArtifact struct {
	Directory  string        `json:"directory"`
	RecordHash record.SHA256 `json:"record_hash"`
}

type Checkpoint struct {
	Schema    string             `json:"schema"`
	Binding   Binding            `json:"binding"`
	State     State              `json:"state"`
	Accepted  *AcceptedArtifact  `json:"accepted,omitempty"`
	Published *PublishedArtifact `json:"published,omitempty"`
	SHA256    record.SHA256      `json:"sha256"`
}

// Workspace owns the minimizer state persisted for one parent artifact under an
// output root. It holds that parent's exclusive lock from OpenWorkspace until
// Close, so runs for different parents share a root without sharing state.
type Workspace struct {
	ctx        context.Context
	root       string
	parent     string
	lock       *hostfs.Lock
	checkpoint Checkpoint
	completed  bool
}

// OpenWorkspace locks the state of binding's parent artifact under root and
// either starts it from initial or, with resume, loads the persisted state and
// checks that it belongs to binding and initial.
func OpenWorkspace(ctx context.Context, root string, binding Binding, initial State, resume bool) (_ *Workspace, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if root == "" {
		return nil, errors.New("minimize output directory is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve minimize output directory: %w", err)
	}
	if err := validateBinding(binding); err != nil {
		return nil, err
	}
	if err := Validate(initial); err != nil {
		return nil, fmt.Errorf("validate initial minimizer state: %w", err)
	}
	workspace := &Workspace{ctx: ctx, root: root, parent: parentDirectory(binding.ParentRecordHash)}
	if !resume {
		if err := os.MkdirAll(workspace.stateRoot(), 0o700); err != nil {
			return nil, fmt.Errorf("create minimizer state root: %w", err)
		}
	}
	if err := requireDirectory(workspace.stateRoot()); resume && errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoCheckpoint
	} else if err != nil {
		return nil, fmt.Errorf("inspect minimizer state root: %w", err)
	}
	workspace.lock, err = acquireWorkspaceLock(workspace.stateDirectory() + workspaceLockSuffix)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, workspace.Close())
		}
	}()
	if resume {
		return workspace, workspace.load(binding, initial)
	}
	return workspace, workspace.create(binding, initial)
}

// parentDirectory names one parent artifact's state under the state root.
func parentDirectory(parent record.SHA256) string {
	return "sha256-" + strings.TrimPrefix(string(parent), "sha256:")
}

func acquireWorkspaceLock(path string) (*hostfs.Lock, error) {
	lock, err := hostfs.Try(path)
	switch {
	case errors.Is(err, hostfs.ErrSymbolicLink):
		return nil, errors.New("minimize output lock is a symbolic link")
	case errors.Is(err, hostfs.ErrContended):
		return nil, fmt.Errorf("another minimize run of this parent artifact is using the output directory: %w", err)
	case errors.Is(err, hostfs.ErrUnsupported):
		return nil, errors.New("minimize is unsupported on this host")
	case err != nil:
		return nil, fmt.Errorf("open minimize output lock: %w", err)
	default:
		return lock, nil
	}
}

func (workspace *Workspace) create(binding Binding, initial State) error {
	if _, err := os.Lstat(workspace.checkpointPath()); err == nil {
		return fmt.Errorf("%w; resume it or remove %s", ErrCheckpointExists, workspace.stateDirectory())
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect minimizer state: %w", err)
	}
	// A state directory without a checkpoint is what a completed run leaves
	// when it dies between removing the checkpoint and removing the directory.
	if err := os.RemoveAll(workspace.stateDirectory()); err != nil {
		return fmt.Errorf("remove stale minimizer state directory: %w", err)
	}
	for _, directory := range []string{workspace.stateDirectory(), workspace.AcceptedRoot()} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return fmt.Errorf("create minimizer state directory: %w", err)
		}
	}
	for _, directory := range []string{workspace.stateRoot(), workspace.root} {
		if err := syncDirectory(directory); err != nil {
			return fmt.Errorf("sync minimize output directory: %w", err)
		}
	}
	return workspace.write(Checkpoint{Schema: CheckpointSchema, Binding: binding, State: initial})
}

func (workspace *Workspace) load(binding Binding, initial State) error {
	checkpoint, err := readCheckpoint(workspace.checkpointPath())
	if err != nil {
		return err
	}
	switch {
	case checkpoint.Binding.ParentRecordHash != binding.ParentRecordHash:
		return errors.New("minimizer state belongs to a different parent artifact")
	case checkpoint.Binding.ImplementationSHA256 != binding.ImplementationSHA256:
		return errors.New("minimizer state was written by a different minimizer implementation")
	case checkpoint.Binding.ToolchainBuildKey != binding.ToolchainBuildKey:
		return errors.New("minimizer state was written under a different toolchain build key")
	case checkpoint.State.AttemptBudget != initial.AttemptBudget:
		return fmt.Errorf("minimizer state has attempt budget %d, not %d", checkpoint.State.AttemptBudget, initial.AttemptBudget)
	}
	sameStart, err := sameStart(checkpoint.State, initial)
	if err != nil {
		return err
	}
	if !sameStart {
		return errors.New("minimizer state does not start from the parent artifact's candidate")
	}
	if checkpoint.Accepted != nil {
		if err := requireDirectory(filepath.Join(workspace.AcceptedRoot(), checkpoint.Accepted.Directory)); err != nil {
			return fmt.Errorf("minimizer state references a missing accepted artifact: %w", err)
		}
	}
	workspace.checkpoint = checkpoint
	return workspace.pruneAccepted()
}

// Checkpoint returns the last persisted checkpoint.
func (workspace *Workspace) Checkpoint() Checkpoint {
	checkpoint := workspace.checkpoint
	checkpoint.State = cloneState(checkpoint.State)
	if checkpoint.Accepted != nil {
		accepted := *checkpoint.Accepted
		checkpoint.Accepted = &accepted
	}
	if checkpoint.Published != nil {
		published := *checkpoint.Published
		checkpoint.Published = &published
	}
	return checkpoint
}

// AcceptedRoot is the artifact store a caller publishes an accepted artifact
// into before committing the state that references it.
func (workspace *Workspace) AcceptedRoot() string {
	return filepath.Join(workspace.stateDirectory(), workspaceAcceptedName)
}

// Commit persists the state after one committed attempt. accepted is required
// exactly when the attempt was accepted, and its artifact must already be on
// disk, so a persisted state never references an artifact that is missing.
func (workspace *Workspace) Commit(state State, accepted *AcceptedArtifact) error {
	if workspace.completed {
		return errors.New("minimizer workspace is already complete")
	}
	previous := workspace.checkpoint
	if previous.Published != nil {
		return errors.New("minimizer state is already published")
	}
	sameStart, err := sameStart(state, previous.State)
	if err != nil {
		return err
	}
	if !sameStart || state.AttemptBudget != previous.State.AttemptBudget || state.Attempts != previous.State.Attempts+1 {
		return errors.New("minimizer state does not follow the persisted state")
	}
	next := Checkpoint{Schema: CheckpointSchema, Binding: previous.Binding, State: cloneState(state), Accepted: previous.Accepted}
	switch len(state.Accepted) - len(previous.State.Accepted) {
	case 0:
		if accepted != nil {
			return errors.New("rejected minimizer attempt carries an accepted artifact")
		}
	case 1:
		if accepted == nil {
			return errors.New("accepted minimizer attempt requires its retained artifact")
		}
		reference := *accepted
		next.Accepted = &reference
		if err := validateAccepted(reference); err != nil {
			return err
		}
		if err := requireDirectory(filepath.Join(workspace.AcceptedRoot(), reference.Directory)); err != nil {
			return fmt.Errorf("accepted minimizer artifact is not retained: %w", err)
		}
	default:
		return errors.New("minimizer state does not follow the persisted state")
	}
	if err := workspace.write(next); err != nil {
		return err
	}
	return workspace.pruneAccepted()
}

// RecordPublication persists the identity of the final artifact after it is
// published, so a resumed run validates it and does not publish again.
func (workspace *Workspace) RecordPublication(published PublishedArtifact) error {
	if workspace.completed {
		return errors.New("minimizer workspace is already complete")
	}
	next := workspace.checkpoint
	if next.Published != nil {
		return errors.New("minimizer state is already published")
	}
	next.Published = &published
	if err := validatePublished(published); err != nil {
		return err
	}
	if err := requireDirectory(filepath.Join(workspace.root, published.Directory)); err != nil {
		return fmt.Errorf("published minimized artifact is missing: %w", err)
	}
	return workspace.write(next)
}

// Complete removes the persisted state once the run's result is final.
func (workspace *Workspace) Complete() error {
	if workspace.completed {
		return nil
	}
	if err := workspace.ctx.Err(); err != nil {
		return err
	}
	// The checkpoint goes first: a state directory without one is discarded
	// by the next run, while a checkpoint without its artifact fails closed.
	if err := os.Remove(workspace.checkpointPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove minimizer state: %w", err)
	}
	if err := syncDirectory(workspace.stateDirectory()); err != nil {
		return fmt.Errorf("sync minimizer state directory: %w", err)
	}
	workspace.completed = true
	if err := os.RemoveAll(workspace.stateDirectory()); err != nil {
		return fmt.Errorf("remove minimizer state directory: %w", err)
	}
	if err := syncDirectory(workspace.stateRoot()); err != nil {
		return fmt.Errorf("sync minimizer state root: %w", err)
	}
	return nil
}

// Close releases the workspace lock.
func (workspace *Workspace) Close() error {
	if workspace == nil {
		return nil
	}
	lock := workspace.lock
	workspace.lock = nil
	return lock.Release()
}

func (workspace *Workspace) stateRoot() string {
	return filepath.Join(workspace.root, workspaceStateRoot)
}

func (workspace *Workspace) stateDirectory() string {
	return filepath.Join(workspace.stateRoot(), workspace.parent)
}

func (workspace *Workspace) checkpointPath() string {
	return filepath.Join(workspace.stateDirectory(), workspaceCheckpointName)
}

func (workspace *Workspace) write(checkpoint Checkpoint) error {
	sealed, err := sealCheckpoint(checkpoint)
	if err != nil {
		return err
	}
	encoded, err := canonicaljson.CanonicalJSON(sealed)
	if err != nil {
		return err
	}
	if len(encoded) > maximumCheckpointBytes {
		return fmt.Errorf("minimizer state requires %d bytes, exceeding its %d-byte capacity", len(encoded), maximumCheckpointBytes)
	}
	if err := hostfs.ReplaceContext(workspace.ctx, workspace.checkpointPath(), encoded, 0o600); err != nil {
		return fmt.Errorf("write minimizer state: %w", err)
	}
	workspace.checkpoint = sealed
	return nil
}

// pruneAccepted keeps only the referenced accepted artifact: superseded ones,
// and one retained for an attempt whose state was never committed, are dropped.
func (workspace *Workspace) pruneAccepted() error {
	entries, err := os.ReadDir(workspace.AcceptedRoot())
	if err != nil {
		return fmt.Errorf("list accepted minimizer artifacts: %w", err)
	}
	for _, entry := range entries {
		if workspace.checkpoint.Accepted != nil && entry.Name() == workspace.checkpoint.Accepted.Directory {
			continue
		}
		if err := os.RemoveAll(filepath.Join(workspace.AcceptedRoot(), entry.Name())); err != nil {
			return fmt.Errorf("remove superseded accepted minimizer artifact: %w", err)
		}
	}
	return nil
}

func readCheckpoint(path string) (Checkpoint, error) {
	file, info, err := hostfs.OpenPath(path)
	if errors.Is(err, os.ErrNotExist) {
		return Checkpoint{}, ErrNoCheckpoint
	}
	if err != nil {
		return Checkpoint{}, fmt.Errorf("open minimizer state: %w", err)
	}
	if info.Size() > maximumCheckpointBytes {
		return Checkpoint{}, errors.Join(errors.New("minimizer state exceeds its byte capacity"), file.Close())
	}
	encoded, err := io.ReadAll(io.LimitReader(file, maximumCheckpointBytes+1))
	if err := errors.Join(err, file.Close()); err != nil {
		return Checkpoint{}, fmt.Errorf("read minimizer state: %w", err)
	}
	var checkpoint Checkpoint
	if err := canonicaljson.DecodeCanonicalJSON(encoded, &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("decode minimizer state: %w", err)
	}
	if err := validateCheckpoint(checkpoint); err != nil {
		return Checkpoint{}, err
	}
	return checkpoint, nil
}

func sealCheckpoint(checkpoint Checkpoint) (Checkpoint, error) {
	identity, err := checkpointIdentity(checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}
	checkpoint.SHA256 = identity
	return checkpoint, validateCheckpoint(checkpoint)
}

func checkpointIdentity(checkpoint Checkpoint) (record.SHA256, error) {
	checkpoint.SHA256 = ""
	encoded, err := canonicaljson.CanonicalJSON(checkpoint)
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomad3-minimizer-checkpoint/v1", encoded), nil
}

func validateCheckpoint(checkpoint Checkpoint) error {
	if checkpoint.Schema != CheckpointSchema {
		return fmt.Errorf("unknown minimizer state schema %q", checkpoint.Schema)
	}
	want, err := checkpointIdentity(checkpoint)
	if err != nil {
		return err
	}
	if checkpoint.SHA256 != want {
		return errors.New("minimizer state file identity changed")
	}
	if err := validateBinding(checkpoint.Binding); err != nil {
		return err
	}
	if err := Validate(checkpoint.State); err != nil {
		return err
	}
	if (checkpoint.Accepted != nil) != (len(checkpoint.State.Accepted) != 0) {
		return errors.New("minimizer state and its accepted artifact disagree")
	}
	if checkpoint.Accepted != nil {
		if err := validateAccepted(*checkpoint.Accepted); err != nil {
			return err
		}
	}
	if checkpoint.Published != nil {
		if checkpoint.Accepted == nil || checkpoint.State.StopReason == "" {
			return errors.New("minimizer state is published before it stopped with an accepted artifact")
		}
		if err := validatePublished(*checkpoint.Published); err != nil {
			return err
		}
	}
	return nil
}

func validateBinding(binding Binding) error {
	for _, identity := range []record.SHA256{binding.ParentRecordHash, binding.ImplementationSHA256} {
		if _, err := identity.Bytes(); err != nil {
			return fmt.Errorf("minimizer state binding: %w", err)
		}
	}
	if binding.ToolchainBuildKey == "" {
		return errors.New("minimizer state binding requires a toolchain build key")
	}
	return nil
}

func validateAccepted(accepted AcceptedArtifact) error {
	if accepted.ChoiceReplayStatus == "" {
		return errors.New("accepted minimizer artifact has no choice replay status")
	}
	return validateArtifactReference(accepted.Directory, accepted.RecordHash)
}

func validatePublished(published PublishedArtifact) error {
	return validateArtifactReference(published.Directory, published.RecordHash)
}

func validateArtifactReference(directory string, recordHash record.SHA256) error {
	if directory == "" || directory != filepath.Base(directory) || !filepath.IsLocal(directory) || directory[0] == '.' {
		return fmt.Errorf("minimizer artifact directory %q is not a store entry", directory)
	}
	if _, err := recordHash.Bytes(); err != nil {
		return fmt.Errorf("minimizer artifact record hash: %w", err)
	}
	return nil
}

func sameStart(left, right State) (bool, error) {
	if !sameCandidate(left.Original, right.Original) {
		return false, nil
	}
	leftConfig, err := canonicaljson.CanonicalJSON(left.Config)
	if err != nil {
		return false, err
	}
	rightConfig, err := canonicaljson.CanonicalJSON(right.Config)
	if err != nil {
		return false, err
	}
	return string(leftConfig) == string(rightConfig), nil
}

func requireDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", filepath.Base(path))
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
