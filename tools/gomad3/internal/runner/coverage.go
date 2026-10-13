package runner

import (
	"fmt"
	"sort"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	campaignoptions "go.temporal.io/server/tools/gomad3/internal/runner/options"
)

func coverageHasSemantic(mode campaignoptions.CoverageMode) bool {
	return mode == campaignoptions.CoverageSemantic || mode == campaignoptions.CoverageSemanticChoice
}

func coverageHasChoice(mode campaignoptions.CoverageMode) bool {
	return mode == campaignoptions.CoverageChoice || mode == campaignoptions.CoverageSemanticChoice
}

func projectChoiceFeatures(trace backend.ChoiceTrace, prepared target.Prepared) (choice.FeatureProjection, []string, error) {
	targetIdentity, err := record.ParseSHA256(prepared.SHA256)
	if err != nil {
		return choice.FeatureProjection{}, nil, fmt.Errorf("decode choice target identity: %w", err)
	}
	targetDigest, err := targetIdentity.Bytes()
	if err != nil {
		return choice.FeatureProjection{}, nil, err
	}
	projection, err := choice.ProjectTrace(trace.Trace, trace.Limit, targetDigest)
	if err != nil {
		return choice.FeatureProjection{}, nil, fmt.Errorf("project choice coverage: %w", err)
	}
	identities := make([]string, 0, len(projection.Features.Values))
	for _, feature := range projection.Features.Values {
		identities = append(identities, feature.ID())
	}
	sort.Strings(identities)
	identities = compactStrings(identities)
	return projection.Features, identities, nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func novelStrings(observed []string, prior map[string]struct{}) []string {
	novel := make([]string, 0, len(observed))
	for _, value := range observed {
		if _, found := prior[value]; !found {
			novel = append(novel, value)
		}
	}
	return novel
}

func addStrings(destination map[string]struct{}, values []string) {
	for _, value := range values {
		destination[value] = struct{}{}
	}
}

// choiceTraceObserved reports whether the target ran to a point where it wrote
// its choice trace: a watchdog kill or a cancellation stops it before.
func choiceTraceObserved(result backend.ExecutionResult) bool {
	return !result.WatchdogTimeout && !result.Cancelled
}
