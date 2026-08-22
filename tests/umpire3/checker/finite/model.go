package finite

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type AttemptRequest struct {
	Action   protocolcatalog.ActionKind
	Outcomes []protocolexperiment.ActionOutcome
}

type ObservedAttempt struct {
	Action  protocolcatalog.ActionKind
	Outcome protocolexperiment.ActionOutcome
}

type AttemptReplay struct {
	Accepted        bool
	RejectedAction  protocolcatalog.ActionKind
	RejectedOutcome protocolexperiment.ActionOutcome
	LiveOnlyActions []protocolcatalog.ActionKind
}

type AttemptExecutionView struct {
	FirstOrder protocolchecker.FirstOrderView
	Attempts   protocolchecker.AttemptView
	Finite     *protocolchecker.FiniteReplayTarget
}

func (v AttemptExecutionView) Outcomes(action protocolcatalog.ActionKind) ([]protocolexperiment.ActionOutcome, bool) {
	if v.Finite != nil {
		return finiteOutcomes(*v.Finite, action)
	}
	return attemptOutcomes(v.Attempts, action)
}

func (v AttemptExecutionView) Replay(requests []AttemptRequest) (AttemptReplay, error) {
	if v.Finite != nil {
		return replayFinite(*v.Finite, requests)
	}
	return replayAttempts(v.Attempts, v.FirstOrder, requests)
}

func (v AttemptExecutionView) ReplayObserved(attempts []ObservedAttempt) (AttemptReplay, error) {
	requests := make([]AttemptRequest, len(attempts))
	for index, attempt := range attempts {
		requests[index] = AttemptRequest{Action: attempt.Action, Outcomes: []protocolexperiment.ActionOutcome{attempt.Outcome}}
	}
	return v.Replay(requests)
}

func (v AttemptExecutionView) CanonicalModel() string {
	if v.Finite != nil {
		return v.Finite.CanonicalModel
	}
	return v.FirstOrder.CanonicalModel
}

func (v AttemptExecutionView) Variant() string {
	if v.Finite != nil {
		return v.Finite.Variant
	}
	return v.FirstOrder.Variant
}

func (v AttemptExecutionView) Target() protocolcatalog.TargetID {
	if v.Finite != nil {
		return v.Finite.Target
	}
	return v.FirstOrder.Target
}

func (v AttemptExecutionView) Property() protocolcatalog.PropertyID {
	if v.Finite != nil {
		return v.Finite.Property
	}
	return v.FirstOrder.Property
}

func (v AttemptExecutionView) World() string {
	if v.Finite != nil {
		return v.Finite.World
	}
	return v.FirstOrder.World
}

func (v AttemptExecutionView) SemanticHash() string {
	if v.Finite != nil {
		return v.Finite.SemanticHash
	}
	return v.Attempts.SemanticHash
}

//go:embed testdata/generated/nexus-cancellation.first-order.json
var defaultNexusFirstOrderJSON []byte

//go:embed testdata/generated/nexus-cancellation-mutated.first-order.json
var defaultNexusMutatedFirstOrderJSON []byte

//go:embed testdata/generated/nexus-cancellation.attempt.json
var defaultNexusAttemptJSON []byte

//go:embed testdata/generated/nexus-cancellation-mutated.attempt.json
var defaultNexusMutatedAttemptJSON []byte

//go:embed testdata/generated/finite-replay-catalog.json
var defaultFiniteReplayCatalogJSON []byte

func DefaultFirstOrderView(
	target protocolcatalog.TargetID,
	variant string,
) (protocolchecker.FirstOrderView, bool, error) {
	var encoded []byte
	switch {
	case target == protocolcatalog.TargetIDNexusCancellation && variant == "sound":
		encoded = defaultNexusFirstOrderJSON
	case target == protocolcatalog.TargetIDNexusCancellation && variant == "stale-completion-guard-removed":
		encoded = defaultNexusMutatedFirstOrderJSON
	default:
		return protocolchecker.FirstOrderView{}, false, nil
	}
	view, err := protocolchecker.DecodeFirstOrderView(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.FirstOrderView{}, true, err
	}
	return view, true, nil
}

func DefaultFirstOrderViewForFaults(
	target protocolcatalog.TargetID,
	faults []protocolcatalog.FaultKind,
) (protocolchecker.FirstOrderView, bool, error) {
	faultSet := make(map[protocolcatalog.FaultKind]struct{}, len(faults))
	for _, fault := range faults {
		faultSet[fault] = struct{}{}
	}
	for _, variant := range []string{"stale-completion-guard-removed", "sound"} {
		view, found, err := DefaultFirstOrderView(target, variant)
		if err != nil {
			return protocolchecker.FirstOrderView{}, found, err
		}
		if !found {
			continue
		}
		matches := true
		for _, fault := range view.ActivatingFaults {
			if _, active := faultSet[fault]; !active {
				matches = false
				break
			}
		}
		if matches {
			return view, true, nil
		}
	}
	return protocolchecker.FirstOrderView{}, false, nil
}

func DefaultAttemptView(
	target protocolcatalog.TargetID,
	variant string,
) (protocolchecker.AttemptView, bool, error) {
	var encoded []byte
	switch {
	case target == protocolcatalog.TargetIDNexusCancellation && variant == "sound":
		encoded = defaultNexusAttemptJSON
	case target == protocolcatalog.TargetIDNexusCancellation && variant == "stale-completion-guard-removed":
		encoded = defaultNexusMutatedAttemptJSON
	default:
		return protocolchecker.AttemptView{}, false, nil
	}
	view, err := protocolchecker.DecodeAttemptView(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.AttemptView{}, true, err
	}
	firstOrder, found, err := DefaultFirstOrderView(target, variant)
	if err != nil || !found {
		return protocolchecker.AttemptView{}, true, errors.New("attempt view has no matching first-order view")
	}
	if err := view.ValidateAgainst(firstOrder); err != nil {
		return protocolchecker.AttemptView{}, true, err
	}
	return view, true, nil
}

func DefaultFiniteReplayCatalog() (protocolchecker.FiniteReplayCatalog, error) {
	return protocolchecker.DecodeFiniteReplayCatalog(defaultFiniteReplayCatalogJSON)
}

func DefaultAttemptExecutionView(
	experiment protocolexperiment.Experiment,
) (AttemptExecutionView, bool, error) {
	target := targetForExperiment(experiment)
	if target == "" {
		return AttemptExecutionView{}, false, nil
	}
	faults := make([]protocolcatalog.FaultKind, len(experiment.Faults))
	for index, fault := range experiment.Faults {
		faults[index] = protocolcatalog.FaultKind(fault.Kind)
	}
	firstOrder, found, err := DefaultFirstOrderViewForFaults(target, faults)
	if err != nil {
		return AttemptExecutionView{}, found, err
	}
	if found {
		attempts, attemptFound, attemptErr := DefaultAttemptView(target, firstOrder.Variant)
		if attemptErr != nil {
			return AttemptExecutionView{}, true, attemptErr
		}
		if !attemptFound {
			return AttemptExecutionView{}, true, fmt.Errorf(
				"first-order view %q has no matching attempt view", firstOrder.Variant)
		}
		return AttemptExecutionView{FirstOrder: firstOrder, Attempts: attempts}, true, nil
	}
	finiteCatalog, err := DefaultFiniteReplayCatalog()
	if err != nil {
		return AttemptExecutionView{}, true, err
	}
	finite, found := finiteCatalog.Target(target, protocolcatalog.PropertyID(experiment.Property.Identifier))
	if !found {
		return AttemptExecutionView{}, false, nil
	}
	return AttemptExecutionView{Finite: &finite}, true, nil
}

func DefaultAttemptExecutionViewForTarget(
	target protocolcatalog.TargetID,
	property protocolcatalog.PropertyID,
	variant string,
) (AttemptExecutionView, bool, error) {
	firstOrder, found, err := DefaultFirstOrderView(target, variant)
	if err != nil {
		return AttemptExecutionView{}, found, err
	}
	if found {
		if firstOrder.Property != property {
			return AttemptExecutionView{}, false, nil
		}
		attempts, attemptFound, attemptErr := DefaultAttemptView(target, variant)
		if attemptErr != nil {
			return AttemptExecutionView{}, true, attemptErr
		}
		if !attemptFound {
			return AttemptExecutionView{}, true,
				fmt.Errorf("first-order view %q has no matching attempt view", variant)
		}
		return AttemptExecutionView{FirstOrder: firstOrder, Attempts: attempts}, true, nil
	}
	finiteCatalog, err := DefaultFiniteReplayCatalog()
	if err != nil {
		return AttemptExecutionView{}, true, err
	}
	finite, found := finiteCatalog.Target(target, property)
	if !found || finite.Variant != variant {
		return AttemptExecutionView{}, false, nil
	}
	return AttemptExecutionView{Finite: &finite}, true, nil
}

func targetForExperiment(experiment protocolexperiment.Experiment) protocolcatalog.TargetID {
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return ""
	}
	boundTarget := protocolcatalog.TargetID("")
	if strings.HasPrefix(experiment.Provenance.ProofManifest, "composition:") {
		boundTarget = protocolcatalog.TargetID(strings.TrimPrefix(experiment.Provenance.ProofManifest, "composition:"))
	}
	var candidates []protocolcatalog.TargetID
	for _, target := range composition.Targets {
		if boundTarget != "" && target.Identifier != boundTarget {
			continue
		}
		if !slices.Contains(target.Properties, protocolcatalog.PropertyID(experiment.Property.Identifier)) {
			continue
		}
		modules := make([]string, len(target.Modules))
		for index, module := range target.Modules {
			modules[index] = string(module)
		}
		if slices.Equal(modules, experiment.Model.Modules) {
			candidates = append(candidates, target.Identifier)
			continue
		}
		containsAll := true
		for _, module := range experiment.Model.Modules {
			if !slices.Contains(modules, module) {
				containsAll = false
				break
			}
		}
		if containsAll {
			candidates = append(candidates, target.Identifier)
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return ""
}
