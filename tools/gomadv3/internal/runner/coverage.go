package runner

import (
	"fmt"
	"sort"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func coverageHasSemantic(mode CoverageMode) bool {
	return mode == CoverageSemantic || mode == CoverageSemanticChoice
}

func coverageHasChoice(mode CoverageMode) bool {
	return mode == CoverageChoice || mode == CoverageSemanticChoice
}

func projectChoiceFeatures(trace process.ChoiceTrace, prepared target.Prepared) (choicewire.FeatureProjection, []string, error) {
	targetIdentity, err := record.ParseSHA256(prepared.SHA256)
	if err != nil {
		return choicewire.FeatureProjection{}, nil, fmt.Errorf("decode choice target identity: %w", err)
	}
	targetDigest, err := targetIdentity.Bytes()
	if err != nil {
		return choicewire.FeatureProjection{}, nil, err
	}
	projection, err := choicewire.ProjectTrace(trace.Trace, trace.Limit, targetDigest)
	if err != nil {
		return choicewire.FeatureProjection{}, nil, fmt.Errorf("project choice coverage: %w", err)
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
