package agentworkflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/store"
	"go.temporal.io/server/tools/agentworkflow/internal/workspace"
)

type Engine struct {
	backend Backend
	limits  Limits
	store   *store.Store
	observe func(Progress)
}

func Open(config Config) (*Engine, error) {
	if config.Backend == nil {
		return nil, errors.New("agentworkflow backend is required")
	}
	if config.Limits == (Limits{}) {
		config.Limits = DefaultLimits()
	}
	if err := validateLimits(config.Limits); err != nil {
		return nil, err
	}
	runStore, err := store.Open(config.Root)
	if err != nil {
		return nil, fmt.Errorf("open agentworkflow store: %w", err)
	}
	return &Engine{backend: config.Backend, limits: config.Limits, store: runStore, observe: config.Observe}, nil
}

func (engine *Engine) Run(ctx context.Context, request Request) (Result, error) {
	request, err := normalizeRequest(request, engine.limits)
	if err != nil {
		return Result{}, err
	}
	backend, err := engine.describe(ctx)
	if err != nil {
		return Result{}, err
	}
	requestData, err := json.Marshal(request)
	if err != nil {
		return Result{}, fmt.Errorf("encode agentworkflow request: %w", err)
	}
	if int64(len(requestData)) > engine.limits.MaxOutputBytes {
		return Result{}, fmt.Errorf("%w: encoded request", ErrCapacity)
	}
	id, err := newRunID()
	if err != nil {
		return Result{}, err
	}
	startedAt := time.Now().UTC()
	run, err := engine.store.Create(string(id), requestData, startedAt)
	if err != nil {
		return Result{}, mapStoreError(err)
	}
	checkpoint := checkpoint{
		Schema: "agentworkflow.checkpoint/v2", Request: request, Backend: backend, Limits: engine.limits,
		StartedAt: startedAt, Completed: make(map[string]bool), Phase: "validate", progress: engine.observe,
	}
	if err := saveCheckpoint(run, &checkpoint, "validated", "validate", ""); err != nil {
		closeErr := run.Close()
		return Result{}, errors.Join(err, closeErr)
	}
	prepared, err := workspace.Prepare(ctx, request.Project.Root, run.Directory(), workspaceOptions(request.Project, engine.limits))
	if err != nil {
		result := checkpoint.result(id, run.Directory(), OutcomeInfrastructureFailed, "prepare", err.Error())
		return result, engine.publishAndClose(run, result, err)
	}
	checkpoint.SourceDigest = prepared.Digest
	checkpoint.CandidateDigest = prepared.Digest
	checkpoint.Phase = "prepare"
	checkpoint.Completed["prepare"] = true
	if err := saveCheckpoint(run, &checkpoint, "prepared", "prepare", ""); err != nil {
		result := checkpoint.result(id, run.Directory(), OutcomeInfrastructureFailed, "prepare", err.Error())
		return result, engine.publishAndClose(run, result, err)
	}
	result, workflowErr := engine.execute(ctx, run, prepared, &checkpoint)
	return result, engine.publishAndClose(run, result, workflowErr)
}

func (engine *Engine) Resume(ctx context.Context, id RunID) (Result, error) {
	if err := validateComponent("run ID", string(id)); err != nil {
		return Result{}, err
	}
	inspection, inspectErr := engine.store.Inspect(string(id), engine.limits.MaxOutputBytes)
	if inspectErr != nil && !errors.Is(inspectErr, os.ErrNotExist) {
		return Result{}, mapStoreError(inspectErr)
	}
	if inspectErr == nil && inspection.Manifest.State == "terminal" {
		return engine.readResult(string(id))
	}
	run, err := engine.store.Acquire(string(id))
	if err != nil {
		return Result{}, mapStoreError(err)
	}
	inspection, checkpoint, err := engine.recoverRun(run, id)
	if err != nil {
		closeErr := run.Close()
		return Result{}, errors.Join(err, closeErr)
	}
	backend, err := engine.describe(ctx)
	if err != nil {
		closeErr := run.Close()
		return Result{}, errors.Join(err, closeErr)
	}
	if !sameBackend(checkpoint.Backend, backend) {
		closeErr := run.Close()
		return Result{}, errors.Join(fmt.Errorf("%w: backend identity changed", ErrUnsupported), closeErr)
	}
	prepared, err := workspace.Reopen(checkpoint.Request.Project.Root, run.Directory(), checkpoint.SourceDigest, workspaceOptions(checkpoint.Request.Project, checkpoint.Limits))
	if err != nil {
		result := checkpoint.result(id, run.Directory(), OutcomeCorrupt, "resume", err.Error())
		return result, engine.publishAndClose(run, result, fmt.Errorf("%w: %v", ErrCorrupt, err))
	}
	currentSource, err := workspace.Snapshot(ctx, prepared.Source, workspaceOptions(checkpoint.Request.Project, checkpoint.Limits))
	if err != nil {
		result := checkpoint.result(id, run.Directory(), OutcomeInfrastructureFailed, "resume", err.Error())
		return result, engine.publishAndClose(run, result, err)
	}
	if currentSource != checkpoint.SourceDigest {
		result := checkpoint.result(id, run.Directory(), OutcomeInconclusive, "resume", "source changed after the run was admitted")
		return result, engine.publishAndClose(run, result, fmt.Errorf("%w: admitted %s, current %s", ErrSourceDrift, checkpoint.SourceDigest, currentSource))
	}
	if !reconcileMutationAttempt(&checkpoint, inspection.Attempts) {
		result := checkpoint.result(id, run.Directory(), OutcomeRecoverableInterruption, checkpoint.Phase, "mutation may be partial and no resumable provider session was retained")
		return result, engine.publishAndClose(run, result, nil)
	}
	result, workflowErr := engine.execute(ctx, run, prepared, &checkpoint)
	return result, engine.publishAndClose(run, result, workflowErr)
}

func (engine *Engine) recoverRun(run *store.Run, id RunID) (store.Inspection, checkpoint, error) {
	if err := run.RecoverAttempts(engine.limits.MaxOutputBytes, engine.limits.MaxEvents, time.Now().UTC()); err != nil {
		return store.Inspection{}, checkpoint{}, mapStoreError(err)
	}
	inspection, err := engine.store.Inspect(string(id), engine.limits.MaxOutputBytes)
	if err != nil {
		return store.Inspection{}, checkpoint{}, mapStoreError(err)
	}
	checkpointData, err := run.ReadCheckpoint(engine.limits.MaxOutputBytes)
	if err != nil {
		return store.Inspection{}, checkpoint{}, mapStoreError(err)
	}
	var value checkpoint
	if err := strictDecode(checkpointData, &value); err != nil || value.Schema != "agentworkflow.checkpoint/v2" {
		return store.Inspection{}, checkpoint{}, fmt.Errorf("%w: decode checkpoint: %v", ErrCorrupt, err)
	}
	if value.Limits != engine.limits {
		return store.Inspection{}, checkpoint{}, fmt.Errorf("%w: resume limits differ from admitted limits", ErrUnsupported)
	}
	value.progress = engine.observe
	return inspection, value, nil
}

func reconcileMutationAttempt(checkpoint *checkpoint, attempts []store.AttemptManifest) bool {
	if !isMutationPhase(checkpoint.Phase) || checkpoint.Completed[checkpoint.Phase] || checkpoint.ImplementationSession != "" {
		return true
	}
	for index := len(attempts) - 1; index >= 0; index-- {
		attempt := attempts[index]
		if attempt.Stage != checkpoint.Phase || attempt.Session == "" {
			continue
		}
		if attempt.Status == "completed" || (attempt.Status == "interrupted" && slices.Contains(checkpoint.Backend.Capabilities, CapabilityResume)) {
			checkpoint.ImplementationSession = attempt.Session
			return true
		}
	}
	return false
}

func (engine *Engine) Inspect(_ context.Context, id RunID) (Status, error) {
	if err := validateComponent("run ID", string(id)); err != nil {
		return Status{}, err
	}
	inspection, err := engine.store.Inspect(string(id), engine.limits.MaxOutputBytes)
	if err != nil {
		return Status{
			Schema: "agentworkflow.status/v2", RunID: id, State: "corrupt", Outcome: OutcomeCorrupt,
			CorruptReason: err.Error(),
		}, mapStoreError(err)
	}
	status := Status{
		Schema: "agentworkflow.status/v2", RunID: id, State: inspection.Manifest.State,
		Phase: inspection.Manifest.Phase, Outcome: Outcome(inspection.Manifest.Outcome), Recoverable: inspection.Recoverable,
		StartedAt: inspection.Manifest.StartedAt, UpdatedAt: inspection.Manifest.UpdatedAt,
	}
	if inspection.Manifest.ResultPath != "" {
		result, readErr := engine.readResult(string(id))
		if readErr != nil {
			status.State = "corrupt"
			status.Outcome = OutcomeCorrupt
			status.CorruptReason = readErr.Error()
			return status, readErr
		}
		status.Result = &result
	}
	return status, nil
}

func (engine *Engine) Diff(ctx context.Context, id RunID) (_ []Change, returnedErr error) {
	result, err := engine.readResult(string(id))
	if err != nil {
		return nil, err
	}
	run, err := engine.store.Acquire(string(id))
	if err != nil {
		return nil, mapStoreError(err)
	}
	defer func() { returnedErr = errors.Join(returnedErr, run.Close()) }()
	checkpoint, prepared, err := engine.reopenCheckpoint(run)
	if err != nil {
		return nil, err
	}
	changes, candidateDigest, err := workspace.Diff(ctx, prepared)
	if err != nil {
		return nil, err
	}
	if candidateDigest != result.CandidateDigest || checkpoint.CandidateDigest != result.CandidateDigest {
		return nil, fmt.Errorf("%w: candidate digest differs from terminal result", ErrCorrupt)
	}
	return publicChanges(changes), nil
}

func (engine *Engine) Apply(ctx context.Context, id RunID) (returnedErr error) {
	result, err := engine.readResult(string(id))
	if err != nil {
		return err
	}
	run, err := engine.store.Acquire(string(id))
	if err != nil {
		return mapStoreError(err)
	}
	defer func() { returnedErr = errors.Join(returnedErr, run.Close()) }()
	checkpoint, prepared, err := engine.reopenCheckpoint(run)
	if err != nil {
		return err
	}
	if !workflowStage(checkpoint.Request.Workflow, StageApply).Enabled {
		return fmt.Errorf("%w: apply stage is disabled for run %s", ErrUnsupported, id)
	}
	if result.Outcome != OutcomeSucceeded {
		return fmt.Errorf("cannot apply run %s with outcome %s", id, result.Outcome)
	}
	current, err := workspace.Snapshot(ctx, prepared.Source, workspaceOptions(checkpoint.Request.Project, checkpoint.Limits))
	if err != nil {
		return err
	}
	if current != checkpoint.SourceDigest {
		return fmt.Errorf("%w: admitted %s, current %s", ErrSourceDrift, checkpoint.SourceDigest, current)
	}
	changes, digest, err := workspace.Diff(ctx, prepared)
	if err != nil {
		return err
	}
	if digest != result.CandidateDigest {
		return fmt.Errorf("%w: candidate digest differs from terminal result", ErrCorrupt)
	}
	if err := workspace.ValidateChanges(changes, checkpoint.Request.Project.ForbiddenPaths); err != nil {
		return err
	}
	backup := filepath.Join(run.Directory(), fmt.Sprintf("apply-backup-%06d", run.Manifest().Generation+1))
	if err := workspace.Apply(ctx, prepared, backup); err != nil {
		if strings.Contains(err.Error(), "source drift") {
			return fmt.Errorf("%w: %v", ErrSourceDrift, err)
		}
		return err
	}
	return nil
}

func (engine *Engine) reopenCheckpoint(run *store.Run) (checkpoint, workspace.Prepared, error) {
	data, err := run.ReadCheckpoint(engine.limits.MaxOutputBytes)
	if err != nil {
		return checkpoint{}, workspace.Prepared{}, mapStoreError(err)
	}
	var value checkpoint
	if err := strictDecode(data, &value); err != nil {
		return checkpoint{}, workspace.Prepared{}, fmt.Errorf("%w: decode checkpoint: %v", ErrCorrupt, err)
	}
	prepared, err := workspace.Reopen(value.Request.Project.Root, run.Directory(), value.SourceDigest, workspaceOptions(value.Request.Project, value.Limits))
	return value, prepared, err
}

func (engine *Engine) describe(ctx context.Context) (BackendInfo, error) {
	info, err := engine.backend.Describe(ctx)
	if err != nil {
		return BackendInfo{}, fmt.Errorf("describe agentworkflow backend: %w", err)
	}
	info.Name = strings.TrimSpace(info.Name)
	info.Version = strings.TrimSpace(info.Version)
	if err := validateComponent("backend name", info.Name); err != nil || info.Version == "" || strings.TrimSpace(info.ConfigurationDigest) == "" {
		return BackendInfo{}, errors.Join(errors.New("backend returned an invalid identity"), err)
	}
	slices.Sort(info.Capabilities)
	if len(info.Capabilities) != len(slices.Compact(append([]Capability(nil), info.Capabilities...))) {
		return BackendInfo{}, errors.New("backend returned duplicate capabilities")
	}
	required := []Capability{
		CapabilityReadOnly, CapabilityWorkspaceWrite, CapabilityStructuredOutput, CapabilityCancellation,
	}
	for _, capability := range required {
		if !slices.Contains(info.Capabilities, capability) {
			return BackendInfo{}, fmt.Errorf("%w: backend %s lacks %s", ErrUnsupported, info.Name, capability)
		}
	}
	return info, nil
}

func (engine *Engine) readResult(id string) (Result, error) {
	inspection, err := engine.store.Inspect(id, engine.limits.MaxOutputBytes)
	if err != nil {
		return Result{}, mapStoreError(err)
	}
	if inspection.Manifest.ResultPath == "" {
		return Result{}, os.ErrNotExist
	}
	data, err := os.ReadFile(filepath.Join(engine.store.Root(), id, inspection.Manifest.ResultPath))
	if err != nil {
		return Result{}, err
	}
	var result Result
	if err := strictDecode(data, &result); err != nil || result.Schema != "agentworkflow.result/v2" || string(result.RunID) != id {
		return Result{}, fmt.Errorf("%w: invalid result: %v", ErrCorrupt, err)
	}
	return result, nil
}

func (engine *Engine) publishAndClose(run *store.Run, result Result, primary error) error {
	encoded, encodeErr := json.Marshal(result)
	if encodeErr == nil {
		encodeErr = run.PublishResult(encoded, string(result.Outcome), result.Phase, time.Now().UTC())
	}
	if encodeErr == nil && engine.observe != nil {
		engine.observe(Progress{RunID: result.RunID, State: "terminal", Phase: result.Phase, Outcome: result.Outcome})
	}
	closeErr := run.Close()
	return errors.Join(primary, encodeErr, closeErr)
}

func saveCheckpoint(run *store.Run, checkpoint *checkpoint, state, phase, outcome string) error {
	checkpoint.Phase = phase
	encoded, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("encode agentworkflow checkpoint: %w", err)
	}
	if err := run.WriteCheckpoint(encoded, state, phase, outcome, time.Now().UTC()); err != nil {
		return err
	}
	if checkpoint.progress != nil {
		checkpoint.progress(Progress{RunID: RunID(run.ID()), State: state, Phase: phase, Outcome: Outcome(outcome)})
	}
	return nil
}

func normalizeRequest(request Request, limits Limits) (Request, error) {
	request.Task.Objective = strings.TrimSpace(request.Task.Objective)
	if request.Task.Objective == "" {
		return Request{}, errors.New("task objective is required")
	}
	request.Task.SuccessCriteria = compactNonempty(request.Task.SuccessCriteria)
	if len(request.Task.SuccessCriteria) == 0 {
		return Request{}, errors.New("at least one success criterion is required")
	}
	request.Task.Constraints = compactNonempty(request.Task.Constraints)
	request.Task.NonGoals = compactNonempty(request.Task.NonGoals)
	project, err := normalizeProject(request.Project)
	if err != nil {
		return Request{}, err
	}
	request.Project = project
	request.Policy, err = normalizePolicy(request.Policy, limits)
	if err != nil {
		return Request{}, err
	}
	request.Workflow, err = normalizeWorkflow(request.Workflow)
	if err != nil {
		return Request{}, err
	}
	return request, nil
}

func normalizeProject(project Project) (Project, error) {
	root, err := filepath.Abs(project.Root)
	if err != nil || root == string(filepath.Separator) {
		return Project{}, errors.Join(errors.New("project root must be a non-root directory"), err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Project{}, errors.Join(errors.New("project root is not a directory"), err)
	}
	project.Root = root
	if project.Source.Mode == "" {
		project.Source.Mode = SourceDirectoryCopy
	}
	if project.Source.Mode != SourceDirectoryCopy {
		return Project{}, fmt.Errorf("%w: source mode %q", ErrUnsupported, project.Source.Mode)
	}
	project.Source.Exclude, err = normalizeRelativePaths(project.Source.Exclude)
	if err != nil {
		return Project{}, fmt.Errorf("source exclusions: %w", err)
	}
	for _, exclusion := range project.Source.Exclude {
		if exclusion == "." || exclusion == ".spec" || strings.HasPrefix(exclusion, ".spec/") {
			return Project{}, errors.New("source exclusions cannot exclude .spec or its contents")
		}
	}
	project.Instructions, err = normalizeRelativePaths(project.Instructions)
	if err != nil {
		return Project{}, fmt.Errorf("instruction files: %w", err)
	}
	for _, instruction := range project.Instructions {
		info, statErr := os.Lstat(filepath.Join(project.Root, filepath.FromSlash(instruction)))
		if statErr != nil || !info.Mode().IsRegular() {
			return Project{}, errors.Join(fmt.Errorf("instruction file %q is not a regular file", instruction), statErr)
		}
	}
	project.ForbiddenPaths = append(project.ForbiddenPaths, ".spec")
	project.ForbiddenPaths = append(project.ForbiddenPaths, project.Instructions...)
	project.ForbiddenPaths, err = normalizeRelativePaths(project.ForbiddenPaths)
	if err != nil {
		return Project{}, fmt.Errorf("forbidden paths: %w", err)
	}
	project.Checks, err = normalizeChecks(project.Checks)
	if err != nil {
		return Project{}, err
	}
	project.Environment.Allow = compactNonempty(project.Environment.Allow)
	for _, name := range project.Environment.Allow {
		if strings.ContainsRune(name, '=') {
			return Project{}, fmt.Errorf("environment name %q is invalid", name)
		}
	}
	return project, nil
}

func normalizeChecks(checks []Check) ([]Check, error) {
	if len(checks) > 32 {
		return nil, errors.New("project cannot declare more than 32 checks")
	}
	result := append([]Check(nil), checks...)
	seen := make(map[string]struct{}, len(result))
	for index := range result {
		check := &result[index]
		if err := validateComponent("check name", check.Name); err != nil {
			return nil, err
		}
		if _, found := seen[check.Name]; found {
			return nil, fmt.Errorf("check %q is duplicated", check.Name)
		}
		seen[check.Name] = struct{}{}
		if len(check.Command) == 0 || strings.TrimSpace(check.Command[0]) == "" {
			return nil, fmt.Errorf("check %q command is required", check.Name)
		}
		check.Command = append([]string(nil), check.Command...)
		if check.Directory == "" {
			check.Directory = "."
		}
		paths, pathErr := normalizeRelativePaths([]string{check.Directory})
		if pathErr != nil {
			return nil, fmt.Errorf("check %q directory: %w", check.Name, pathErr)
		}
		check.Directory = paths[0]
		if check.Timeout < 0 {
			return nil, fmt.Errorf("check %q timeout cannot be negative", check.Name)
		}
	}
	return result, nil
}

func normalizePolicy(policy Policy, limits Limits) (Policy, error) {
	if policy.Assurance == "" {
		policy.Assurance = AssuranceStandard
	}
	if policy.MaxRepairs < 0 {
		return Policy{}, errors.New("max repairs cannot be negative")
	}
	switch policy.Assurance {
	case AssuranceFast:
		if len(policy.Reviewers) == 0 {
			policy.Reviewers = []string{"correctness"}
		}
	case AssuranceStandard:
		if len(policy.Reviewers) == 0 {
			policy.Reviewers = []string{"correctness", "tests"}
		}
	case AssuranceHigh:
		if len(policy.Reviewers) == 0 {
			policy.Reviewers = []string{"correctness", "failure-concurrency", "security", "maintainability"}
		}
	default:
		return Policy{}, fmt.Errorf("assurance %q is invalid", policy.Assurance)
	}
	policy.Reviewers = compactNonempty(policy.Reviewers)
	if len(policy.Reviewers) == 0 || len(policy.Reviewers) > limits.MaxReviewers {
		return Policy{}, fmt.Errorf("reviewer count must be between 1 and %d", limits.MaxReviewers)
	}
	for _, reviewer := range policy.Reviewers {
		if err := validateComponent("reviewer", reviewer); err != nil {
			return Policy{}, err
		}
	}
	if policy.BlockingSeverity == "" {
		policy.BlockingSeverity = SeverityMedium
	}
	if severityRank(policy.BlockingSeverity) < severityRank(SeverityLow) {
		return Policy{}, fmt.Errorf("blocking severity %q is invalid", policy.BlockingSeverity)
	}
	return policy, nil
}

func validateLimits(limits Limits) error {
	if limits.InvocationTimeout <= 0 || limits.CheckTimeout <= 0 || limits.MaxOutputBytes <= 0 ||
		limits.MaxEvents <= 0 || limits.MaxSourceBytes <= 0 || limits.MaxSourceFiles <= 0 || limits.MaxReviewers <= 0 {
		return errors.New("agentworkflow limits must be positive")
	}
	if limits.MaxOutputBytes > int64(^uint(0)>>1) {
		return errors.New("agentworkflow output limit exceeds addressable memory")
	}
	return nil
}

func workspaceOptions(project Project, limits Limits) workspace.Options {
	return workspace.Options{MaxBytes: limits.MaxSourceBytes, MaxFiles: limits.MaxSourceFiles, Exclude: project.Source.Exclude}
}

func publicChanges(changes []workspace.Change) []Change {
	result := make([]Change, len(changes))
	for index, change := range changes {
		result[index] = Change{Path: change.Path, Kind: change.Kind, Bytes: change.Bytes, Digest: change.Digest}
	}
	return result
}

func sameBackend(left, right BackendInfo) bool {
	return left.Name == right.Name && left.Version == right.Version && left.ConfigurationDigest == right.ConfigurationDigest && slices.Equal(left.Capabilities, right.Capabilities)
}

func mapStoreError(err error) error {
	switch {
	case errors.Is(err, store.ErrCapacity):
		return fmt.Errorf("%w: %v", ErrCapacity, err)
	case errors.Is(err, store.ErrCorrupt):
		return fmt.Errorf("%w: %v", ErrCorrupt, err)
	case errors.Is(err, store.ErrLocked):
		return fmt.Errorf("%w: %v", ErrLocked, err)
	default:
		return err
	}
}

func newRunID() (RunID, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate run identity: %w", err)
	}
	return RunID(time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random)), nil
}

func validateComponent(kind, value string) error {
	if value == "" || value == "." || value == ".." {
		return fmt.Errorf("%s is invalid", kind)
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return fmt.Errorf("%s %q contains an invalid character", kind, value)
	}
	return nil
}

func normalizeRelativePaths(paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if path == "" || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, fmt.Errorf("relative path %q is invalid", path)
		}
		result = append(result, path)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func compactNonempty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityAdvisory:
		return 0
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return -1
	}
}

func isMutationPhase(phase string) bool {
	return phase == "implement" || strings.HasPrefix(phase, "repair-")
}
