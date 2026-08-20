package gomadv3sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"unicode/utf8"
)

const ExplorationPlanSchema = "gomadv3.simulation-exploration-plan/v1"
const MaximumExplorationPlanBytes = 16 << 20
const MaximumExplorationOverrides = 1 << 16

type ExplorationDimension string

const (
	ExplorationRuntime  ExplorationDimension = "runtime"
	ExplorationScenario ExplorationDimension = "scenario"
	ExplorationNetwork  ExplorationDimension = "network"
	ExplorationStorage  ExplorationDimension = "storage"
	ExplorationFault    ExplorationDimension = "fault"
	ExplorationCrash    ExplorationDimension = "crash"
)

type ExplorationOverride struct {
	Dimension            ExplorationDimension `json:"dimension"`
	Ordinal              uint64               `json:"ordinal"`
	SiteSHA256           string               `json:"site_sha256"`
	Alternatives         uint32               `json:"alternatives"`
	AlternativeSetSHA256 string               `json:"alternative_set_sha256"`
	Selected             uint32               `json:"selected"`
	SelectedSHA256       string               `json:"selected_sha256"`
	Identity             string               `json:"identity"`
}

type ExplorationDecision struct {
	Dimension            ExplorationDimension `json:"dimension"`
	Ordinal              uint64               `json:"ordinal"`
	SiteSHA256           string               `json:"site_sha256"`
	Alternatives         []string             `json:"alternatives"`
	AlternativeSetSHA256 string               `json:"alternative_set_sha256"`
	Selected             uint32               `json:"selected"`
	Identity             string               `json:"identity"`
}

type ExplorationPlan struct {
	Schema           string                `json:"schema"`
	ExecutionSHA256  string                `json:"execution_sha256"`
	ControllerSHA256 string                `json:"controller_sha256"`
	BaseSeed         uint64                `json:"base_seed"`
	Overrides        []ExplorationOverride `json:"overrides"`
	CandidateSHA256  string                `json:"candidate_sha256"`
}

func NewExplorationOverride(dimension ExplorationDimension, ordinal uint64, site string, alternatives []string, selected uint32) (ExplorationOverride, error) {
	override := ExplorationOverride{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: site, Alternatives: uint32(len(alternatives)), Selected: selected,
	}
	if len(alternatives) > int(^uint32(0)) || selected >= uint32(len(alternatives)) {
		return ExplorationOverride{}, errors.New("exploration override selected rank is invalid")
	}
	var err error
	override.AlternativeSetSHA256, err = explorationAlternativeSetIdentity(dimension, ordinal, site, alternatives)
	if err != nil {
		return ExplorationOverride{}, err
	}
	override.SelectedSHA256 = alternatives[selected]
	override.Identity, err = explorationOverrideIdentity(override)
	if err != nil {
		return ExplorationOverride{}, err
	}
	return override, validateExplorationOverride(override)
}

func NewExplorationPlan(execution, controller string, baseSeed uint64, overrides []ExplorationOverride) (ExplorationPlan, error) {
	plan := ExplorationPlan{
		Schema: ExplorationPlanSchema, ExecutionSHA256: execution, ControllerSHA256: controller,
		BaseSeed: baseSeed, Overrides: append([]ExplorationOverride(nil), overrides...),
	}
	sortExplorationOverrides(plan.Overrides)
	identity, err := explorationCandidateIdentity(plan)
	if err != nil {
		return ExplorationPlan{}, err
	}
	plan.CandidateSHA256 = identity
	if err := validateExplorationPlan(plan); err != nil {
		return ExplorationPlan{}, err
	}
	return plan, nil
}

func EncodeExplorationPlan(plan ExplorationPlan) ([]byte, error) {
	if err := validateExplorationPlan(plan); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("encode simulation exploration plan: %w", err)
	}
	if len(encoded) > MaximumExplorationPlanBytes {
		return nil, &CapacityError{Resource: "exploration_plan_bytes", Required: uint64(len(encoded)), Maximum: MaximumExplorationPlanBytes}
	}
	return encoded, nil
}

func DecodeExplorationPlan(data []byte) (ExplorationPlan, error) {
	if len(data) == 0 || len(data) > MaximumExplorationPlanBytes {
		return ExplorationPlan{}, fmt.Errorf("simulation exploration plan must be between 1 and %d bytes", MaximumExplorationPlanBytes)
	}
	if !utf8.Valid(data) {
		return ExplorationPlan{}, errors.New("simulation exploration plan is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan ExplorationPlan
	if err := decoder.Decode(&plan); err != nil {
		return ExplorationPlan{}, fmt.Errorf("decode simulation exploration plan: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ExplorationPlan{}, errors.New("simulation exploration plan contains trailing JSON")
	}
	if err := validateExplorationPlan(plan); err != nil {
		return ExplorationPlan{}, err
	}
	canonical, err := json.Marshal(plan)
	if err != nil {
		return ExplorationPlan{}, fmt.Errorf("canonicalize simulation exploration plan: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return ExplorationPlan{}, errors.New("simulation exploration plan is not canonical JSON")
	}
	return plan, nil
}

func validateExplorationPlan(plan ExplorationPlan) error {
	if plan.Schema != ExplorationPlanSchema {
		return fmt.Errorf("simulation exploration plan schema = %q, want %q", plan.Schema, ExplorationPlanSchema)
	}
	if !validSHA256(plan.ExecutionSHA256) || !validSHA256(plan.ControllerSHA256) || !validSHA256(plan.CandidateSHA256) {
		return errors.New("simulation exploration plan identity is invalid")
	}
	if len(plan.Overrides) > MaximumExplorationOverrides {
		return &CapacityError{Resource: "exploration_overrides", Required: uint64(len(plan.Overrides)), Maximum: MaximumExplorationOverrides}
	}
	for index, override := range plan.Overrides {
		if err := validateExplorationOverride(override); err != nil {
			return fmt.Errorf("simulation exploration override %d: %w", index, err)
		}
		if index != 0 && !explorationOverrideBefore(plan.Overrides[index-1], override) {
			return errors.New("simulation exploration overrides are not strictly ordered")
		}
	}
	want, err := explorationCandidateIdentity(plan)
	if err != nil || want != plan.CandidateSHA256 {
		return errors.Join(errors.New("simulation exploration candidate identity does not match"), err)
	}
	return nil
}

func validateExplorationOverride(override ExplorationOverride) error {
	if explorationDimensionOrder(override.Dimension) < 0 || override.Alternatives < 2 || override.Selected >= override.Alternatives {
		return errors.New("exploration override shape is invalid")
	}
	for _, identity := range []string{override.SiteSHA256, override.AlternativeSetSHA256, override.SelectedSHA256, override.Identity} {
		if !validSHA256(identity) {
			return errors.New("exploration override identity is invalid")
		}
	}
	want, err := explorationOverrideIdentity(override)
	if err != nil || want != override.Identity {
		return errors.Join(errors.New("exploration override identity does not match"), err)
	}
	return nil
}

func explorationAlternativeSetIdentity(dimension ExplorationDimension, ordinal uint64, site string, alternatives []string) (string, error) {
	if explorationDimensionOrder(dimension) < 0 || !validSHA256(site) || len(alternatives) < 2 || len(alternatives) > int(^uint32(0)) {
		return "", errors.New("exploration alternative set is invalid")
	}
	seen := make(map[string]struct{}, len(alternatives))
	for _, alternative := range alternatives {
		if !validSHA256(alternative) {
			return "", errors.New("exploration alternative identity is invalid")
		}
		if _, ok := seen[alternative]; ok {
			return "", errors.New("exploration alternatives are duplicated")
		}
		seen[alternative] = struct{}{}
	}
	return explorationDomainHash("gomadv3-combined-frontier-alternative-set/v1", map[string]any{
		"alternatives": append([]string(nil), alternatives...), "dimension": dimension, "ordinal": ordinal, "site_sha256": site,
	})
}

func explorationOverrideIdentity(override ExplorationOverride) (string, error) {
	return explorationDomainHash("gomadv3-combined-frontier-forced-decision/v1", map[string]any{
		"alternative_set_sha256": override.AlternativeSetSHA256, "alternatives": override.Alternatives,
		"dimension": override.Dimension, "identity": "", "ordinal": override.Ordinal, "selected": override.Selected,
		"selected_sha256": override.SelectedSHA256, "site_sha256": override.SiteSHA256,
	})
}

func newExplorationDecision(dimension ExplorationDimension, ordinal uint64, site string, alternatives []string, selected uint32) (ExplorationDecision, error) {
	decision := ExplorationDecision{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: site,
		Alternatives: append([]string(nil), alternatives...), Selected: selected,
	}
	var err error
	decision.AlternativeSetSHA256, err = explorationAlternativeSetIdentity(dimension, ordinal, site, alternatives)
	if err != nil {
		return ExplorationDecision{}, err
	}
	decision.Identity, err = explorationDecisionIdentity(decision)
	if err != nil {
		return ExplorationDecision{}, err
	}
	return decision, validateExplorationDecision(decision)
}

func validateExplorationDecision(decision ExplorationDecision) error {
	if explorationDimensionOrder(decision.Dimension) < 0 || len(decision.Alternatives) < 2 || len(decision.Alternatives) > int(^uint32(0)) || decision.Selected >= uint32(len(decision.Alternatives)) || !validSHA256(decision.SiteSHA256) || !validSHA256(decision.AlternativeSetSHA256) || !validSHA256(decision.Identity) {
		return errors.New("exploration decision shape is invalid")
	}
	wantSet, err := explorationAlternativeSetIdentity(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives)
	if err != nil || wantSet != decision.AlternativeSetSHA256 {
		return errors.Join(errors.New("exploration decision alternative set does not match"), err)
	}
	wantIdentity, err := explorationDecisionIdentity(decision)
	if err != nil || wantIdentity != decision.Identity {
		return errors.Join(errors.New("exploration decision identity does not match"), err)
	}
	return nil
}

func explorationDecisionIdentity(decision ExplorationDecision) (string, error) {
	return explorationDomainHash("gomadv3-combined-frontier-decision/v1", map[string]any{
		"alternative_set_sha256": decision.AlternativeSetSHA256, "alternatives": append([]string(nil), decision.Alternatives...),
		"dimension": decision.Dimension, "identity": "", "ordinal": decision.Ordinal,
		"selected": decision.Selected, "site_sha256": decision.SiteSHA256,
	})
}

func explorationCandidateIdentity(plan ExplorationPlan) (string, error) {
	overrides := make([]any, len(plan.Overrides))
	for index, override := range plan.Overrides {
		overrides[index] = map[string]any{
			"alternative_set_sha256": override.AlternativeSetSHA256, "alternatives": override.Alternatives,
			"dimension": override.Dimension, "identity": override.Identity, "ordinal": override.Ordinal,
			"selected": override.Selected, "selected_sha256": override.SelectedSHA256, "site_sha256": override.SiteSHA256,
		}
	}
	return explorationDomainHash("gomadv3-combined-frontier-candidate/v1", map[string]any{
		"base_seed": plan.BaseSeed, "controller_sha256": plan.ControllerSHA256,
		"execution_sha256": plan.ExecutionSHA256, "overrides": overrides,
	})
}

func explorationDomainHash(domain string, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s identity: %w", domain, err)
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(encoded)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func scenarioExplorationIdentities(id string, occurrence uint64, alternatives []string) (string, []string, error) {
	if err := validateID("scenario exploration ID", id); err != nil {
		return "", nil, err
	}
	if occurrence == 0 || len(alternatives) < 2 {
		return "", nil, errors.New("scenario exploration choice is invalid")
	}
	site, err := explorationDomainHash("gomadv3-simulation-scenario-site/v1", map[string]any{"id": id, "occurrence": occurrence})
	if err != nil {
		return "", nil, err
	}
	identities := make([]string, len(alternatives))
	for index, alternative := range alternatives {
		if err := validateID("scenario exploration alternative", alternative); err != nil {
			return "", nil, err
		}
		identities[index], err = explorationDomainHash("gomadv3-simulation-scenario-alternative/v1", map[string]any{"site_sha256": site, "value": alternative})
		if err != nil {
			return "", nil, err
		}
	}
	return site, identities, nil
}

func sortExplorationOverrides(overrides []ExplorationOverride) {
	sort.Slice(overrides, func(left, right int) bool { return explorationOverrideBefore(overrides[left], overrides[right]) })
}

func explorationOverrideBefore(left, right ExplorationOverride) bool {
	if explorationDimensionOrder(left.Dimension) != explorationDimensionOrder(right.Dimension) {
		return explorationDimensionOrder(left.Dimension) < explorationDimensionOrder(right.Dimension)
	}
	return left.Ordinal < right.Ordinal
}

func explorationDimensionOrder(dimension ExplorationDimension) int {
	switch dimension {
	case ExplorationRuntime:
		return 0
	case ExplorationScenario:
		return 1
	case ExplorationNetwork:
		return 2
	case ExplorationStorage:
		return 3
	case ExplorationFault:
		return 4
	case ExplorationCrash:
		return 5
	default:
		return -1
	}
}

func cloneExplorationPlan(plan ExplorationPlan) ExplorationPlan {
	plan.Overrides = append([]ExplorationOverride(nil), plan.Overrides...)
	return plan
}

func cloneExplorationPlanPointer(plan *ExplorationPlan) *ExplorationPlan {
	if plan == nil {
		return nil
	}
	cloned := cloneExplorationPlan(*plan)
	return &cloned
}

func cloneExplorationDecisions(decisions []ExplorationDecision) []ExplorationDecision {
	if decisions == nil {
		return nil
	}
	cloned := make([]ExplorationDecision, len(decisions))
	for index, decision := range decisions {
		cloned[index] = decision
		cloned[index].Alternatives = append([]string(nil), decision.Alternatives...)
	}
	return cloned
}

func equalExplorationPlan(left, right ExplorationPlan) bool {
	return left.Schema == right.Schema && left.ExecutionSHA256 == right.ExecutionSHA256 && left.ControllerSHA256 == right.ControllerSHA256 && left.BaseSeed == right.BaseSeed && left.CandidateSHA256 == right.CandidateSHA256 && slices.Equal(left.Overrides, right.Overrides)
}

func equalOptionalExplorationPlan(left, right *ExplorationPlan) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return equalExplorationPlan(*left, *right)
}

func validateExplorationEvidence(plan *ExplorationPlan, decisions []ExplorationDecision, limits Limits) error {
	if plan == nil {
		if len(decisions) != 0 {
			return errors.New("simulation exploration decisions have no plan")
		}
		return nil
	}
	if err := validateExplorationPlan(*plan); err != nil {
		return err
	}
	if err := checkCapacity("exploration_decisions", uint64(len(decisions)), limits.ScenarioDecisions); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(decisions))
	for index, decision := range decisions {
		if err := validateExplorationDecision(decision); err != nil {
			return fmt.Errorf("simulation exploration decision %d: %w", index, err)
		}
		key := string(decision.Dimension) + ":" + fmt.Sprint(decision.Ordinal)
		if _, ok := seen[key]; ok {
			return errors.New("simulation exploration decisions contain a duplicate dimension ordinal")
		}
		seen[key] = struct{}{}
	}
	for _, override := range plan.Overrides {
		decision, ok := findExplorationDecision(decisions, override.Dimension, override.Ordinal)
		if !ok || override.SiteSHA256 != decision.SiteSHA256 || override.Alternatives != uint32(len(decision.Alternatives)) || override.AlternativeSetSHA256 != decision.AlternativeSetSHA256 || override.Selected != decision.Selected || override.SelectedSHA256 != decision.Alternatives[decision.Selected] {
			return errors.New("simulation exploration record does not prove a forced decision")
		}
	}
	return nil
}
