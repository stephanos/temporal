package mutation

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type MutationKind string

const (
	MutationProtobufValue     MutationKind = "protobuf-value"
	MutationScenarioParameter MutationKind = "scenario-parameter"
	MutationSchedule          MutationKind = "schedule"
	MutationFaultScope        MutationKind = "fault-scope"
	MutationFaultOccurrence   MutationKind = "fault-occurrence"
	MutationWorkerResponse    MutationKind = "worker-response"
	MutationTopology          MutationKind = "topology"
)

type MutationRequest struct {
	Experiment    protocolexperiment.Experiment
	Seed          int64
	MaxCandidates int
	Values        []protocolexperiment.Value
	FaultScopes   []protocolexperiment.FaultScope
	TopologyKinds []protocolcatalog.EntityKind
}

type Mutation struct {
	Kind       MutationKind                  `json:"kind"`
	Path       string                        `json:"path"`
	Digest     string                        `json:"digest"`
	Experiment protocolexperiment.Experiment `json:"experiment"`
}

type RejectedMutation struct {
	Kind   MutationKind `json:"kind"`
	Path   string       `json:"path"`
	Reason string       `json:"reason"`
}

type MutationReport struct {
	Seed     int64              `json:"seed"`
	Complete bool               `json:"complete"`
	Selected []Mutation         `json:"selected"`
	Rejected []RejectedMutation `json:"rejected"`
	Omitted  int                `json:"omitted"`
	Omission string             `json:"omission,omitempty"`
}

func Mutate(request MutationRequest) (MutationReport, error) {
	if request.MaxCandidates <= 0 {
		return MutationReport{}, errors.New("positive mutation candidate budget is required")
	}
	if err := request.Experiment.Validate(); err != nil {
		return MutationReport{}, fmt.Errorf("validate mutation source: %w", err)
	}
	encodedSource, err := json.Marshal(request.Experiment)
	if err != nil {
		return MutationReport{}, fmt.Errorf("encode mutation source: %w", err)
	}
	clone := func() (protocolexperiment.Experiment, error) {
		var result protocolexperiment.Experiment
		if err := json.Unmarshal(encodedSource, &result); err != nil {
			return protocolexperiment.Experiment{}, fmt.Errorf("clone mutation source: %w", err)
		}
		return result, nil
	}
	report := MutationReport{Seed: request.Seed, Complete: true}
	var candidates []Mutation
	add := func(kind MutationKind, path string, experiment protocolexperiment.Experiment) {
		if err := experiment.Validate(); err != nil {
			report.Rejected = append(report.Rejected, RejectedMutation{Kind: kind, Path: path, Reason: err.Error()})
			return
		}
		digest, err := experiment.Digest()
		if err != nil {
			report.Rejected = append(report.Rejected, RejectedMutation{Kind: kind, Path: path, Reason: err.Error()})
			return
		}
		candidates = append(candidates, Mutation{Kind: kind, Path: path, Digest: digest, Experiment: experiment})
	}

	parameter, err := clone()
	if err != nil {
		return MutationReport{}, err
	}
	parameter.Scope.Seed = request.Seed
	if parameter.Scope.Seed == request.Experiment.Scope.Seed {
		parameter.Scope.Seed++
	}
	add(MutationScenarioParameter, "scope.seed", parameter)

	for actionIndex, action := range request.Experiment.Actions {
		for argumentIndex, argument := range action.Arguments {
			for _, replacement := range request.Values {
				if replacement.Type != argument.Value.Type || replacement.Type == protocolexperiment.ValueSymbol {
					continue
				}
				candidate, err := clone()
				if err != nil {
					return MutationReport{}, err
				}
				candidate.Actions[actionIndex].Arguments[argumentIndex].Value = replacement
				kind := MutationProtobufValue
				if strings.Contains(strings.ToLower(argument.Name), "response") ||
					strings.Contains(strings.ToLower(argument.Name), "result") {
					kind = MutationWorkerResponse
				}
				add(kind, fmt.Sprintf("actions[%d].arguments[%s]", actionIndex, argument.Name), candidate)
			}
		}
	}
	for index := 0; index+1 < len(request.Experiment.Actions); index++ {
		left := request.Experiment.Actions[index].Identifier
		right := request.Experiment.Actions[index+1].Identifier
		if ordered(request.Experiment.Order, left, right) || ordered(request.Experiment.Order, right, left) {
			continue
		}
		candidate, err := clone()
		if err != nil {
			return MutationReport{}, err
		}
		candidate.Actions[index], candidate.Actions[index+1] = candidate.Actions[index+1], candidate.Actions[index]
		add(MutationSchedule, fmt.Sprintf("actions[%d:%d]", index, index+2), candidate)
	}
	for faultIndex := range request.Experiment.Faults {
		for _, scope := range request.FaultScopes {
			candidate, err := clone()
			if err != nil {
				return MutationReport{}, err
			}
			candidate.Faults[faultIndex].Scope = scope
			add(MutationFaultScope, fmt.Sprintf("faults[%d].scope", faultIndex), candidate)
		}
		for _, occurrence := range []protocolexperiment.FaultOccurrence{{First: 1, Count: 1}, {First: 2, Count: 1}} {
			candidate, err := clone()
			if err != nil {
				return MutationReport{}, err
			}
			candidate.Faults[faultIndex].Occurrence = occurrence
			add(MutationFaultOccurrence, fmt.Sprintf("faults[%d].occurrence", faultIndex), candidate)
		}
	}
	for _, kind := range request.TopologyKinds {
		candidate, err := clone()
		if err != nil {
			return MutationReport{}, err
		}
		candidate.Resources = append(candidate.Resources, protocolexperiment.Resource{
			Identifier: fmt.Sprintf("mutated-%s-%d", kind, len(candidate.Resources)+1), Kind: string(kind),
		})
		add(MutationTopology, "resources", candidate)
	}

	slices.SortFunc(candidates, func(left, right Mutation) int {
		leftOrder := mutationOrder(request.Seed, left)
		rightOrder := mutationOrder(request.Seed, right)
		if leftOrder < rightOrder {
			return -1
		}
		if leftOrder > rightOrder {
			return 1
		}
		return compare(left.Digest, right.Digest)
	})
	candidates = slices.CompactFunc(candidates, func(left, right Mutation) bool { return left.Digest == right.Digest })
	if len(candidates) > request.MaxCandidates {
		report.Complete = false
		report.Omitted = len(candidates) - request.MaxCandidates
		report.Omission = "mutation candidate budget exhausted"
		candidates = candidates[:request.MaxCandidates]
	}
	report.Selected = candidates
	slices.SortFunc(report.Rejected, func(left, right RejectedMutation) int {
		if result := compare(string(left.Kind), string(right.Kind)); result != 0 {
			return result
		}
		return compare(left.Path, right.Path)
	})
	return report, nil
}

func DefaultCoverageCatalog() ([]CoveragePoint, error) {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return nil, err
	}
	protobuf, err := protocolcatalog.DefaultProtobufInventory()
	if err != nil {
		return nil, err
	}
	var points []CoveragePoint
	for _, action := range catalog.Actions {
		points = append(points,
			CoveragePoint{Kind: CoverageAction, Identifier: action.Identifier},
			CoveragePoint{Kind: CoverageTransition, Identifier: action.Identifier})
	}
	for _, property := range catalog.Properties {
		points = append(points, CoveragePoint{Kind: CoverageProperty, Identifier: property.Identifier})
	}
	for _, relation := range catalog.Relations {
		points = append(points, CoveragePoint{Kind: CoverageRelation, Identifier: relation.Identifier})
	}
	for _, item := range catalog.Evidence {
		points = append(points, CoveragePoint{Kind: CoverageEvidence, Identifier: item.Identifier})
	}
	for _, item := range catalog.Faults {
		points = append(points, CoveragePoint{Kind: CoverageFault, Identifier: item.Identifier})
	}
	for _, target := range catalog.Targets {
		points = append(points, CoveragePoint{Kind: CoverageTopology, Identifier: target.Identifier})
	}
	for _, module := range composition.Modules {
		points = append(points, CoveragePoint{Kind: CoverageRefinement, Identifier: string(module.Identifier)})
	}
	for _, class := range protobuf.FieldClasses {
		points = append(points, CoveragePoint{Kind: CoverageProtobuf, Identifier: class})
	}
	for _, schedule := range []protocolexperiment.OrderRelation{
		protocolexperiment.OrderUser, protocolexperiment.OrderSemantic, protocolexperiment.OrderSameSource, protocolexperiment.OrderRuntimeCausal,
	} {
		points = append(points, CoveragePoint{Kind: CoverageSchedule, Identifier: string(schedule)})
	}
	for _, profile := range []string{
		"local-in-process", "ci-test-cluster", "remote-deployment", "grpc-only-black-box", "production-canary",
	} {
		points = append(points, CoveragePoint{Kind: CoverageProfile, Identifier: profile})
	}
	return normalizeCoverage(points), nil
}

func mutationCoverage(kind MutationKind, path string) (CoveragePoint, error) {
	if path == "" {
		return CoveragePoint{}, errors.New("mutation coverage path is required")
	}
	var coverageKind CoverageKind
	switch kind {
	case MutationProtobufValue, MutationWorkerResponse:
		coverageKind = CoverageProtobuf
	case MutationScenarioParameter:
		coverageKind = CoverageParameter
	case MutationSchedule:
		coverageKind = CoverageSchedule
	case MutationFaultScope, MutationFaultOccurrence:
		coverageKind = CoverageFault
	case MutationTopology:
		coverageKind = CoverageTopology
	default:
		return CoveragePoint{}, fmt.Errorf("unsupported mutation coverage kind %q", kind)
	}
	return CoveragePoint{Kind: coverageKind, Identifier: path}, nil
}

func ordered(constraints []protocolexperiment.OrderConstraint, left, right string) bool {
	for _, constraint := range constraints {
		if constraint.Before == left && constraint.After == right {
			return true
		}
	}
	return false
}

func mutationOrder(seed int64, mutation Mutation) uint64 {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%s", seed, mutation.Kind, mutation.Path, mutation.Digest)))
	return binary.BigEndian.Uint64(digest[:8])
}
