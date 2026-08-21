package mutationaudit

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"go.temporal.io/server/tests/umpire3/campaign"
	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const FormatVersion = "umpire3/semantic-mutation-audit/v1"

//go:embed results/semantic-mutations.audit.json
var defaultReportJSON []byte

type Stage string

const (
	StageExactExploration Stage = "exact-exploration"
	StageLeanRefinement   Stage = "lean-refinement"
	StageLeanTemporal     Stage = "lean-temporal"
	StageLiveEvidence     Stage = "live-evidence"
	StageMinimization     Stage = "minimization"
	StageNativeSearch     Stage = "native-search"
	StagePromotion        Stage = "promotion"
	StageReplay           Stage = "replay"
	StageVeil             Stage = "veil"
)

const (
	mutationNexusStaleCompletion = "nexus-stale-completion-guard-removed"
	mutationTaskDeliveryFairness = "task-delivery-fairness-removed"
	mutationAdapterCorruption    = "adapter-response-corruption-v1"
)

type Evidence struct {
	Stage         Stage                `json:"stage"`
	Mutation      string               `json:"mutation"`
	ResultClass   protocol.ResultClass `json:"resultClass"`
	TrustBadge    protocol.TrustBadge  `json:"trustBadge"`
	Digest        string               `json:"digest"`
	Declaration   string               `json:"declaration,omitempty"`
	BindingDigest string               `json:"bindingDigest,omitempty"`
}

type Report struct {
	FormatVersion        string                 `json:"formatVersion"`
	CampaignSourceDigest string                 `json:"campaignSourceDigest"`
	Evidence             []Evidence             `json:"evidence"`
	NativeTrace          protocol.SemanticTrace `json:"nativeTrace"`
	VeilTrace            protocol.SemanticTrace `json:"veilTrace"`
	TemporalTrace        protocol.SemanticTrace `json:"temporalTrace"`
	ArtifactDigest       string                 `json:"artifactDigest"`
}

func Default() (Report, error) {
	return DecodeReport(defaultReportJSON, protocol.DefaultDecodeLimit)
}

func DecodeReport(encoded []byte, limit int64) (Report, error) {
	if limit <= 0 || int64(len(encoded)) > limit {
		return Report{}, errors.New("semantic mutation audit exceeds decode limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode semantic mutation audit: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Report{}, errors.New("semantic mutation audit must contain one JSON document")
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
	if r.FormatVersion != FormatVersion || !validDigest(r.CampaignSourceDigest) ||
		len(r.Evidence) != 9 || !validDigest(r.ArtifactDigest) {
		return errors.New("complete semantic mutation audit identity and evidence are required")
	}
	if err := r.NativeTrace.Validate(); err != nil {
		return fmt.Errorf("validate native mutation trace: %w", err)
	}
	if err := r.VeilTrace.Validate(); err != nil {
		return fmt.Errorf("validate Veil mutation trace: %w", err)
	}
	if err := r.TemporalTrace.Validate(); err != nil {
		return fmt.Errorf("validate temporal mutation trace: %w", err)
	}
	if r.NativeTrace.Producer != protocol.SemanticTraceProducerNative ||
		r.VeilTrace.Producer != protocol.SemanticTraceProducerVeil ||
		r.TemporalTrace.Producer != protocol.SemanticTraceProducerLeanTemporal ||
		!slices.Equal(r.NativeTrace.Steps, r.VeilTrace.Steps) {
		return errors.New("semantic mutation traces do not agree on their declared producers and finite witness")
	}
	expected, err := r.expectedEvidence()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(r.Evidence, expected) {
		return errors.New("semantic mutation audit evidence does not match current typed artifacts")
	}
	if r.ArtifactDigest != r.computedDigest() {
		return errors.New("semantic mutation audit digest does not match its contents")
	}
	return nil
}

func seal(report Report) (Report, error) {
	report.FormatVersion = FormatVersion
	evidence, err := report.expectedEvidence()
	if err != nil {
		return Report{}, err
	}
	report.Evidence = evidence
	report.ArtifactDigest = report.computedDigest()
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) expectedEvidence() ([]Evidence, error) {
	proofs, err := protocol.DefaultProofManifests()
	if err != nil {
		return nil, err
	}
	byIdentifier := make(map[string]protocol.ProofManifest, len(proofs))
	for _, proof := range proofs {
		byIdentifier[proof.Identifier] = proof
	}
	refinement, found := byIdentifier["nexus-cancellation-mutation-rejection-v1"]
	if !found {
		return nil, errors.New("semantic mutation audit requires the Lean refinement rejection manifest")
	}
	exact, found := byIdentifier["nexus-cancellation-exact-witness-v1"]
	if !found {
		return nil, errors.New("semantic mutation audit requires the exact mutation witness manifest")
	}
	refinementDigest, err := refinement.Digest()
	if err != nil {
		return nil, err
	}
	exactDigest, err := exact.Digest()
	if err != nil {
		return nil, err
	}

	binding, err := veil.DefaultMutatedBinding()
	if err != nil {
		return nil, err
	}
	backendResult, err := veil.DefaultMutatedResult()
	if err != nil {
		return nil, err
	}
	defaultVeilTrace, err := protocol.SemanticTraceFromBackendResult(backendResult)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(r.VeilTrace, defaultVeilTrace) ||
		backendResult.GeneratedArtifactDigest != binding.ArtifactDigest {
		return nil, errors.New("semantic mutation audit Veil trace does not match the retained declaration result")
	}
	backendJSON, err := backendResult.CanonicalJSON()
	if err != nil {
		return nil, err
	}

	campaignAudit, err := campaign.DefaultApprovedMutationAuditForBinding(
		"nexus-cancellation-stale-completion-v1",
		string(protocol.PropertyIDNexusCancellationWonExcludesSuccess),
		r.CampaignSourceDigest,
	)
	if err != nil {
		return nil, err
	}
	if err := campaignAudit.ValidateCoverageGuidance(); err != nil {
		return nil, err
	}
	if campaignAudit.Replay.Result.Trace == nil {
		return nil, errors.New("semantic mutation audit campaign has no live replay trace")
	}

	nativeDigest, err := traceDigest(r.NativeTrace)
	if err != nil {
		return nil, err
	}
	temporalDigest, err := traceDigest(r.TemporalTrace)
	if err != nil {
		return nil, err
	}
	promotionDigest := digest([]byte(campaignAudit.PromotionSource))
	evidence := []Evidence{
		{Stage: StageExactExploration, Mutation: mutationNexusStaleCompletion,
			ResultClass: exact.ResultClass, TrustBadge: exact.TrustBadge,
			Digest: exactDigest, Declaration: exact.Theorem},
		{Stage: StageLeanRefinement, Mutation: mutationNexusStaleCompletion,
			ResultClass: refinement.ResultClass, TrustBadge: refinement.TrustBadge,
			Digest: refinementDigest, Declaration: refinement.Theorem},
		{Stage: StageLeanTemporal, Mutation: mutationTaskDeliveryFairness,
			ResultClass: protocol.ResultClassLassoWitness, TrustBadge: protocol.TrustBadgeCheckedCertificate,
			Digest: temporalDigest, Declaration: r.TemporalTrace.Binding.Declaration},
		{Stage: StageLiveEvidence, Mutation: mutationAdapterCorruption,
			ResultClass: campaignAudit.ResultClass, TrustBadge: campaignAudit.TrustBadge,
			Digest:      campaignAudit.ArtifactDigest,
			Declaration: campaignAudit.Replay.Result.Trace.Binding.Declaration},
		{Stage: StageMinimization, Mutation: mutationAdapterCorruption,
			ResultClass: protocol.ResultClassMetadataValidated, TrustBadge: protocol.TrustBadgeTestedInstance,
			Digest: campaignAudit.MinimizedDigest},
		{Stage: StageNativeSearch, Mutation: mutationNexusStaleCompletion,
			ResultClass: protocol.ResultClassTraceWitness, TrustBadge: protocol.TrustBadgeCheckedCertificate,
			Digest: nativeDigest, Declaration: r.NativeTrace.Binding.Declaration},
		{Stage: StagePromotion, Mutation: mutationAdapterCorruption,
			ResultClass: protocol.ResultClassMetadataValidated, TrustBadge: protocol.TrustBadgeTestedInstance,
			Digest: promotionDigest},
		{Stage: StageReplay, Mutation: mutationAdapterCorruption,
			ResultClass: protocol.ResultClassTraceWitness, TrustBadge: protocol.TrustBadgeCheckedCertificate,
			Digest:      campaignAudit.ReplayBundleDigest,
			Declaration: campaignAudit.Replay.Result.Trace.Binding.Declaration},
		{Stage: StageVeil, Mutation: mutationNexusStaleCompletion,
			ResultClass: backendResult.ResultClass, TrustBadge: backendResult.TrustBadge,
			Digest: digest(backendJSON), Declaration: binding.Binding.SemanticBinding.Declaration,
			BindingDigest: binding.ArtifactDigest},
	}
	slices.SortFunc(evidence, func(left, right Evidence) int {
		return strings.Compare(string(left.Stage), string(right.Stage))
	})
	return evidence, nil
}

func traceDigest(trace protocol.SemanticTrace) (string, error) {
	encoded, err := trace.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digest(encoded), nil
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
