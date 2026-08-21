package protocol

import (
	"bytes"
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

type AttemptExecutionView struct {
	FirstOrder FirstOrderView
	Attempts   AttemptView
	Finite     *FiniteReplayTarget
}

func (v AttemptExecutionView) Outcomes(action ActionKind) ([]ActionOutcome, bool) {
	if v.Finite != nil {
		return v.Finite.Outcomes(action)
	}
	return v.Attempts.Outcomes(action)
}

func (v AttemptExecutionView) Replay(requests []AttemptRequest) (AttemptReplay, error) {
	if v.Finite != nil {
		return v.Finite.Replay(requests)
	}
	return v.Attempts.Replay(v.FirstOrder, requests)
}

func (v AttemptExecutionView) ReplayObserved(attempts []ObservedAttempt) (AttemptReplay, error) {
	if v.Finite != nil {
		return v.Finite.ReplayObserved(attempts)
	}
	return v.Attempts.ReplayObserved(v.FirstOrder, attempts)
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

func (v AttemptExecutionView) Target() TargetID {
	if v.Finite != nil {
		return v.Finite.Target
	}
	return v.FirstOrder.Target
}

func (v AttemptExecutionView) Property() PropertyID {
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

//go:embed generated/nexus-cancellation.first-order.json
var defaultNexusFirstOrderJSON []byte

//go:embed generated/nexus-cancellation-mutated.first-order.json
var defaultNexusMutatedFirstOrderJSON []byte

func DefaultFirstOrderView(target TargetID, variant string) (FirstOrderView, bool, error) {
	var encoded []byte
	switch {
	case target == TargetIDNexusCancellation && variant == "sound":
		encoded = defaultNexusFirstOrderJSON
	case target == TargetIDNexusCancellation && variant == "stale-completion-guard-removed":
		encoded = defaultNexusMutatedFirstOrderJSON
	default:
		return FirstOrderView{}, false, nil
	}
	view, err := DecodeFirstOrderView(bytes.NewReader(encoded), DefaultDecodeLimit)
	if err != nil {
		return FirstOrderView{}, true, err
	}
	return view, true, nil
}

func DefaultFirstOrderViewForFaults(target TargetID, faults []FaultKind) (FirstOrderView, bool, error) {
	faultSet := make(map[FaultKind]struct{}, len(faults))
	for _, fault := range faults {
		faultSet[fault] = struct{}{}
	}
	for _, variant := range []string{"stale-completion-guard-removed", "sound"} {
		view, found, err := DefaultFirstOrderView(target, variant)
		if err != nil {
			return FirstOrderView{}, found, err
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
	return FirstOrderView{}, false, nil
}

func DefaultAttemptExecutionView(experiment Experiment) (AttemptExecutionView, bool, error) {
	target := targetForExperiment(experiment)
	if target == "" {
		return AttemptExecutionView{}, false, nil
	}
	faults := make([]FaultKind, len(experiment.Faults))
	for index, fault := range experiment.Faults {
		faults[index] = FaultKind(fault.Kind)
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
	finite, found := finiteCatalog.Target(target, PropertyID(experiment.Property.Identifier))
	if !found {
		return AttemptExecutionView{}, false, nil
	}
	return AttemptExecutionView{Finite: &finite}, true, nil
}

func DefaultAttemptExecutionViewForTarget(
	target TargetID,
	property PropertyID,
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

func targetForExperiment(experiment Experiment) TargetID {
	composition, err := DefaultComposition()
	if err != nil {
		return ""
	}
	boundTarget := TargetID("")
	if strings.HasPrefix(experiment.Provenance.ProofManifest, "composition:") {
		boundTarget = TargetID(strings.TrimPrefix(experiment.Provenance.ProofManifest, "composition:"))
	}
	for _, target := range composition.Targets {
		if boundTarget != "" && target.Identifier != boundTarget {
			continue
		}
		if !slices.Contains(target.Properties, PropertyID(experiment.Property.Identifier)) {
			continue
		}
		modules := make([]string, len(target.Modules))
		for index, module := range target.Modules {
			modules[index] = string(module)
		}
		if slices.Equal(modules, experiment.Model.Modules) {
			return target.Identifier
		}
	}
	return ""
}
