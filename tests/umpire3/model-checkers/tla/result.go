package tla

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

const ResultFormatVersion = "umpire3/temporal-backend-result/v2"

const tlcLivenessViolationExitCode = 13

type Backend string

const (
	BackendTLC      Backend = "tlc"
	BackendApalache Backend = "apalache"
)

type Result struct {
	FormatVersion           string                               `json:"formatVersion"`
	Backend                 Backend                              `json:"backend"`
	BackendVersion          string                               `json:"backendVersion"`
	ViewVersion             string                               `json:"viewVersion"`
	Target                  protocol.TargetID                    `json:"target"`
	Property                protocol.PropertyID                  `json:"property"`
	World                   string                               `json:"world"`
	Variant                 string                               `json:"variant"`
	SemanticHash            string                               `json:"semanticHash"`
	GeneratedArtifactDigest string                               `json:"generatedArtifactDigest"`
	ResultClass             protocol.ResultClass                 `json:"resultClass"`
	TrustBadge              protocol.TrustBadge                  `json:"trustBadge"`
	Exact                   bool                                 `json:"exact"`
	Bound                   int                                  `json:"bound,omitempty"`
	ExecutionLimits         protocol.BackendExecutionLimits      `json:"executionLimits"`
	Fairness                []string                             `json:"fairness"`
	Axioms                  []string                             `json:"axioms"`
	Omissions               []string                             `json:"omissions"`
	EvidenceDigest          string                               `json:"evidenceDigest"`
	Lasso                   *protocol.TemporalLasso              `json:"lasso,omitempty"`
	Replay                  *protocol.TemporalLassoReplayReceipt `json:"replay,omitempty"`
}

var (
	tlcStateHeader = regexp.MustCompile(`^State ([0-9]+): (.+)$`)
	tlcPhase       = regexp.MustCompile(`^phase = "([^"]+)"$`)
	resultDigest   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

func DecodeResult(input io.Reader, limit int64, view protocol.TemporalView) (Result, error) {
	if limit <= 0 {
		return Result{}, errors.New("positive temporal backend result decode limit is required")
	}
	encoded, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil {
		return Result{}, fmt.Errorf("read temporal backend result: %w", err)
	}
	if int64(len(encoded)) > limit {
		return Result{}, fmt.Errorf("temporal backend result exceeds %d-byte limit", limit)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var result Result
	if err := decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode temporal backend result: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Result{}, errors.New("temporal backend result must contain exactly one JSON value")
	}
	if err := result.Validate(view); err != nil {
		return Result{}, err
	}
	return result, nil
}

func NormalizeTLC(view protocol.TemporalView, raw RawResult) (Result, error) {
	result, err := baseResult(view, BackendTLC, TLCVersion, raw.Limits)
	if err != nil {
		return Result{}, err
	}
	switch {
	case strings.Contains(raw.Output, "Model checking completed. No error has been found"):
		if raw.ExitCode != 0 {
			return Result{}, fmt.Errorf("TLC successful result exited with status %d", raw.ExitCode)
		}
		result.ResultClass = protocol.ResultClassExternalNoCounterexample
		result.TrustBadge = protocol.TrustBadgeExternalTool
		result.Omissions = []string{"tlc-fingerprint-collisions-not-ruled-out"}
	case strings.Contains(raw.Output, "Temporal properties were violated"):
		if raw.ExitCode != tlcLivenessViolationExitCode {
			return Result{}, fmt.Errorf("TLC counterexample exited with unexpected status %d", raw.ExitCode)
		}
		lasso, err := parseTLCLasso(view, raw.Output)
		if err != nil {
			return Result{}, err
		}
		result.ResultClass = protocol.ResultClassLassoWitness
		result.TrustBadge = protocol.TrustBadgeExternalTool
		result.Exact = true
		result.Lasso = &lasso
	default:
		return Result{}, errors.New("TLC output has no recognized bounded temporal result")
	}
	if err := result.seal(); err != nil {
		return Result{}, err
	}
	if err := result.Validate(view); err != nil {
		return Result{}, err
	}
	return result, nil
}

func NormalizeApalache(view protocol.TemporalView, raw RawResult) (Result, error) {
	if !strings.Contains(raw.Output, "Checker reports no error") {
		return Result{}, errors.New("Apalache output has no recognized bounded result")
	}
	if raw.ExitCode != 0 {
		return Result{}, fmt.Errorf("Apalache successful result exited with status %d", raw.ExitCode)
	}
	result, err := baseResult(view, BackendApalache, ApalacheVersion, raw.Limits)
	if err != nil {
		return Result{}, err
	}
	result.ResultClass = protocol.ResultClassBoundedSafe
	result.TrustBadge = protocol.TrustBadgeExternalTool
	result.Exact = true
	result.Bound = view.Bounds.MaxTraceLength
	result.Omissions = []string{"apalache-external-smt-result", "temporal-property-not-checked"}
	if err := result.seal(); err != nil {
		return Result{}, err
	}
	if err := result.Validate(view); err != nil {
		return Result{}, err
	}
	return result, nil
}

func baseResult(view protocol.TemporalView, backend Backend, version string,
	limits ToolLimits) (Result, error) {
	generated, err := Generate(view)
	if err != nil {
		return Result{}, err
	}
	fairness := make([]string, len(view.Fairness))
	for index, assumption := range view.Fairness {
		fairness[index] = assumption.Identifier
	}
	if limits.Timeout <= 0 || limits.MaxOutputBytes <= 0 || limits.CPUSeconds <= 0 || limits.MemoryBytes <= 0 {
		return Result{}, errors.New("temporal result requires enforced timeout, output, CPU, and memory limits")
	}
	executionLimits := protocol.BackendExecutionLimits{
		TimeoutMillis: limits.Timeout.Milliseconds(), CPUSeconds: limits.CPUSeconds,
		MemoryBytes: limits.MemoryBytes, MaxOutputBytes: limits.MaxOutputBytes,
	}
	return Result{
		FormatVersion: ResultFormatVersion,
		Backend:       backend, BackendVersion: version, ViewVersion: view.FormatVersion,
		Target: view.Target, Property: view.Property, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash, GeneratedArtifactDigest: generated.Digest(),
		ExecutionLimits: executionLimits,
		Fairness:        fairness, Axioms: []string{}, Omissions: []string{},
	}, nil
}

func (r *Result) seal() error {
	r.EvidenceDigest = ""
	encoded, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode temporal result evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	r.EvidenceDigest = fmt.Sprintf("sha256:%x", digest)
	return nil
}

func (r Result) validEvidenceDigest() bool {
	if !resultDigest.MatchString(r.EvidenceDigest) {
		return false
	}
	expected := r
	if err := expected.seal(); err != nil {
		return false
	}
	return expected.EvidenceDigest == r.EvidenceDigest
}

func (r Result) CanonicalJSON(view protocol.TemporalView) ([]byte, error) {
	if err := r.Validate(view); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

func (r Result) Validate(view protocol.TemporalView) error {
	if err := view.Validate(); err != nil {
		return err
	}
	generated, err := Generate(view)
	if err != nil {
		return err
	}
	if r.FormatVersion != ResultFormatVersion || r.ViewVersion != view.FormatVersion ||
		r.Target != view.Target || r.Property != view.Property || r.World != view.World ||
		r.Variant != view.Variant || r.SemanticHash != view.SemanticHash ||
		r.Axioms == nil || r.Omissions == nil || !r.validEvidenceDigest() {
		return errors.New("complete temporal backend result identity and provenance are required")
	}
	if r.GeneratedArtifactDigest != generated.Digest() {
		return errors.New("temporal backend result does not match the generated TLA+ artifact")
	}
	if r.ExecutionLimits.TimeoutMillis <= 0 || r.ExecutionLimits.CPUSeconds <= 0 ||
		r.ExecutionLimits.MemoryBytes <= 0 || r.ExecutionLimits.MaxOutputBytes <= 0 {
		return errors.New("temporal backend result requires enforced resource limits")
	}
	expectedFairness := make([]string, len(view.Fairness))
	for index, assumption := range view.Fairness {
		expectedFairness[index] = assumption.Identifier
	}
	if !equalStrings(r.Fairness, expectedFairness) {
		return errors.New("temporal backend result fairness differs from its generated view")
	}
	switch r.Backend {
	case BackendTLC:
		if r.BackendVersion != TLCVersion {
			return errors.New("TLC result does not use the pinned version")
		}
	case BackendApalache:
		if r.BackendVersion != ApalacheVersion {
			return errors.New("Apalache result does not use the pinned version")
		}
	default:
		return fmt.Errorf("unknown temporal backend %q", r.Backend)
	}
	switch r.ResultClass {
	case protocol.ResultClassLassoWitness:
		if r.Backend != BackendTLC || !r.Exact ||
			(r.TrustBadge != protocol.TrustBadgeExternalTool && r.TrustBadge != protocol.TrustBadgeCheckedCertificate) ||
			r.Bound != 0 || r.Lasso == nil || len(r.Omissions) != 0 {
			return errors.New("lasso witness requires exact external TLC trace evidence")
		}
		if err := r.Lasso.Validate(view); err != nil {
			return err
		}
		if r.TrustBadge == protocol.TrustBadgeCheckedCertificate {
			if r.Replay == nil {
				return errors.New("checked temporal lasso trust requires a Lean replay receipt")
			}
			if err := r.Replay.Validate(); err != nil {
				return err
			}
			if r.Replay.Target != r.Target || r.Replay.Property != r.Property || r.Replay.World != r.World ||
				r.Replay.Variant != r.Variant || r.Replay.SemanticHash != r.SemanticHash ||
				!slices.Equal(r.Replay.Lasso.States, r.Lasso.States) ||
				!slices.Equal(r.Replay.Lasso.Actions, r.Lasso.Actions) ||
				r.Replay.Lasso.LoopStart != r.Lasso.LoopStart || !slices.Equal(r.Axioms, r.Replay.Axioms) {
				return errors.New("checked temporal replay receipt does not match its backend lasso")
			}
			return nil
		}
		if r.Replay != nil {
			return errors.New("external temporal lasso cannot carry an unattached replay receipt")
		}
		if len(r.Axioms) != 0 {
			return errors.New("external temporal lasso cannot claim Lean axioms")
		}
		return nil
	case protocol.ResultClassExternalNoCounterexample:
		if r.Backend != BackendTLC || r.Exact || r.TrustBadge != protocol.TrustBadgeExternalTool ||
			r.Lasso != nil || r.Replay != nil || len(r.Axioms) != 0 ||
			!slices.Equal(r.Omissions, []string{"tlc-fingerprint-collisions-not-ruled-out"}) {
			return errors.New("TLC no-counterexample result must remain collision-qualified external evidence")
		}
	case protocol.ResultClassBoundedSafe:
		if r.Backend != BackendApalache || !r.Exact || r.TrustBadge != protocol.TrustBadgeExternalTool ||
			r.Bound != view.Bounds.MaxTraceLength || r.Lasso != nil || r.Replay != nil || len(r.Axioms) != 0 ||
			!slices.Equal(r.Omissions, []string{"apalache-external-smt-result", "temporal-property-not-checked"}) {
			return errors.New("Apalache bounded result must disclose that only the selected invariant was checked")
		}
	default:
		return fmt.Errorf("temporal backend cannot report result class %q", r.ResultClass)
	}
	return nil
}

func parseTLCLasso(view protocol.TemporalView, output string) (protocol.TemporalLasso, error) {
	actionNames := make(map[string]protocol.ActionKind, len(view.Actions))
	for _, action := range view.Actions {
		actionNames[identifier(string(action))] = action
	}
	var states []string
	var incoming []protocol.ActionKind
	currentLabel := ""
	stuttering := false
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if match := tlcStateHeader.FindStringSubmatch(line); len(match) == 3 {
			if _, err := strconv.Atoi(match[1]); err != nil {
				return protocol.TemporalLasso{}, err
			}
			currentLabel = match[2]
			if currentLabel == "Stuttering" {
				stuttering = true
			}
			continue
		}
		match := tlcPhase.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		states = append(states, match[1])
		if len(states) == 1 {
			incoming = append(incoming, "")
			continue
		}
		label := strings.TrimPrefix(currentLabel, "<")
		label, _, _ = strings.Cut(label, " ")
		action, exists := actionNames[label]
		if !exists {
			return protocol.TemporalLasso{}, fmt.Errorf("TLC trace action %q has no canonical source mapping", label)
		}
		incoming = append(incoming, action)
	}
	if len(states) == 0 || !stuttering {
		return protocol.TemporalLasso{}, errors.New("TLC temporal counterexample has no replayable stuttering lasso")
	}
	actions := make([]protocol.ActionKind, len(states))
	for index := 1; index < len(states); index++ {
		actions[index-1] = incoming[index]
	}
	lasso := protocol.TemporalLasso{States: states, Actions: actions, LoopStart: len(states) - 1}
	if err := lasso.Validate(view); err != nil {
		return protocol.TemporalLasso{}, fmt.Errorf("replay TLC lasso: %w", err)
	}
	return lasso, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
