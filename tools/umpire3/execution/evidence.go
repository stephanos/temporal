package execution

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	checkertrace "go.temporal.io/server/tools/umpire3/checker/trace"
	umpire3fault "go.temporal.io/server/tools/umpire3/execution/fault"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexecution "go.temporal.io/server/tools/umpire3/protocol/execution"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func finalizeSemanticTrace(
	result *Result,
	experiment protocolexperiment.Experiment,
	view finite.AttemptExecutionView,
	hasView bool,
	attempts []finite.ObservedAttempt,
) {
	if result.Claim.Kind != ClaimViolating {
		result.Trace = nil
		return
	}
	if !hasView || len(attempts) == 0 {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "violating evidence has no canonical attempt trace"
		return
	}
	trace, err := checkertrace.NewLive(experiment, view, attempts)
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "replay violating evidence: " + err.Error()
		return
	}
	result.Trace = &trace
}

func finalizeFootprint(result *Result, factory Factory, session Session) {
	provider, ok := session.(umpire3fault.FootprintProvider)
	if !ok {
		provider, ok = factory.(umpire3fault.FootprintProvider)
	}
	if !ok {
		return
	}
	report, err := provider.FootprintReport()
	if err == nil {
		err = report.RequireComplete()
	}
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "qualify learned footprint: " + err.Error()
		result.Omissions = append(result.Omissions, result.Claim.Reason)
		return
	}
	result.Footprint = &report
}

func appendCorroboratingFactObservations(
	ctx context.Context,
	session Session,
	checkpoint protocolexperiment.Checkpoint,
	bindings Bindings,
	timeout time.Duration,
	result *Result,
	primary Observation,
	requiredEvidence []protocolcatalog.EvidenceID,
	catalog observation.Catalog,
) (bool, string) {
	corroborating, ok := session.(CorroboratingFactSession)
	if !ok {
		return true, ""
	}
	corroborateCtx, cancelCorroborate := context.WithTimeout(ctx, timeout)
	factSets, err := corroborating.CorroborateFacts(corroborateCtx, checkpoint, bindings)
	cancelCorroborate()
	if err != nil {
		reason := "corroborate facts: " + err.Error()
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	if len(factSets) == 0 {
		reason := "corroborating facts are unavailable"
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	program, exists := catalog.Program(protocolcatalog.ObservationID(checkpoint.Observation))
	if !exists {
		reason := fmt.Sprintf("observation %q has no generated interpreter program", checkpoint.Observation)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	sourceIdentities := map[string]struct{}{primary.SourceIdentity: {}}
	for _, facts := range factSets {
		evaluation := program.Evaluate(facts)
		if evaluation.Value != observation.True && evaluation.Value != observation.False {
			reason := fmt.Sprintf("corroborating typed observation is %s: %v", evaluation.Value, evaluation.Support)
			result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		interpreted, interpretErr := interpretedObservation(checkpoint, evaluation, facts)
		if interpretErr != nil {
			reason := "interpret corroborating facts: " + interpretErr.Error()
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if !factsShareObservationIdentity(facts, interpreted) {
			reason := "corroborating fact set combines multiple source or entity identities"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if _, duplicate := sourceIdentities[interpreted.SourceIdentity]; duplicate {
			reason := "corroborating facts are not independently sourced"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		sourceIdentities[interpreted.SourceIdentity] = struct{}{}
		if interpreted.EntityIdentity != primary.EntityIdentity || !slices.Equal(interpreted.Lineage, primary.Lineage) {
			reason := "corroborating facts identify a different entity lineage"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if interpreted.Satisfied != primary.Satisfied {
			reason := "corroborating facts contradict the primary source"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if qualified, reason := qualifyObservation(checkpoint, interpreted, requiredEvidence); !qualified {
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = "corroborating " + reason
			return false, result.Claim.Reason
		}
		if interpreted.ObservedAtUnixNano == 0 {
			interpreted.ObservedAtUnixNano = time.Now().UnixNano()
		}
		result.Facts = appendDistinctFacts(result.Facts, facts)
		result.Observations = append(result.Observations, interpreted)
	}
	return true, ""
}

func factsShareObservationIdentity(facts []observation.Fact, interpreted Observation) bool {
	if interpreted.SourceIdentity == "" || len(facts) == 0 {
		return false
	}
	for _, fact := range facts {
		if fact.Source.Identity != interpreted.SourceIdentity ||
			fact.Source.ClockDomain != interpreted.ClockDomain ||
			fact.Source.EntityIdentity != interpreted.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, interpreted.Lineage) {
			return false
		}
	}
	return true
}

func factsShareEntityIdentity(facts []observation.Fact, interpreted Observation) bool {
	if interpreted.EntityIdentity == "" || len(facts) == 0 {
		return false
	}
	for _, fact := range facts {
		if fact.Source.EntityIdentity != interpreted.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, interpreted.Lineage) {
			return false
		}
	}
	return true
}

func contradictionCheckpoint(
	observations []Observation,
	contradictions []protocolcatalog.ObservationID,
) string {
	for _, contradiction := range contradictions {
		for _, observed := range observations {
			if observed.Kind == string(contradiction) {
				return observed.CheckpointID
			}
		}
	}
	return ""
}

func appendDistinctFacts(existing []observation.Fact, additions []observation.Fact) []observation.Fact {
	for _, addition := range additions {
		if slices.ContainsFunc(existing, func(current observation.Fact) bool {
			return reflect.DeepEqual(current, addition)
		}) {
			continue
		}
		existing = append(existing, addition)
	}
	return existing
}

func interpretedObservation(
	checkpoint protocolexperiment.Checkpoint,
	evaluation observation.Evaluation,
	facts []observation.Fact,
) (Observation, error) {
	if len(evaluation.Support) == 0 {
		return Observation{}, errors.New("typed observation returned no supporting fact")
	}
	byIdentifier := make(map[string]observation.Fact, len(facts))
	for _, fact := range facts {
		byIdentifier[fact.Identifier] = fact
	}
	supporting := make([]observation.Fact, len(evaluation.Support))
	for index, identifier := range evaluation.Support {
		fact, exists := byIdentifier[identifier]
		if !exists {
			return Observation{}, fmt.Errorf("typed observation supporting fact %q is missing", identifier)
		}
		supporting[index] = fact
	}
	identity := supporting[0].Source
	latest := supporting[0]
	var causalReferences []string
	for _, fact := range supporting {
		if fact.Source.Identity != identity.Identity || fact.Source.ClockDomain != identity.ClockDomain ||
			fact.Source.EntityIdentity != identity.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, identity.Lineage) ||
			fact.Source.PayloadDigest != identity.PayloadDigest {
			return Observation{}, errors.New("typed observation supporting facts have inconsistent identity")
		}
		if fact.Source.Sequence > latest.Source.Sequence {
			latest = fact
		}
		for _, reference := range fact.Source.CausalReferences {
			if !slices.Contains(causalReferences, reference) {
				causalReferences = append(causalReferences, reference)
			}
		}
	}
	causalReference := ""
	if len(latest.Source.CausalReferences) != 0 {
		causalReference = latest.Source.CausalReferences[0]
	} else if len(causalReferences) != 0 {
		causalReference = causalReferences[0]
	}
	return Observation{
		CheckpointID:     checkpoint.Identifier,
		Kind:             checkpoint.Observation,
		Satisfied:        evaluation.Value == observation.True,
		Source:           identity.Identity,
		SourceIdentity:   identity.Identity,
		ClockDomain:      identity.ClockDomain,
		SourceSequence:   latest.Source.Sequence,
		Reference:        latest.Source.Reference,
		CausalReference:  causalReference,
		CausalReferences: causalReferences,
		EntityIdentity:   identity.EntityIdentity,
		Lineage:          append([]string(nil), identity.Lineage...),
		PayloadDigest:    identity.PayloadDigest,
		SupportingFacts:  append([]string(nil), evaluation.Support...),
	}, nil
}

func finalizeOutcome(result *Result) {
	var terminal *protocolexecution.TerminalEvidence
	for _, action := range result.Actions {
		evidence := action.Evidence
		if evidence.TerminalState == "" && evidence.TerminalDisposition == "" {
			continue
		}
		candidate := protocolexecution.TerminalEvidence{
			State: evidence.TerminalState, Disposition: evidence.TerminalDisposition,
			Reference: evidence.Reference, EntityIdentity: evidence.EntityIdentity,
		}
		terminal = &candidate
	}
	outcome, err := ClassifyOutcome(result.Claim.Kind, terminal)
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "qualify lifecycle outcome: " + err.Error()
		result.Omissions = append(result.Omissions, result.Claim.Reason)
		outcome, _ = ClassifyOutcome(result.Claim.Kind, nil)
	}
	result.Outcome = outcome
}
