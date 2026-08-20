package canary

import (
	"bytes"
	"context"
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

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

const FormatVersion = "umpire3/canary/v1"

var ErrUnsafeRequest = errors.New("unsafe Umpire3 canary request")

type Mode string

const (
	ModeShadow          Mode = "shadow"
	ModeSafeWrite       Mode = "safe-write"
	ModeControlledFault Mode = "controlled-fault"
)

type Approval struct {
	FormatVersion      string        `json:"formatVersion"`
	Identifier         string        `json:"identifier"`
	ApproverIdentity   string        `json:"approverIdentity"`
	ExperimentDigest   string        `json:"experimentDigest"`
	CatalogDigest      string        `json:"catalogDigest"`
	ProfileDigest      string        `json:"profileDigest"`
	Tenant             string        `json:"tenant"`
	Namespace          string        `json:"namespace"`
	Mode               Mode          `json:"mode"`
	AllowedActions     []string      `json:"allowedActions"`
	AllowedFaults      []string      `json:"allowedFaults"`
	DestructiveActions []string      `json:"destructiveActions"`
	AllowWrites        bool          `json:"allowWrites"`
	AllowFaults        bool          `json:"allowFaults"`
	MaxActions         int           `json:"maxActions"`
	MaxFaults          int           `json:"maxFaults"`
	MaxConcurrent      int           `json:"maxConcurrent"`
	MaxRatePerSecond   int           `json:"maxRatePerSecond"`
	MaxDuration        time.Duration `json:"maxDuration"`
	CleanupTimeout     time.Duration `json:"cleanupTimeout"`
	MaxEvidenceBytes   int64         `json:"maxEvidenceBytes"`
	MaxOutputBytes     int64         `json:"maxOutputBytes"`
	ApprovalDigest     string        `json:"approvalDigest"`
}

func Seal(approval Approval) (Approval, error) {
	approval.FormatVersion = FormatVersion
	approval.AllowedActions = sortedUnique(approval.AllowedActions)
	approval.AllowedFaults = sortedUnique(approval.AllowedFaults)
	approval.DestructiveActions = sortedUnique(approval.DestructiveActions)
	approval.ApprovalDigest = ""
	digest, err := approvalDigest(approval)
	if err != nil {
		return Approval{}, err
	}
	approval.ApprovalDigest = digest
	return approval, nil
}

type Request struct {
	Experiment        protocol.Experiment `json:"experiment"`
	Profile           profile.Definition  `json:"profile"`
	Approval          Approval            `json:"approval"`
	WorkerEnvironment []string            `json:"-"`
}

type AuditRecord struct {
	Sequence int    `json:"sequence"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

type RecoveryRecord struct {
	FormatVersion    string            `json:"formatVersion"`
	ApprovalID       string            `json:"approvalID"`
	ApprovalDigest   string            `json:"approvalDigest"`
	ExperimentDigest string            `json:"experimentDigest"`
	Namespace        string            `json:"namespace"`
	Tenant           string            `json:"tenant"`
	Resources        map[string]string `json:"resources"`
	CleanupPending   bool              `json:"cleanupPending"`
}

type Result struct {
	FormatVersion  string                `json:"formatVersion"`
	ApprovalDigest string                `json:"approvalDigest"`
	Runtime        umpire3runtime.Result `json:"runtime"`
	Audit          []AuditRecord         `json:"audit"`
	Recovery       RecoveryRecord        `json:"recovery"`
	PrimaryFailure string                `json:"primaryFailure,omitempty"`
	CleanupFailure string                `json:"cleanupFailure,omitempty"`
	StopReason     string                `json:"stopReason,omitempty"`
	Complete       bool                  `json:"complete"`
}

type Store interface {
	Save(context.Context, RecoveryRecord) error
	Load(context.Context, string) (RecoveryRecord, error)
	Delete(context.Context, string) error
}

type Controller struct {
	Store Store
}

type WorkerOperation string

const (
	OperationExecute WorkerOperation = "execute"
	OperationCleanup WorkerOperation = "cleanup"
)

type WorkerRequest struct {
	FormatVersion string              `json:"formatVersion"`
	Operation     WorkerOperation     `json:"operation"`
	Experiment    protocol.Experiment `json:"experiment"`
	Profile       profile.Definition  `json:"profile"`
	Approval      Approval            `json:"approval"`
	Recovery      RecoveryRecord      `json:"recovery"`
}

type WorkerResponse struct {
	FormatVersion   string                `json:"formatVersion"`
	Result          umpire3runtime.Result `json:"result"`
	Resources       map[string]string     `json:"resources"`
	CleanupComplete bool                  `json:"cleanupComplete"`
}

func (c Controller) Run(ctx context.Context, request Request) (result Result, retErr error) {
	experimentDigest, err := preflight(request, c.Store)
	if err != nil {
		return Result{}, err
	}
	result = Result{FormatVersion: FormatVersion, ApprovalDigest: request.Approval.ApprovalDigest}
	result.Recovery = RecoveryRecord{
		FormatVersion: FormatVersion, ApprovalID: request.Approval.Identifier,
		ApprovalDigest: request.Approval.ApprovalDigest, ExperimentDigest: experimentDigest,
		Namespace: request.Approval.Namespace, Tenant: request.Approval.Tenant,
		Resources: map[string]string{
			"namespace": request.Approval.Namespace,
			"tenant":    request.Approval.Tenant,
		},
		CleanupPending: true,
	}
	if err := c.Store.Save(ctx, result.Recovery); err != nil {
		return Result{}, fmt.Errorf("persist recovery intent: %w", err)
	}
	result.Audit = append(result.Audit, AuditRecord{Sequence: 1, Decision: "recovery-intent-persisted"})

	workerResponse, executeErr := runWorker(ctx, request, OperationExecute, result.Recovery,
		request.Approval.MaxDuration, request.Approval.MaxOutputBytes)
	if executeErr != nil {
		result.PrimaryFailure = errorClass(executeErr)
		result.StopReason = "worker execution failed"
		result.Audit = append(result.Audit, AuditRecord{Sequence: 2, Decision: "stopped", Reason: result.StopReason})
	} else {
		result.Runtime = workerResponse.Result
		for key, value := range workerResponse.Resources {
			if key != "" && value != "" {
				result.Recovery.Resources[key] = value
			}
		}
		if err := c.Store.Save(ctx, result.Recovery); err != nil {
			result.PrimaryFailure = "recovery-store"
			result.StopReason = "recovery metadata persistence failed"
		} else if err := validateWorkerResult(request, experimentDigest, workerResponse.Result); err != nil {
			result.PrimaryFailure = errorClass(err)
			result.StopReason = err.Error()
		} else if workerResponse.Result.Claim.Kind == umpire3runtime.ClaimViolating {
			result.StopReason = "property violation"
		} else {
			result.Audit = append(result.Audit, AuditRecord{Sequence: 2, Decision: "semantic-result-qualified"})
		}
	}

	cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), request.Approval.CleanupTimeout)
	cleanupResponse, cleanupErr := runWorker(cleanupCtx, request, OperationCleanup, result.Recovery,
		request.Approval.CleanupTimeout, request.Approval.MaxOutputBytes)
	cancelCleanup()
	if cleanupErr != nil || !cleanupResponse.CleanupComplete {
		if cleanupErr != nil {
			result.CleanupFailure = errorClass(cleanupErr)
		} else {
			result.CleanupFailure = "cleanup-incomplete"
		}
		if result.StopReason == "" {
			result.StopReason = "cleanup failed"
		}
		result.Audit = append(result.Audit, AuditRecord{Sequence: 3, Decision: "cleanup-pending", Reason: result.CleanupFailure})
	} else {
		result.Recovery.CleanupPending = false
		if err := c.Store.Delete(context.WithoutCancel(ctx), request.Approval.Identifier); err != nil {
			result.CleanupFailure = "recovery-store"
			result.StopReason = "cleanup record deletion failed"
		} else {
			result.Audit = append(result.Audit, AuditRecord{Sequence: 3, Decision: "cleanup-complete"})
		}
	}
	result.Complete = result.StopReason == "" && result.PrimaryFailure == "" && result.CleanupFailure == ""
	return result, nil
}

func (c Controller) ResumeCleanup(ctx context.Context, definition profile.Definition, approval Approval, environment []string) (Result, error) {
	if c.Store == nil {
		return Result{}, errors.New("recovery store is required")
	}
	recovery, err := c.Store.Load(ctx, approval.Identifier)
	if err != nil {
		return Result{}, err
	}
	request := Request{Profile: definition, Approval: approval, WorkerEnvironment: environment}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), approval.CleanupTimeout)
	response, cleanupErr := runWorker(cleanupCtx, request, OperationCleanup, recovery,
		approval.CleanupTimeout, approval.MaxOutputBytes)
	cancel()
	result := Result{
		FormatVersion: FormatVersion, ApprovalDigest: approval.ApprovalDigest, Recovery: recovery,
		Audit: []AuditRecord{{Sequence: 1, Decision: "cleanup-resumed"}},
	}
	if cleanupErr != nil || !response.CleanupComplete {
		result.CleanupFailure = errorClass(cleanupErr)
		if cleanupErr == nil {
			result.CleanupFailure = "cleanup-incomplete"
		}
		result.StopReason = "cleanup failed"
		return result, nil
	}
	result.Recovery.CleanupPending = false
	if err := c.Store.Delete(context.WithoutCancel(ctx), approval.Identifier); err != nil {
		return Result{}, err
	}
	result.Complete = true
	result.Audit = append(result.Audit, AuditRecord{Sequence: 2, Decision: "cleanup-complete"})
	return result, nil
}

func preflight(request Request, store Store) (string, error) {
	if store == nil {
		return "", fmt.Errorf("%w: recovery store is required", ErrUnsafeRequest)
	}
	if err := request.Experiment.Validate(); err != nil {
		return "", fmt.Errorf("%w: invalid experiment: %v", ErrUnsafeRequest, err)
	}
	if request.Profile.Kind != profile.KindCanary || !request.Profile.Environment.HardExecutionBudget ||
		len(request.Profile.WorkerCommand()) == 0 {
		return "", fmt.Errorf("%w: production canary requires a hard-budget canary profile", ErrUnsafeRequest)
	}
	profileDigest, err := request.Profile.Digest()
	if err != nil {
		return "", err
	}
	experimentDigest, err := request.Experiment.Digest()
	if err != nil {
		return "", err
	}
	approval := request.Approval
	sealed, err := Seal(approval)
	if err != nil {
		return "", err
	}
	if approval.FormatVersion != FormatVersion || sealed.ApprovalDigest != approval.ApprovalDigest {
		return "", fmt.Errorf("%w: approval digest is invalid", ErrUnsafeRequest)
	}
	if approval.Identifier == "" || approval.ApproverIdentity == "" || approval.Tenant == "" || approval.Namespace == "" ||
		approval.Namespace != request.Profile.Namespace || approval.ExperimentDigest != experimentDigest ||
		approval.CatalogDigest != request.Experiment.Model.CatalogHash || approval.ProfileDigest != profileDigest {
		return "", fmt.Errorf("%w: approval identity, isolation, or immutable digests do not match", ErrUnsafeRequest)
	}
	if approval.MaxActions <= 0 || approval.MaxFaults < 0 || approval.MaxConcurrent <= 0 ||
		approval.MaxRatePerSecond <= 0 || approval.MaxDuration <= 0 || approval.CleanupTimeout <= 0 ||
		approval.MaxEvidenceBytes <= 0 || approval.MaxOutputBytes <= 0 {
		return "", fmt.Errorf("%w: complete count, rate, duration, evidence, output, and cleanup budgets are required", ErrUnsafeRequest)
	}
	if len(request.Experiment.Actions) > approval.MaxActions || len(request.Experiment.Faults) > approval.MaxFaults {
		return "", fmt.Errorf("%w: experiment exceeds approved action or fault count", ErrUnsafeRequest)
	}
	if approval.Mode == ModeShadow && (approval.AllowWrites || approval.AllowFaults) {
		return "", fmt.Errorf("%w: shadow mode cannot authorize writes or faults", ErrUnsafeRequest)
	}
	if approval.Mode == ModeSafeWrite && !approval.AllowWrites {
		return "", fmt.Errorf("%w: safe-write mode requires explicit write authority", ErrUnsafeRequest)
	}
	if approval.Mode == ModeControlledFault && (!approval.AllowWrites || !approval.AllowFaults) {
		return "", fmt.Errorf("%w: controlled-fault mode requires explicit write and fault authority", ErrUnsafeRequest)
	}
	if approval.Mode != ModeShadow && approval.Mode != ModeSafeWrite && approval.Mode != ModeControlledFault {
		return "", fmt.Errorf("%w: unknown canary mode", ErrUnsafeRequest)
	}
	for _, action := range request.Experiment.Actions {
		if !slices.Contains(approval.AllowedActions, action.Kind) {
			return "", fmt.Errorf("%w: action %q is not allowlisted", ErrUnsafeRequest, action.Kind)
		}
		if slices.Contains(approval.DestructiveActions, action.Kind) && !approval.AllowWrites {
			return "", fmt.Errorf("%w: destructive action %q is not authorized", ErrUnsafeRequest, action.Kind)
		}
	}
	for _, fault := range request.Experiment.Faults {
		if !approval.AllowFaults || !slices.Contains(approval.AllowedFaults, fault.Kind) {
			return "", fmt.Errorf("%w: fault %q is not authorized", ErrUnsafeRequest, fault.Kind)
		}
	}
	return experimentDigest, nil
}

func runWorker(
	ctx context.Context,
	request Request,
	operation WorkerOperation,
	recovery RecoveryRecord,
	timeout time.Duration,
	maxOutputBytes int64,
) (WorkerResponse, error) {
	encoded, err := json.Marshal(WorkerRequest{
		FormatVersion: FormatVersion, Operation: operation, Experiment: request.Experiment, Profile: request.Profile,
		Approval: request.Approval, Recovery: recovery,
	})
	if err != nil {
		return WorkerResponse{}, fmt.Errorf("encode worker request: %w", err)
	}
	worker, err := process.Run(ctx, process.Request{
		Command: request.Profile.WorkerCommand(), Environment: request.WorkerEnvironment,
		Input: encoded, Timeout: timeout, MaxOutputBytes: maxOutputBytes,
	})
	if err != nil {
		return WorkerResponse{}, err
	}
	var response WorkerResponse
	if err := decodeStrict(worker.Output, &response); err != nil {
		return WorkerResponse{}, fmt.Errorf("decode worker response: %w", err)
	}
	if response.FormatVersion != FormatVersion {
		return WorkerResponse{}, errors.New("worker response format mismatch")
	}
	return response, nil
}

func validateWorkerResult(request Request, digest string, result umpire3runtime.Result) error {
	if result.ExperimentDigest != digest || result.Claim.Property != request.Experiment.Property.Identifier {
		return errors.New("worker semantic identity drift")
	}
	if result.FormatVersion != umpire3runtime.ResultFormatVersion {
		return fmt.Errorf("worker returned unsupported runtime result format %q", result.FormatVersion)
	}
	if err := result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate worker result assurance: %w", err)
	}
	if result.Environment.BuildID != request.Profile.Environment.BuildID ||
		result.Environment.ConfigurationIdentity != request.Profile.Environment.ConfigurationIdentity {
		return errors.New("worker build or configuration attestation drift")
	}
	if result.Claim.Kind != umpire3runtime.ClaimConforming && result.Claim.Kind != umpire3runtime.ClaimViolating {
		return fmt.Errorf("worker stopped with %s", result.Claim.Kind)
	}
	if err := result.Evidence.Validate(); err != nil {
		return fmt.Errorf("evidence failure: %w", err)
	}
	if len(result.Evidence.Facts) == 0 || len(result.Evidence.Claims) == 0 || len(result.Evidence.Omissions) != 0 {
		return errors.New("evidence loss or omission")
	}
	claimFound := false
	for _, claim := range result.Evidence.Claims {
		if claim.Property == result.Claim.Property && claim.Verdict == string(result.Claim.Kind) {
			claimFound = true
		}
	}
	if !claimFound {
		return errors.New("evidence graph does not support the semantic claim")
	}
	encoded, err := result.Evidence.CanonicalJSON()
	if err != nil {
		return err
	}
	if int64(len(encoded)) > request.Approval.MaxEvidenceBytes {
		return errors.New("evidence budget exhausted")
	}
	return nil
}

func approvalDigest(approval Approval) (string, error) {
	encoded, err := json.Marshal(approval)
	if err != nil {
		return "", fmt.Errorf("encode approval: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("worker response contains trailing data")
	}
	return nil
}

func errorClass(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, process.ErrDeadline):
		return "deadline"
	case errors.Is(err, process.ErrOutputLimit):
		return "output-limit"
	default:
		return "infrastructure"
	}
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	slices.Sort(result)
	return slices.Compact(result)
}

type MemoryStore struct {
	mu      sync.Mutex
	records map[string]RecoveryRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]RecoveryRecord)}
}

func (s *MemoryStore) Save(_ context.Context, record RecoveryRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ApprovalID] = cloneRecovery(record)
	return nil
}

func (s *MemoryStore) Load(_ context.Context, identifier string) (RecoveryRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[identifier]
	if !exists {
		return RecoveryRecord{}, os.ErrNotExist
	}
	return cloneRecovery(record), nil
}

func (s *MemoryStore) Delete(_ context.Context, identifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, identifier)
	return nil
}

type FileStore struct {
	root string
	mu   sync.Mutex
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (s *FileStore) Save(ctx context.Context, record RecoveryRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(record.ApprovalID)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(s.root, ".recovery-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(encoded); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func (s *FileStore) Load(ctx context.Context, identifier string) (RecoveryRecord, error) {
	if err := ctx.Err(); err != nil {
		return RecoveryRecord{}, err
	}
	path, err := s.path(identifier)
	if err != nil {
		return RecoveryRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		return RecoveryRecord{}, err
	}
	var record RecoveryRecord
	if err := decodeStrict(data, &record); err != nil {
		return RecoveryRecord{}, err
	}
	return record, nil
}

func (s *FileStore) Delete(ctx context.Context, identifier string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(identifier)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *FileStore) path(identifier string) (string, error) {
	if s.root == "" || identifier == "" || filepath.Base(identifier) != identifier || strings.Contains(identifier, "..") {
		return "", errors.New("safe recovery root and approval identifier are required")
	}
	return filepath.Join(s.root, identifier+".json"), nil
}

func cloneRecovery(record RecoveryRecord) RecoveryRecord {
	clone := record
	clone.Resources = make(map[string]string, len(record.Resources))
	for key, value := range record.Resources {
		clone.Resources[key] = value
	}
	return clone
}
