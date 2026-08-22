package corpus

import (
	"fmt"
	"sort"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/world"
)

func semanticFeatures(manifest record.ExecutionRecord, coverage deterministicio.SemanticCoverage, ioTranscript, worldTransitions []byte, choices *choice.FeatureProjection) ([]Feature, error) {
	choiceCount := 0
	if choices != nil {
		choiceCount = len(choices.Values)
	}
	features := make([]Feature, 0, len(coverage.Probes)+choiceCount+8)
	if manifest.Outcome.Domain != "success" {
		features = append(features,
			Feature{Kind: FeatureFailure, Value: string(manifest.Outcome.FailureSignature)},
			Feature{Kind: FeatureInvariant, Value: manifest.Outcome.Domain + "/" + manifest.Outcome.Reason},
		)
	}
	if terminal := manifest.World.Terminal; terminal.Kind != "" && terminal.Kind != "none" {
		detail := record.HashBytes([]byte(terminal.Detail))
		features = append(features, Feature{Kind: FeatureTerminal, Value: terminal.Kind + "/" + string(detail)})
	}
	features = append(features, Feature{Kind: FeatureOutcome, Value: manifest.Outcome.Domain + "/" + manifest.Outcome.Reason + "/" + manifest.Outcome.Termination})
	worldFeatures, err := semanticWorldFeatures(manifest.World, worldTransitions)
	if err != nil {
		return nil, err
	}
	features = append(features, worldFeatures...)
	operations, err := deterministicio.DecodeTranscript(ioTranscript)
	if err != nil {
		return nil, err
	}
	var previous string
	for _, operation := range operations {
		if operation.Name == "boundary.probe" {
			continue
		}
		outcome := fmt.Sprintf("%s/%d", operation.Name, operation.Result)
		features = append(features, Feature{Kind: FeatureIOOutcome, Value: outcome})
		if previous != "" {
			features = append(features, Feature{Kind: FeatureOperationPair, Value: previous + "->" + outcome})
		}
		previous = outcome
	}
	for _, probe := range coverage.Probes {
		features = append(features, Feature{Kind: FeatureBoundaryProbe, Value: probe})
	}
	if choices != nil {
		for _, feature := range choices.Values {
			features = append(features, Feature{Kind: FeatureChoice, Value: feature.ID()})
		}
	}
	return canonicalFeatures(features), nil
}

func semanticWorldFeatures(manifest record.World, encoded []byte) ([]Feature, error) {
	if manifest.Initial.Schema == "gomadv3.world.snapshot/none" && manifest.Transitions.Schema == "gomadv3.world.transitions/none" && manifest.Final.Schema == "gomadv3.world.snapshot/none" {
		if len(encoded) != 0 || manifest.Transitions.Count != 0 {
			return nil, fmt.Errorf("none World contains transitions")
		}
		return nil, nil
	}
	if manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || manifest.Transitions.Schema != "gomadv3.world.transitions/v1" || manifest.Final.Schema != "gomadv3.world.snapshot/v1" {
		return nil, fmt.Errorf("guided World schema combination is incompatible")
	}
	transitions, err := canonicaljson.StrictDecodeJSONLines[world.Transition](encoded)
	if err != nil {
		return nil, fmt.Errorf("decode guided World transitions: %w", err)
	}
	if uint64(len(transitions)) != uint64(manifest.Transitions.Count) {
		return nil, fmt.Errorf("guided World transition count mismatch")
	}
	state := "unchanged"
	if manifest.Initial.SemanticDigest != manifest.Final.SemanticDigest {
		state = "changed"
	}
	features := []Feature{{Kind: FeatureWorld, Value: "state/" + manifest.Final.Schema + "/" + state}}
	for _, adapter := range manifest.Adapters {
		state = "unchanged"
		if adapter.InitialDigest != adapter.FinalDigest {
			state = "changed"
		}
		features = append(features, Feature{Kind: FeatureWorld, Value: "adapter/" + adapter.Schema + "/" + state})
	}
	var previous string
	for index, transition := range transitions {
		outcome, err := worldTransitionOutcome(transition)
		if err != nil {
			return nil, fmt.Errorf("guided World transition %d: %w", index, err)
		}
		features = append(features, Feature{Kind: FeatureWorld, Value: "transition/" + outcome[len("world."):]})
		if previous != "" {
			features = append(features, Feature{Kind: FeatureOperationPair, Value: previous + "->" + outcome})
		}
		previous = outcome
	}
	return features, nil
}

func worldTransitionOutcome(transition world.Transition) (string, error) {
	switch transition.Kind {
	case "register":
		if transition.Register == nil {
			return "", fmt.Errorf("register body is missing")
		}
		request := transition.Register.Request
		return "world.register/" + request.Resource.Adapter + "/" + request.Resource.Kind + "/" + request.Kind, nil
	case "ready":
		if transition.Ready == nil {
			return "", fmt.Errorf("ready body is missing")
		}
		return "world.ready/" + transition.Ready.Readiness.Kind, nil
	case "cancel":
		if transition.Cancel == nil {
			return "", fmt.Errorf("cancel body is missing")
		}
		return "world.cancel/" + string(transition.Cancel.Cancellation.Status), nil
	case "quiesce":
		if transition.Quiesce == nil {
			return "", fmt.Errorf("quiesce body is missing")
		}
		result := transition.Quiesce.Result
		return fmt.Sprintf("world.quiesce/%s/deliveries=%d/blocked=%d", result.Kind, len(result.Deliveries), len(result.Blocked)), nil
	default:
		return "", fmt.Errorf("unknown kind %q", transition.Kind)
	}
}

func canonicalFeatures(features []Feature) []Feature {
	unique := make(map[Feature]struct{}, len(features))
	for _, feature := range features {
		unique[feature] = struct{}{}
	}
	features = make([]Feature, 0, len(unique))
	for feature := range unique {
		features = append(features, feature)
	}
	sort.Slice(features, func(i, j int) bool {
		leftRank, rightRank := featureRank(features[i].Kind), featureRank(features[j].Kind)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if features[i].Kind != features[j].Kind {
			return features[i].Kind < features[j].Kind
		}
		return features[i].Value < features[j].Value
	})
	return features
}
