package finite

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

const (
	BenchmarkFormatVersion = "umpire3/native-benchmark/v1"
	benchmarkEvidenceClass = "performance-measurement"
	benchmarkReplicas      = 10
)

type BenchmarkOptions struct {
	ParallelWorkers int
	Limits          SearchLimits
	CheckerCommand  []string
}

type BenchmarkRuntime struct {
	GoVersion     string `json:"goVersion"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	CheckerDigest string `json:"checkerDigest"`
}

type BenchmarkCertificate struct {
	Digest               string `json:"digest"`
	Bytes                int    `json:"bytes"`
	ExpandedStates       int    `json:"expandedStates"`
	RepresentativeStates int    `json:"representativeStates"`
	Transitions          int    `json:"transitions"`
	StateBytes           int    `json:"stateBytes"`
	MaxDepth             int    `json:"maxDepth"`
}

type SearchMeasurement struct {
	Workers           int    `json:"workers"`
	DurationNanos     int64  `json:"durationNanos"`
	PeakHeapBytes     uint64 `json:"peakHeapBytes"`
	CertificateDigest string `json:"certificateDigest"`
}

type LeanCheckMeasurement struct {
	DurationNanos   int64  `json:"durationNanos"`
	PeakMemoryBytes int64  `json:"peakMemoryBytes"`
	ReceiptBytes    int    `json:"receiptBytes"`
	ReceiptDigest   string `json:"receiptDigest"`
}

type RecoveryMeasurement struct {
	InterruptedWorkers          int    `json:"interruptedWorkers"`
	ResumeWorkers               int    `json:"resumeWorkers"`
	CheckpointDigest            string `json:"checkpointDigest"`
	CheckpointBytes             int    `json:"checkpointBytes"`
	CompletedDepth              int    `json:"completedDepth"`
	ResumedCertificateDigest    string `json:"resumedCertificateDigest"`
	MatchesUninterrupted        bool   `json:"matchesUninterrupted"`
	PartialPublicationRecovered bool   `json:"partialPublicationRecovered"`
}

type BenchmarkReport struct {
	FormatVersion string                     `json:"formatVersion"`
	EvidenceClass string                     `json:"evidenceClass"`
	TrustBadge    protocolcatalog.TrustBadge `json:"trustBadge"`
	Target        protocolcatalog.TargetID   `json:"target"`
	Property      protocolcatalog.PropertyID `json:"property"`
	World         string                     `json:"world"`
	Variant       string                     `json:"variant"`
	SemanticHash  string                     `json:"semanticHash"`
	ViewDigest    string                     `json:"viewDigest"`
	Replicas      int                        `json:"replicas"`
	Runtime       BenchmarkRuntime           `json:"runtime"`
	Certificate   BenchmarkCertificate       `json:"certificate"`
	SearchRuns    []SearchMeasurement        `json:"searchRuns"`
	LeanCheck     LeanCheckMeasurement       `json:"leanCheck"`
	Recovery      RecoveryMeasurement        `json:"recovery"`
	Digest        string                     `json:"digest"`
}

type certificateChecker func(
	context.Context,
	[]string,
	protocolchecker.FirstOrderView,
	Certificate,
) (Receipt, CertificateCheckMeasurement, error)

func Benchmark(
	ctx context.Context,
	view protocolchecker.FirstOrderView,
	options BenchmarkOptions,
) (BenchmarkReport, Certificate, Receipt, error) {
	return runBenchmark(ctx, view, options, MeasureCertificateCheck)
}

func runBenchmark(
	ctx context.Context,
	view protocolchecker.FirstOrderView,
	options BenchmarkOptions,
	checker certificateChecker,
) (BenchmarkReport, Certificate, Receipt, error) {
	if options.ParallelWorkers <= 1 || len(options.CheckerCommand) == 0 || checker == nil {
		return BenchmarkReport{}, Certificate{}, Receipt{},
			errors.New("parallel workers and a canonical Lean checker are required")
	}
	checkerDigest, err := executableDigest(options.CheckerCommand[0])
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	serial, serialMeasurement, err := measureSearch(ctx, view, 1, options.Limits)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, fmt.Errorf("measure serial native search: %w", err)
	}
	parallel, parallelMeasurement, err := measureSearch(
		ctx, view, options.ParallelWorkers, options.Limits)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, fmt.Errorf("measure parallel native search: %w", err)
	}
	if serial.Digest != parallel.Digest {
		return BenchmarkReport{}, Certificate{}, Receipt{},
			errors.New("native certificate changed with worker count")
	}
	recovery, err := measureRecovery(ctx, view, options.ParallelWorkers, options.Limits, parallel)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	receipt, checkMeasurement, err := checker(ctx, options.CheckerCommand, view, parallel)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, fmt.Errorf("measure Lean certificate check: %w", err)
	}
	certificateJSON, err := parallel.CanonicalJSON(view)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	receiptJSON, err := receipt.CanonicalJSON(parallel)
	if err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	report := BenchmarkReport{
		FormatVersion: BenchmarkFormatVersion, EvidenceClass: benchmarkEvidenceClass,
		TrustBadge: protocolcatalog.TrustBadgeTestedInstance,
		Target:     parallel.Target, Property: parallel.Property, World: parallel.World,
		Variant: parallel.Variant, SemanticHash: parallel.SemanticHash,
		ViewDigest: parallel.ViewDigest, Replicas: benchmarkReplicas,
		Runtime: BenchmarkRuntime{
			GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
			CheckerDigest: checkerDigest,
		},
		Certificate: BenchmarkCertificate{
			Digest: parallel.Digest, Bytes: len(certificateJSON),
			ExpandedStates:       parallel.Statistics.ExpandedStates,
			RepresentativeStates: parallel.Statistics.RepresentativeStates,
			Transitions:          parallel.Statistics.Transitions, StateBytes: parallel.Statistics.StateBytes,
			MaxDepth: parallel.Statistics.MaxDepth,
		},
		SearchRuns: []SearchMeasurement{serialMeasurement, parallelMeasurement},
		LeanCheck: LeanCheckMeasurement{
			DurationNanos:   checkMeasurement.DurationNanos,
			PeakMemoryBytes: checkMeasurement.PeakMemoryBytes,
			ReceiptBytes:    len(receiptJSON), ReceiptDigest: digest(receiptJSON),
		},
		Recovery: recovery,
	}
	if err := report.seal(); err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	if err := report.Validate(view, parallel, receipt); err != nil {
		return BenchmarkReport{}, Certificate{}, Receipt{}, err
	}
	return report, parallel, receipt, nil
}

func measureSearch(
	ctx context.Context,
	view protocolchecker.FirstOrderView,
	workers int,
	limits SearchLimits,
) (Certificate, SearchMeasurement, error) {
	peakHeap := currentHeapBytes()
	options := Options{
		Workers: workers, Replicas: benchmarkReplicas, Limits: limits,
		Checkpoint: func(Checkpoint) error {
			peakHeap = max(peakHeap, currentHeapBytes())
			return nil
		},
	}
	started := time.Now()
	certificate, err := Produce(ctx, view, options, nil)
	duration := time.Since(started).Nanoseconds()
	peakHeap = max(peakHeap, currentHeapBytes())
	if err != nil {
		return Certificate{}, SearchMeasurement{}, err
	}
	return certificate, SearchMeasurement{
		Workers: workers, DurationNanos: duration, PeakHeapBytes: peakHeap,
		CertificateDigest: certificate.Digest,
	}, nil
}

func measureRecovery(
	ctx context.Context,
	view protocolchecker.FirstOrderView,
	parallelWorkers int,
	limits SearchLimits,
	uninterrupted Certificate,
) (RecoveryMeasurement, error) {
	directory, err := os.MkdirTemp("", "umpire3-native-benchmark-")
	if err != nil {
		return RecoveryMeasurement{}, fmt.Errorf("create native benchmark directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	checkpointPath := filepath.Join(directory, "checkpoint.json")
	interruptedWorkers := min(3, parallelWorkers)
	interrupted := errors.New("native benchmark interruption")
	options := Options{
		Workers: interruptedWorkers, Replicas: benchmarkReplicas, Limits: limits,
		Checkpoint: func(checkpoint Checkpoint) error {
			if err := SaveCheckpoint(checkpointPath, checkpoint); err != nil {
				return err
			}
			if checkpoint.CompletedDepth >= 1 {
				return interrupted
			}
			return nil
		},
	}
	if _, err := Produce(ctx, view, options, nil); !errors.Is(err, interrupted) {
		return RecoveryMeasurement{}, fmt.Errorf("interrupt checkpointed native benchmark: %w", err)
	}
	checkpoint, err := LoadCheckpoint(checkpointPath, protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return RecoveryMeasurement{}, err
	}
	checkpointJSON, err := checkpoint.CanonicalJSON()
	if err != nil {
		return RecoveryMeasurement{}, err
	}
	partialRecovered, err := verifyPartialPublicationRecovery(checkpointPath, checkpoint.Digest)
	if err != nil {
		return RecoveryMeasurement{}, err
	}
	resumed, err := Produce(ctx, view, Options{
		Workers: parallelWorkers, Replicas: benchmarkReplicas, Limits: limits,
	}, &checkpoint)
	if err != nil {
		return RecoveryMeasurement{}, fmt.Errorf("resume checkpointed native benchmark: %w", err)
	}
	return RecoveryMeasurement{
		InterruptedWorkers: interruptedWorkers, ResumeWorkers: parallelWorkers,
		CheckpointDigest: checkpoint.Digest, CheckpointBytes: len(checkpointJSON),
		CompletedDepth: checkpoint.CompletedDepth, ResumedCertificateDigest: resumed.Digest,
		MatchesUninterrupted:        resumed.Digest == uninterrupted.Digest,
		PartialPublicationRecovered: partialRecovered,
	}, nil
}

func verifyPartialPublicationRecovery(checkpointPath string, expectedDigest string) (bool, error) {
	checkpointJSON, err := os.ReadFile(checkpointPath)
	if err != nil {
		return false, fmt.Errorf("read published native checkpoint: %w", err)
	}
	partial, err := os.CreateTemp(filepath.Dir(checkpointPath), ".umpire3-checkpoint-partial-*")
	if err != nil {
		return false, fmt.Errorf("create partial native checkpoint: %w", err)
	}
	partialPath := partial.Name()
	defer func() { _ = os.Remove(partialPath) }()
	if err := partial.Chmod(0o600); err != nil {
		_ = partial.Close()
		return false, fmt.Errorf("secure partial native checkpoint: %w", err)
	}
	if _, err := partial.Write(checkpointJSON[:max(1, len(checkpointJSON)/2)]); err != nil {
		_ = partial.Close()
		return false, fmt.Errorf("write partial native checkpoint: %w", err)
	}
	if err := partial.Close(); err != nil {
		return false, fmt.Errorf("close partial native checkpoint: %w", err)
	}
	loaded, err := LoadCheckpoint(checkpointPath, protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return false, fmt.Errorf("recover published native checkpoint: %w", err)
	}
	return loaded.Digest == expectedDigest, nil
}

func currentHeapBytes() uint64 {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return memory.HeapAlloc
}

func executableDigest(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("resolve native checker executable: %w", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open native checker executable: %w", err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return "", fmt.Errorf("hash native checker executable: %w", errors.Join(copyErr, closeErr))
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func DecodeBenchmarkReport(
	input io.Reader,
	limit int64,
	view protocolchecker.FirstOrderView,
	certificate Certificate,
	receipt Receipt,
) (BenchmarkReport, error) {
	var report BenchmarkReport
	if err := decodeStrict(input, limit, &report); err != nil {
		return BenchmarkReport{}, fmt.Errorf("decode native benchmark report: %w", err)
	}
	if err := report.Validate(view, certificate, receipt); err != nil {
		return BenchmarkReport{}, err
	}
	return report, nil
}

func (r BenchmarkReport) CanonicalJSON(
	view protocolchecker.FirstOrderView,
	certificate Certificate,
	receipt Receipt,
) ([]byte, error) {
	if err := r.Validate(view, certificate, receipt); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

func (r *BenchmarkReport) seal() error {
	r.Digest = ""
	encoded, err := json.Marshal(r)
	if err != nil {
		return err
	}
	r.Digest = digest(encoded)
	return nil
}

func (r BenchmarkReport) Validate(
	view protocolchecker.FirstOrderView,
	certificate Certificate,
	receipt Receipt,
) error {
	if err := certificate.Validate(view); err != nil {
		return err
	}
	if err := receipt.Validate(certificate); err != nil {
		return err
	}
	viewDigest, err := firstOrderViewDigest(view)
	if err != nil {
		return err
	}
	certificateJSON, err := certificate.CanonicalJSON(view)
	if err != nil {
		return err
	}
	receiptJSON, err := receipt.CanonicalJSON(certificate)
	if err != nil {
		return err
	}
	if r.FormatVersion != BenchmarkFormatVersion || r.EvidenceClass != benchmarkEvidenceClass ||
		r.TrustBadge != protocolcatalog.TrustBadgeTestedInstance || r.Target != view.Target ||
		r.Property != view.Property || r.World != view.World || r.Variant != view.Variant ||
		r.SemanticHash != view.SemanticHash || r.ViewDigest != viewDigest || r.Replicas != benchmarkReplicas {
		return errors.New("native benchmark identity and tested-instance scope are required")
	}
	if r.Runtime.GoVersion == "" || r.Runtime.GOOS == "" || r.Runtime.GOARCH == "" ||
		!digestPattern.MatchString(r.Runtime.CheckerDigest) {
		return errors.New("native benchmark runtime and checker identity are required")
	}
	if r.Certificate != (BenchmarkCertificate{
		Digest: certificate.Digest, Bytes: len(certificateJSON),
		ExpandedStates:       certificate.Statistics.ExpandedStates,
		RepresentativeStates: certificate.Statistics.RepresentativeStates,
		Transitions:          certificate.Statistics.Transitions, StateBytes: certificate.Statistics.StateBytes,
		MaxDepth: certificate.Statistics.MaxDepth,
	}) || r.Certificate.ExpandedStates != r.Certificate.RepresentativeStates*benchmarkReplicas {
		return errors.New("native benchmark certificate scope does not prove the recorded 10x search")
	}
	if len(r.SearchRuns) != 2 || r.SearchRuns[0].Workers != 1 || r.SearchRuns[1].Workers <= 1 {
		return errors.New("native benchmark requires serial and parallel search measurements")
	}
	for _, measurement := range r.SearchRuns {
		if measurement.DurationNanos <= 0 || measurement.PeakHeapBytes == 0 ||
			measurement.CertificateDigest != certificate.Digest {
			return errors.New("native benchmark search measurements must retain deterministic output and costs")
		}
	}
	if r.LeanCheck.DurationNanos <= 0 || r.LeanCheck.PeakMemoryBytes <= 0 ||
		r.LeanCheck.ReceiptBytes != len(receiptJSON) || r.LeanCheck.ReceiptDigest != digest(receiptJSON) {
		return errors.New("native benchmark requires a measured, source-bound Lean certificate check")
	}
	if r.Recovery.InterruptedWorkers <= 0 || r.Recovery.ResumeWorkers != r.SearchRuns[1].Workers ||
		!digestPattern.MatchString(r.Recovery.CheckpointDigest) || r.Recovery.CheckpointBytes <= 0 ||
		r.Recovery.CompletedDepth < 1 || r.Recovery.ResumedCertificateDigest != certificate.Digest ||
		!r.Recovery.MatchesUninterrupted || !r.Recovery.PartialPublicationRecovered {
		return errors.New("native benchmark requires checkpoint resume and partial-publication recovery")
	}
	if !digestPattern.MatchString(r.Digest) {
		return errors.New("native benchmark report digest is required")
	}
	expected := r
	if err := expected.seal(); err != nil || expected.Digest != r.Digest {
		return errors.New("native benchmark report digest does not match its contents")
	}
	return nil
}
