package execution

import (
	"errors"
	"slices"

	evidencegraph "go.temporal.io/server/tests/umpire3/execution/evidence"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func actionEvidenceSize(evidence ActionEvidence) int64 {
	size := len(evidence.Source) + len(evidence.Reference)
	for key, value := range evidence.GroundedBindings {
		size += len(key) + len(value)
	}
	return int64(size)
}

func missingRuntimeBindings(arguments []protocolexperiment.NamedValue, bindings Bindings) []string {
	missing := make(map[string]struct{})
	for _, argument := range arguments {
		collectMissingRuntimeBindings(argument.Value, bindings, missing)
	}
	return uniqueSortedMap(missing)
}

func collectMissingRuntimeBindings(value protocolexperiment.Value, bindings Bindings, missing map[string]struct{}) {
	if value.Type == protocolexperiment.ValueSymbol && value.Text != nil {
		if _, grounded := bindings[*value.Text]; !grounded {
			missing[*value.Text] = struct{}{}
		}
		return
	}
	for _, element := range value.Elements {
		collectMissingRuntimeBindings(element, bindings, missing)
	}
	for _, field := range value.Fields {
		collectMissingRuntimeBindings(field.Value, bindings, missing)
	}
}

func uniqueSortedCapabilities(values []protocolcatalog.CapabilityID) []protocolcatalog.CapabilityID {
	seen := make(map[protocolcatalog.CapabilityID]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]protocolcatalog.CapabilityID, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func missingCapabilities(experiment protocolexperiment.Experiment, available []protocolcatalog.CapabilityID) []string {
	have := make(map[protocolcatalog.CapabilityID]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	missing := make(map[string]struct{})
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[protocolcatalog.CapabilityID(capability)]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			if _, exists := have[protocolcatalog.CapabilityID(capability)]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	return uniqueSortedMap(missing)
}

func uniqueSortedMap(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func qualifyObservation(
	checkpoint protocolexperiment.Checkpoint,
	observation Observation,
	requiredEvidence []protocolcatalog.EvidenceID,
) (bool, string) {
	if observation.CheckpointID != checkpoint.Identifier || observation.Kind != checkpoint.Observation {
		return false, "observation identity does not match checkpoint"
	}
	if observation.Source == "" || observation.SourceIdentity == "" {
		return false, "observation source identity is missing"
	}
	if observation.ClockDomain == "" || observation.SourceSequence <= 0 || observation.Reference == "" {
		return false, "observation clock, sequence, or reference is missing"
	}
	switch checkpoint.Ordering {
	case "causal":
		if observation.CausalReference == "" && len(observation.CausalReferences) == 0 {
			return false, "causal reference is missing"
		}
	case "source-sequence":
		if observation.SourceSequence <= 0 {
			return false, "source sequence is missing"
		}
	default:
		if checkpoint.Ordering != "none" {
			return false, "unknown ordering requirement"
		}
	}
	if slices.Contains(requiredEvidence, protocolcatalog.EvidenceIDIdentityLineage) &&
		(observation.EntityIdentity == "" || len(observation.Lineage) == 0) {
		return false, "entity identity or lineage is missing"
	}
	return true, ""
}

func finalizeEvidenceGraph(result *Result, maxBytes int64) {
	builder := evidencegraph.NewBuilder(evidencegraph.Limits{
		MaxFacts: max(1, len(result.Observations)), MaxBytes: max(maxBytes, int64(1)),
	})
	var graphErr error
	for _, action := range result.Actions {
		sourceIdentity := action.Evidence.SourceIdentity
		if sourceIdentity == "" {
			sourceIdentity = action.Evidence.Source
		}
		if action.Evidence.Source == "" && action.Evidence.Reference == "" {
			continue
		}
		if err := builder.AddAction(evidencegraph.Action{
			Identifier: action.Identifier, Kind: action.Kind, Outcome: string(action.Evidence.Outcome),
			SourceIdentity: sourceIdentity,
			Reference:      action.Evidence.Reference, EntityIdentity: action.Evidence.EntityIdentity,
			Lineage: action.Evidence.Lineage, PayloadDigest: action.Evidence.PayloadDigest,
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	for _, faultResult := range result.Faults {
		if !faultResult.Realized || faultResult.Reference == "" {
			continue
		}
		sourceIdentity := faultResult.SourceIdentity
		if sourceIdentity == "" {
			sourceIdentity = result.Environment.FaultAuthority
		}
		entityIdentity := faultResult.EntityIdentity
		if entityIdentity == "" {
			entityIdentity = result.Environment.IsolationIdentity
		}
		if entityIdentity == "" {
			entityIdentity = result.ExperimentDigest
		}
		if err := builder.AddAction(evidencegraph.Action{
			Identifier: faultResult.Identifier, Kind: "fault:" + faultResult.Kind,
			Outcome:        "realized",
			SourceIdentity: sourceIdentity, Reference: faultResult.Reference,
			EntityIdentity: entityIdentity, Lineage: []string{result.ExperimentDigest, entityIdentity},
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	checkpointCounts := make(map[string]int, len(result.Observations))
	for _, observation := range result.Observations {
		checkpointCounts[observation.CheckpointID]++
	}
	for _, observation := range result.Observations {
		causalReferences := append([]string(nil), observation.CausalReferences...)
		if observation.CausalReference != "" && !slices.Contains(causalReferences, observation.CausalReference) {
			causalReferences = append(causalReferences, observation.CausalReference)
		}
		identifier := observation.CheckpointID
		if checkpointCounts[observation.CheckpointID] > 1 {
			identifier += "@" + observation.SourceIdentity
		}
		if err := builder.AddFact(evidencegraph.Fact{
			Identifier: identifier, Kind: observation.Kind, Value: observation.Satisfied,
			SourceIdentity: observation.SourceIdentity, ClockDomain: observation.ClockDomain,
			SourceSequence:            observation.SourceSequence,
			AuthoritativeTimeUnixNano: observation.AuthoritativeTimeUnixNano,
			ObservedAtUnixNano:        observation.ObservedAtUnixNano, Reference: observation.Reference,
			CausalReferences: causalReferences, EntityIdentity: observation.EntityIdentity,
			Lineage: observation.Lineage, PayloadDigest: observation.PayloadDigest,
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	for _, omission := range result.Omissions {
		builder.AddOmission(omission)
	}
	if err := builder.AddClaim(evidencegraph.Claim{
		Property: result.Claim.Property, Verdict: string(result.Claim.Kind), Reason: result.Claim.Reason,
	}); err != nil && graphErr == nil {
		graphErr = err
	}
	graph, err := builder.Build()
	result.Evidence = graph
	_ = result.BindEvidenceDigest()
	if graphErr == nil {
		graphErr = err
	}
	if graphErr != nil {
		var contradiction *evidencegraph.ContradictionError
		if errors.As(graphErr, &contradiction) {
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = "normalize evidence graph: " + graphErr.Error()
		} else if result.Claim.Kind == ClaimConforming || result.Claim.Kind == ClaimViolating {
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "normalize evidence graph: " + graphErr.Error()
		}
	}
}
