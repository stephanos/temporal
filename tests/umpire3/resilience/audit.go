package resilience

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tests/umpire3/canary"
	"go.temporal.io/server/tests/umpire3/internal/artifact"
	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/replay"
)

const FormatVersion = "umpire3/resilience-audit/v1"

//go:embed results/control-plane.audit.json
var defaultAuditJSON []byte

type Report struct {
	FormatVersion            string `json:"formatVersion"`
	HostileDecodeRejected    bool   `json:"hostileDecodeRejected"`
	ArtifactLimitRejected    bool   `json:"artifactLimitRejected"`
	CardinalityLimitRejected bool   `json:"cardinalityLimitRejected"`
	SecretRedacted           bool   `json:"secretRedacted"`
	EnvironmentIsolated      bool   `json:"environmentIsolated"`
	DeadlineEnforced         bool   `json:"deadlineEnforced"`
	OutputBoundEnforced      bool   `json:"outputBoundEnforced"`
	ResourceLimitsApplied    bool   `json:"resourceLimitsApplied"`
	TransactionalPublication bool   `json:"transactionalPublication"`
	WorkerCrashRecovered     bool   `json:"workerCrashRecovered"`
	RecoveryRecordDurable    bool   `json:"recoveryRecordDurable"`
	ArtifactDigest           string `json:"artifactDigest"`
}

func RunAudit(ctx context.Context) (Report, error) {
	report := Report{FormatVersion: FormatVersion}
	report.HostileDecodeRejected = hostileDecodeRejected()
	report.ArtifactLimitRejected = artifactLimitRejected()
	report.CardinalityLimitRejected = cardinalityLimitRejected()
	report.SecretRedacted = secretRedacted()

	var err error
	if report.EnvironmentIsolated, err = environmentIsolated(ctx); err != nil {
		return Report{}, err
	}
	if report.DeadlineEnforced, err = deadlineEnforced(ctx); err != nil {
		return Report{}, err
	}
	if report.OutputBoundEnforced, err = outputBoundEnforced(ctx); err != nil {
		return Report{}, err
	}
	if report.ResourceLimitsApplied, err = resourceLimitsApplied(ctx); err != nil {
		return Report{}, err
	}
	if report.TransactionalPublication, err = transactionalPublication(); err != nil {
		return Report{}, err
	}
	if report.WorkerCrashRecovered, err = workerCrashRecovered(ctx); err != nil {
		return Report{}, err
	}
	if report.RecoveryRecordDurable, err = recoveryRecordDurable(ctx); err != nil {
		return Report{}, err
	}
	report.ArtifactDigest = report.computedDigest()
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func DefaultAudit() (Report, error) {
	return DecodeAudit(defaultAuditJSON, protocol.DefaultDecodeLimit)
}

func DecodeAudit(encoded []byte, limit int64) (Report, error) {
	if limit <= 0 || int64(len(encoded)) > limit {
		return Report{}, errors.New("resilience audit exceeds decode limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode resilience audit: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Report{}, errors.New("resilience audit must contain one JSON document")
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

func (r Report) Validate() error {
	if r.FormatVersion != FormatVersion || !r.HostileDecodeRejected || !r.ArtifactLimitRejected ||
		!r.CardinalityLimitRejected || !r.SecretRedacted || !r.EnvironmentIsolated ||
		!r.DeadlineEnforced || !r.OutputBoundEnforced || !r.ResourceLimitsApplied ||
		!r.TransactionalPublication || !r.WorkerCrashRecovered || !r.RecoveryRecordDurable {
		return errors.New("resilience audit requires every hostile-input, isolation, and recovery check")
	}
	if !validDigest(r.ArtifactDigest) || r.ArtifactDigest != r.computedDigest() {
		return errors.New("resilience audit digest does not match its contents")
	}
	return nil
}

func hostileDecodeRejected() bool {
	_, err := protocol.DecodeReleaseManifest([]byte(`{"unknown":true}`))
	return err != nil && strings.Contains(err.Error(), "unknown field")
}

func artifactLimitRejected() bool {
	_, err := replay.DecodeBundle(make([]byte, 65), 64)
	return err != nil && strings.Contains(err.Error(), "decode limit")
}

func cardinalityLimitRejected() bool {
	view, found, err := protocol.DefaultFirstOrderView(protocol.TargetIDNexusCancellation, "sound")
	if err != nil || !found {
		return false
	}
	view.Bounds.ConcreteStateLimit = protocol.MaxFirstOrderConcreteStateLimit + 1
	return view.Validate() != nil
}

func secretRedacted() bool {
	const secret = "umpire3-resilience-secret"
	definition, err := profile.Define(profile.Remote(
		"https://temporal.example", secret, "build", "namespace", "queue"))
	if err != nil {
		return false
	}
	encoded, err := json.Marshal(definition)
	return err == nil && !bytes.Contains(encoded, []byte(secret)) && !strings.Contains(definition.String(), secret)
}

func environmentIsolated(ctx context.Context) (bool, error) {
	result, err := process.Run(ctx, process.Request{
		Command: []string{"/usr/bin/env"}, Environment: []string{"UMPIRE3_RESILIENCE_VISIBLE=allowed"},
		Timeout: time.Second, MaxOutputBytes: 1024,
	})
	if err != nil {
		return false, fmt.Errorf("audit process environment isolation: %w", err)
	}
	return string(result.Output) == "UMPIRE3_RESILIENCE_VISIBLE=allowed\n", nil
}

func deadlineEnforced(ctx context.Context) (bool, error) {
	result, err := process.Run(ctx, process.Request{
		Command: []string{"/bin/sleep", "60"}, Timeout: 50 * time.Millisecond, MaxOutputBytes: 64,
	})
	if !errors.Is(err, process.ErrDeadline) {
		return false, fmt.Errorf("audit process deadline: %w", err)
	}
	return result.TimedOut, nil
}

func outputBoundEnforced(ctx context.Context) (bool, error) {
	result, err := process.Run(ctx, process.Request{
		Command: []string{"/bin/sh", "-c", "while :; do printf 0123456789abcdef; done"},
		Timeout: time.Second, MaxOutputBytes: 8,
	})
	if !errors.Is(err, process.ErrOutputLimit) {
		return false, fmt.Errorf("audit process output limit: %w", err)
	}
	return len(result.Output) == 8, nil
}

func resourceLimitsApplied(ctx context.Context) (bool, error) {
	result, err := process.Run(ctx, process.Request{
		Command: []string{"/bin/sh", "-c", "ulimit -t"}, Timeout: time.Second, MaxOutputBytes: 64,
		Limits: process.Limits{CPUSeconds: 1, MemoryBytes: 64 << 20},
	})
	if err != nil {
		return false, fmt.Errorf("audit process resource limits: %w", err)
	}
	return string(result.Output) == "1\n", nil
}

func transactionalPublication() (bool, error) {
	directory, err := os.MkdirTemp("", "umpire3-resilience-artifact-")
	if err != nil {
		return false, fmt.Errorf("create resilience artifact directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	path := filepath.Join(directory, "nested", "result.json")
	if err := artifact.Publish(path, []byte("first")); err != nil {
		return false, err
	}
	if err := artifact.Publish(path, []byte("second")); err != nil {
		return false, err
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	partial, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".umpire3-artifact-*"))
	if err != nil {
		return false, err
	}
	return string(encoded) == "second" && info.Mode().Perm() == 0o600 && len(partial) == 0, nil
}

func workerCrashRecovered(ctx context.Context) (bool, error) {
	supervisor, err := process.NewSupervisor(process.Request{
		Command: []string{"/bin/sh", "-c", "printf ready; exec /bin/sleep 60"},
		Timeout: time.Second, MaxOutputBytes: 64,
	})
	if err != nil {
		return false, err
	}
	first, err := supervisor.Start(ctx)
	if err != nil {
		return false, err
	}
	if err := waitForOutput(ctx, supervisor, "ready"); err != nil {
		_, _ = supervisor.Stop(context.Background())
		return false, err
	}
	crashed, err := supervisor.Crash(ctx)
	if err != nil {
		return false, err
	}
	second, err := supervisor.Restart(ctx)
	if err != nil {
		return false, err
	}
	if err := waitForOutput(ctx, supervisor, "ready"); err != nil {
		_, _ = supervisor.Stop(context.Background())
		return false, err
	}
	stopped, err := supervisor.Stop(ctx)
	if err != nil {
		return false, err
	}
	return first.Generation == 1 && crashed.Termination == process.TerminationCrash &&
		second.Generation == 2 && first.PID != second.PID && stopped.Termination == process.TerminationStop, nil
}

func waitForOutput(ctx context.Context, supervisor *process.Supervisor, expected string) error {
	deadline, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if string(supervisor.Snapshot().Output) == expected {
			return nil
		}
		select {
		case <-deadline.Done():
			return errors.New("supervised resilience worker did not become ready")
		case <-ticker.C:
		}
	}
}

func recoveryRecordDurable(ctx context.Context) (bool, error) {
	directory, err := os.MkdirTemp("", "umpire3-resilience-recovery-")
	if err != nil {
		return false, err
	}
	defer func() { _ = os.RemoveAll(directory) }()
	store := canary.NewFileStore(directory)
	record := canary.RecoveryRecord{
		FormatVersion: canary.FormatVersion, ApprovalID: "resilience-audit",
		ApprovalDigest:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExperimentDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Namespace:        "isolated", Tenant: "audit", Resources: map[string]string{"namespace": "isolated"},
		CleanupPending: true,
	}
	if err := store.Save(ctx, record); err != nil {
		return false, err
	}
	loaded, err := store.Load(ctx, record.ApprovalID)
	if err != nil {
		return false, err
	}
	if err := store.Delete(ctx, record.ApprovalID); err != nil {
		return false, err
	}
	_, err = store.Load(ctx, record.ApprovalID)
	return loaded.ApprovalDigest == record.ApprovalDigest && loaded.CleanupPending && errors.Is(err, os.ErrNotExist), nil
}

func (r Report) computedDigest() string {
	r.ArtifactDigest = ""
	encoded, _ := json.Marshal(r)
	return digest(encoded)
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
