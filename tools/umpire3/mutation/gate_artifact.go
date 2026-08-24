package mutation

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"slices"
	"strings"

	umpire3execution "go.temporal.io/server/tools/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const MutationGateReportFormatVersion = "umpire3/mutation-gate-report/v2"

//go:embed testdata/retained/cross-layer-mutation.audit.json
var defaultApprovedMutationAuditJSON []byte

func DefaultApprovedMutationAudit(source protocolexperiment.Experiment) (MutationGateReport, error) {
	return DecodeMutationGateReport(defaultApprovedMutationAuditJSON, source)
}

func DefaultApprovedMutationAuditForBinding(
	experimentID string,
	property string,
	digest string,
) (MutationGateReport, error) {
	report, err := decodeMutationGateReport(defaultApprovedMutationAuditJSON)
	if err != nil {
		return MutationGateReport{}, err
	}
	if err := report.ValidateBinding(experimentID, property, digest); err != nil {
		return MutationGateReport{}, err
	}
	return report, nil
}

func DecodeMutationGateReport(
	encoded []byte,
	source protocolexperiment.Experiment,
) (MutationGateReport, error) {
	report, err := decodeMutationGateReport(encoded)
	if err != nil {
		return MutationGateReport{}, err
	}
	if err := report.Validate(source); err != nil {
		return MutationGateReport{}, err
	}
	return report, nil
}

func decodeMutationGateReport(encoded []byte) (MutationGateReport, error) {
	var report MutationGateReport
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return MutationGateReport{}, fmt.Errorf("decode mutation gate report: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return MutationGateReport{}, errors.New("decode mutation gate report: trailing JSON data")
	}
	return report, nil
}

func (r MutationGateReport) Validate(source protocolexperiment.Experiment) error {
	sourceDigest, err := source.Digest()
	if err != nil {
		return err
	}
	return r.ValidateBinding(source.ExperimentID, source.Property.Identifier, sourceDigest)
}

func (r MutationGateReport) ValidateBinding(experimentID string, property string, sourceDigest string) error {
	if r.FormatVersion != MutationGateReportFormatVersion ||
		r.ResultClass != protocolcatalog.ResultClassTraceWitness ||
		r.TrustBadge != protocolcatalog.TrustBadgeTestedInstance || r.Seed == 0 {
		return errors.New("complete typed mutation gate identity is required")
	}
	if experimentID == "" || property == "" || !validMutationGateDigest(sourceDigest) ||
		r.SourceDigest != sourceDigest || r.Minimized.ExperimentID != experimentID {
		return errors.New("mutation gate source digest does not match the audited experiment")
	}
	if r.ExecutionBudget <= 0 || r.CandidateCount <= 0 || len(r.Examined) == 0 ||
		len(r.Examined) > r.ExecutionBudget || r.CandidateCount != len(r.Examined)+r.BudgetDrops {
		return errors.New("mutation gate candidate and execution budgets are inconsistent")
	}
	if !slices.Equal(r.CoverageBefore, normalizeCoverage(r.CoverageBefore)) ||
		!slices.Equal(r.CoverageDelta, normalizeCoverage(r.CoverageDelta)) {
		return errors.New("mutation gate coverage must be sorted and unique")
	}
	for _, point := range append(append([]CoveragePoint(nil), r.CoverageBefore...), r.CoverageDelta...) {
		if point.Kind == "" || point.Identifier == "" {
			return errors.New("mutation gate coverage points require a kind and identifier")
		}
	}
	if len(r.Examined) == 0 || !slices.IsSorted(r.Examined) ||
		len(slices.Compact(append([]string(nil), r.Examined...))) != len(r.Examined) {
		return errors.New("mutation gate examined identities must be nonempty, sorted, and unique")
	}
	if r.Discovered.Identifier == "" || r.Discovered.Layer == "" ||
		r.Discovered.Kind == "" || r.Discovered.Path == "" ||
		!containsMutationGateCandidate(r.Examined, r.Discovered.Kind, r.Discovered.Path) {
		return errors.New("mutation gate discovery is not bound to an examined mutation")
	}
	if !validMutationGateDigest(r.OriginalDigest) || !validMutationGateDigest(r.MinimizedDigest) ||
		!validMutationGateDigest(r.ReplayBundleDigest) {
		return errors.New("mutation gate experiment and replay digests are required")
	}
	minimizedDigest, err := r.Minimized.Digest()
	if err != nil {
		return fmt.Errorf("validate minimized experiment: %w", err)
	}
	if r.MinimizedDigest != minimizedDigest || r.Replay.ExperimentDigest != minimizedDigest {
		return errors.New("mutation gate minimized experiment or replay digest does not match")
	}
	if !r.Replay.Reproduced || len(r.Replay.Drift) != 0 ||
		r.Replay.Baseline.Kind != umpire3execution.ClaimViolating ||
		r.Replay.Current.Kind != umpire3execution.ClaimViolating ||
		r.Replay.Baseline.Property != property ||
		r.Replay.Current.Property != property {
		return errors.New("mutation gate requires a drift-free reproduced violation")
	}
	if err := r.Replay.Result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate mutation replay assurance: %w", err)
	}
	if r.PromotionSource == "" {
		return errors.New("mutation gate promotion source is required")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "promotion.go", r.PromotionSource, 0); err != nil {
		return fmt.Errorf("parse mutation promotion source: %w", err)
	}
	expectedDigest, err := r.computedArtifactDigest()
	if err != nil {
		return err
	}
	if r.ArtifactDigest != expectedDigest {
		return errors.New("mutation gate artifact digest does not match its contents")
	}
	return nil
}

func (r MutationGateReport) ValidateCoverageGuidance() error {
	if r.ExecutionBudget <= 0 || r.ExecutionBudget >= r.CandidateCount ||
		len(r.Examined) != r.ExecutionBudget || r.BudgetDrops != r.CandidateCount-r.ExecutionBudget {
		return errors.New("mutation gate did not constrain execution below the candidate population")
	}
	discovered, err := mutationCoverage(r.Discovered.Kind, r.Discovered.Path)
	if err != nil {
		return err
	}
	if slices.Contains(r.CoverageBefore, discovered) || !slices.Contains(r.CoverageDelta, discovered) {
		return errors.New("mutation gate discovery was not selected as novel coverage")
	}
	return nil
}

func (r MutationGateReport) CanonicalJSON(source protocolexperiment.Experiment) ([]byte, error) {
	if err := r.Validate(source); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode mutation gate report: %w", err)
	}
	return append(encoded, '\n'), nil
}

func sealMutationGateReport(
	report MutationGateReport,
	source protocolexperiment.Experiment,
) (MutationGateReport, error) {
	report.FormatVersion = MutationGateReportFormatVersion
	report.ResultClass = protocolcatalog.ResultClassTraceWitness
	report.TrustBadge = protocolcatalog.TrustBadgeTestedInstance
	var err error
	report.SourceDigest, err = source.Digest()
	if err != nil {
		return MutationGateReport{}, err
	}
	slices.Sort(report.Examined)
	report.ArtifactDigest, err = report.computedArtifactDigest()
	if err != nil {
		return MutationGateReport{}, err
	}
	if err := report.Validate(source); err != nil {
		return MutationGateReport{}, err
	}
	return report, nil
}

func (r MutationGateReport) computedArtifactDigest() (string, error) {
	canonical := r
	canonical.ArtifactDigest = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode mutation gate digest payload: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validMutationGateDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func containsMutationGateCandidate(identities []string, kind MutationKind, path string) bool {
	prefix := string(kind) + ":" + path + "@sha256:"
	for _, identity := range identities {
		if strings.HasPrefix(identity, prefix) && validMutationGateDigest(strings.TrimPrefix(identity, string(kind)+":"+path+"@")) {
			return true
		}
	}
	return false
}
