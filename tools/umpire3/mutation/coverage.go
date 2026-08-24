package mutation

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	protocolmonitor "go.temporal.io/server/tools/umpire3/protocol/monitor"
)

func CoveragePointsForExperiment(
	denominator protocolcatalog.CoverageDenominator,
	experiment protocolexperiment.Experiment,
) ([]protocolcatalog.ModelCoveragePoint, error) {
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validate experiment for model coverage: %w", err)
	}
	target, found := coverageTargetForExperiment(denominator, experiment)
	if !found {
		return nil, errors.New("experiment does not resolve to one model coverage target")
	}
	actions := make(map[string]struct{}, len(experiment.Actions))
	for _, action := range experiment.Actions {
		actions[action.Kind] = struct{}{}
	}
	faults := make(map[string]struct{}, len(experiment.Faults))
	for _, fault := range experiment.Faults {
		faults[fault.Kind] = struct{}{}
	}
	observations := make(map[string]struct{}, len(experiment.Checkpoints))
	for _, checkpoint := range experiment.Checkpoints {
		observations[checkpoint.Observation] = struct{}{}
	}
	evidence := make(map[string]struct{})
	monitors, err := protocolmonitor.DefaultMonitorCatalog()
	if err != nil {
		return nil, fmt.Errorf("load monitor catalog for model coverage: %w", err)
	}
	monitor, found := monitors.Program(protocolcatalog.PropertyID(experiment.Property.Identifier))
	if !found {
		return nil, fmt.Errorf("property %q has no monitor program", experiment.Property.Identifier)
	}
	completeMonitorCoverage := true
	for _, observation := range monitor.Coverage {
		if _, covered := observations[strings.TrimPrefix(observation, "observation.")]; !covered {
			completeMonitorCoverage = false
			break
		}
	}
	if completeMonitorCoverage {
		for _, requirement := range monitor.Evidence {
			evidence[string(requirement)] = struct{}{}
		}
	}
	modules := make(map[string]struct{}, len(experiment.Model.Modules))
	for _, module := range experiment.Model.Modules {
		modules[module] = struct{}{}
	}
	points := make([]protocolcatalog.ModelCoveragePoint, 0, len(target.Points))
	for _, point := range target.Points {
		covered := false
		suffix := strings.TrimPrefix(point.Identifier, string(target.Identifier)+"/")
		switch point.Dimension {
		case protocolcatalog.CoverageTransition:
			_, covered = actions[suffix]
		case protocolcatalog.CoverageProperty:
			covered = suffix == experiment.Property.Identifier
		case protocolcatalog.CoverageFault:
			_, covered = faults[suffix]
		case protocolcatalog.CoverageObservation:
			_, covered = observations[strings.TrimPrefix(suffix, "observation.")]
		case protocolcatalog.CoverageEvidence:
			_, covered = evidence[strings.TrimPrefix(suffix, "evidence:")]
		case protocolcatalog.CoverageRelation, protocolcatalog.CoverageRefinement:
			_, covered = modules[point.Source]
		default:
			return nil, fmt.Errorf("unknown coverage dimension %q", point.Dimension)
		}
		if covered {
			points = append(points, point)
		}
	}
	slices.SortFunc(points, func(left, right protocolcatalog.ModelCoveragePoint) int {
		if left.Dimension < right.Dimension {
			return -1
		}
		if left.Dimension > right.Dimension {
			return 1
		}
		return strings.Compare(left.Identifier, right.Identifier)
	})
	return points, nil
}

func coverageTargetForExperiment(
	denominator protocolcatalog.CoverageDenominator,
	experiment protocolexperiment.Experiment,
) (protocolcatalog.CoverageTarget, bool) {
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return protocolcatalog.CoverageTarget{}, false
	}
	var candidates []protocolcatalog.CoverageTarget
	boundTarget := protocolcatalog.TargetID("")
	if strings.HasPrefix(experiment.Provenance.ProofManifest, "composition:") {
		boundTarget = protocolcatalog.TargetID(strings.TrimPrefix(experiment.Provenance.ProofManifest, "composition:"))
	}
	for _, target := range denominator.Targets {
		if string(target.Property) != experiment.Property.Identifier {
			continue
		}
		if boundTarget != "" && target.Identifier != boundTarget {
			continue
		}
		for _, projection := range composition.Targets {
			if projection.Identifier != target.Identifier {
				continue
			}
			available := make(map[string]struct{}, len(projection.Modules))
			for _, module := range projection.Modules {
				available[string(module)] = struct{}{}
			}
			matches := true
			for _, module := range experiment.Model.Modules {
				if _, exists := available[module]; !exists {
					matches = false
					break
				}
			}
			if matches {
				candidates = append(candidates, target)
			}
		}
	}
	if len(candidates) != 1 {
		return protocolcatalog.CoverageTarget{}, false
	}
	return candidates[0], true
}
