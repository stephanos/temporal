package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

// Output is one admitted in-memory execution closure. It has not been published.
type Output struct {
	admitted    artifact.AdmittedSet
	run         artifactv2.ExperimentRun
	rawEvidence artifactv2.RawEvidence
}

func (o Output) AdmittedSet() artifact.AdmittedSet { return o.admitted }

func (o Output) ExperimentRun() artifactv2.ExperimentRun {
	return cloneExperimentRun(o.run)
}

func (o Output) RawEvidence() artifactv2.RawEvidence {
	return cloneRawEvidence(o.rawEvidence)
}

// InvariantError is one sanitized post-start engine or admission failure.
type InvariantError struct {
	phase             Phase
	code              string
	executionOccurred bool
}

func (e *InvariantError) Error() string {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *InvariantError) Phase() Phase {
	if e == nil {
		return ""
	}
	return e.phase
}

func (e *InvariantError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *InvariantError) ExecutionOccurred() bool {
	return e != nil && e.executionOccurred
}

type phaseContextFactory interface {
	phaseContext(context.Context, Phase, PhaseLimit, bool, time.Time) (context.Context, context.CancelFunc)
}

type boundedPhaseContextFactory struct{}

func (boundedPhaseContextFactory) phaseContext(
	parent context.Context,
	_ Phase,
	limit PhaseLimit,
	detached bool,
	invocationDeadline time.Time,
) (context.Context, context.CancelFunc) {
	if detached {
		parent = context.Background()
	}
	deadline := time.Now().Add(limit.duration)
	if invocationDeadline.Before(deadline) {
		deadline = invocationDeadline
	}
	return context.WithDeadline(parent, deadline)
}

type engineState struct {
	request           CheckedRunRequest
	evidence          *evidenceAccumulator
	phaseOutcomes     []artifactv2.PhaseOutcome
	controlAttempts   []artifactv2.ControlAttempt
	liveResources     map[string]Resource
	acquiredResources map[string]struct{}
	invariant         *InvariantError
	executionStarted  bool
}

// Run executes one already-checked request and returns only admitted in-memory artifacts.
func Run(
	ctx context.Context,
	request CheckedRunRequest,
	factory EnvironmentFactory,
	participant Participant,
) (Output, error) {
	return runWithPhaseContexts(ctx, request, factory, participant, boundedPhaseContextFactory{})
}

func runWithPhaseContexts(
	ctx context.Context,
	request CheckedRunRequest,
	factory EnvironmentFactory,
	participant Participant,
	contexts phaseContextFactory,
) (Output, error) {
	limits := request.PhaseLimits()
	if len(limits) != len(phaseOrder) || factory == nil || participant == nil {
		return Output{}, &InvariantError{code: "umpire.runtime.invariant.request", executionOccurred: false}
	}
	state := &engineState{
		request: request, evidence: newEvidenceAccumulator(request),
		phaseOutcomes:     notStartedPhaseOutcomes(),
		controlAttempts:   []artifactv2.ControlAttempt{},
		liveResources:     make(map[string]Resource),
		acquiredResources: make(map[string]struct{}),
	}
	invocationDeadline := time.Now().Add(totalPhaseDuration(limits))
	state.executionStarted = true

	preparationCommand, ok := request.Command(CommandPrepare)
	if !ok {
		state.recordInvariant(PhasePreparation, "umpire.runtime.invariant.preparation-command")
		return Output{}, state.invariant
	}
	preparationContext, preparationCancel := contexts.phaseContext(
		ctx, PhasePreparation, limits[0], false, invocationDeadline,
	)
	preparationStart := time.Now()
	environment, factoryReceipt := factory.Prepare(preparationContext, request, preparationCommand)
	preparationStatuses := []string{}
	factoryStatus := "failed"
	if err := state.consumeReceipt(PhasePreparation, preparationCommand, factoryReceipt, ""); err != nil {
		state.recordInvariant(PhasePreparation, "umpire.runtime.invariant.factory-receipt")
	} else {
		factoryStatus = terminalPhaseStatus(preparationContext, factoryReceipt.status)
		preparationStatuses = append(preparationStatuses, factoryStatus)
	}
	participantPrepared := false
	if state.invariant == nil && factoryStatus == "succeeded" {
		if environment == nil {
			state.recordInvariant(PhasePreparation, "umpire.runtime.invariant.environment")
		} else {
			participantPrepared = true
			participantReceipt := participant.Prepare(preparationContext, environment, preparationCommand)
			if err := state.consumeReceipt(PhasePreparation, preparationCommand, participantReceipt, ""); err != nil {
				state.recordInvariant(PhasePreparation, "umpire.runtime.invariant.participant-prepare-receipt")
			} else {
				preparationStatuses = append(preparationStatuses,
					terminalPhaseStatus(preparationContext, participantReceipt.status))
			}
		}
	}
	preparationStatus := combinePhaseStatuses(preparationStatuses)
	if state.invariant != nil {
		preparationStatus = "failed"
	}
	state.finishPhase(PhasePreparation, preparationStart, preparationStatus)
	preparationCancel()

	if state.invariant == nil && preparationStatus == "succeeded" {
		state.executeRealizationAndObservation(
			ctx, environment, participant, contexts, invocationDeadline,
		)
	}

	if environment != nil {
		state.executeIsolation(contexts, environment, invocationDeadline)
	}
	state.executeCleanup(
		contexts, environment, participant, participantPrepared, invocationDeadline,
	)
	if state.invariant != nil {
		return Output{}, state.invariant
	}
	return state.buildOutput()
}

func (s *engineState) executeRealizationAndObservation(
	parent context.Context,
	environment Environment,
	participant Participant,
	contexts phaseContextFactory,
	invocationDeadline time.Time,
) {
	realizationCommand, ok := s.request.Command(CommandRealize)
	if !ok {
		s.recordInvariant(PhaseRealization, "umpire.runtime.invariant.realization-command")
		return
	}
	realizationContext, cancel := contexts.phaseContext(
		parent, PhaseRealization, s.request.authority.phaseLimits[1], false, invocationDeadline,
	)
	started := time.Now()
	receipt := participant.Realize(realizationContext, environment, realizationCommand)
	status := "failed"
	if receipt.command == realizationCommand {
		status = terminalPhaseStatus(realizationContext, receipt.status)
	}
	controlStatus := controlStatus(receipt.status, status)
	if err := s.consumeReceipt(PhaseRealization, realizationCommand, receipt, controlStatus); err != nil {
		s.recordInvariant(PhaseRealization, "umpire.runtime.invariant.realization-receipt")
		status = "failed"
	}
	s.finishPhase(PhaseRealization, started, status)
	cancel()
	if s.invariant != nil {
		return
	}

	observationCommand, ok := s.request.Command(CommandObserve)
	if !ok {
		s.recordInvariant(PhaseObservation, "umpire.runtime.invariant.observation-command")
		return
	}
	observationContext, observationCancel := contexts.phaseContext(
		parent, PhaseObservation, s.request.authority.phaseLimits[2], false, invocationDeadline,
	)
	started = time.Now()
	observationReceipt := participant.Observe(observationContext, environment, observationCommand)
	observationStatus := terminalPhaseStatus(observationContext, observationReceipt.status)
	if err := s.consumeReceipt(PhaseObservation, observationCommand, observationReceipt, ""); err != nil {
		s.recordInvariant(PhaseObservation, "umpire.runtime.invariant.observation-receipt")
		observationStatus = "failed"
	}
	s.finishPhase(PhaseObservation, started, observationStatus)
	observationCancel()
}

func (s *engineState) executeIsolation(
	contexts phaseContextFactory,
	environment Environment,
	invocationDeadline time.Time,
) {
	command := s.request.IsolationCommand()
	if command.runIdentity == "" {
		s.recordInvariant(PhaseIsolation, "umpire.runtime.invariant.isolation-command")
		return
	}
	ctx, cancel := contexts.phaseContext(
		context.Background(), PhaseIsolation, s.request.authority.phaseLimits[3], true, invocationDeadline,
	)
	started := time.Now()
	receipt := environment.Isolate(ctx, command)
	status := terminalPhaseStatus(ctx, receipt.status)
	if err := s.consumeReceipt(PhaseIsolation, command, receipt, ""); err != nil {
		s.recordInvariant(PhaseIsolation, "umpire.runtime.invariant.isolation-receipt")
		status = "failed"
	}
	s.finishPhase(PhaseIsolation, started, status)
	cancel()
}

func (s *engineState) executeCleanup(
	contexts phaseContextFactory,
	environment Environment,
	participant Participant,
	participantPrepared bool,
	invocationDeadline time.Time,
) {
	command, ok := s.request.Command(CommandCleanup)
	if !ok {
		s.recordInvariant(PhaseCleanup, "umpire.runtime.invariant.cleanup-command")
		return
	}
	ctx, cancel := contexts.phaseContext(
		context.Background(), PhaseCleanup, s.request.authority.phaseLimits[4], true, invocationDeadline,
	)
	started := time.Now()
	statuses := []string{}
	if environment != nil && participantPrepared {
		receipt := participant.Cleanup(ctx, environment, command)
		status := terminalPhaseStatus(ctx, receipt.status)
		if err := s.consumeReceipt(PhaseCleanup, command, receipt, ""); err != nil {
			s.recordInvariant(PhaseCleanup, "umpire.runtime.invariant.participant-cleanup-receipt")
			status = "failed"
		}
		statuses = append(statuses, status)
	}
	if environment != nil {
		receipt := environment.Cleanup(ctx, command)
		status := terminalPhaseStatus(ctx, receipt.status)
		if err := s.consumeReceipt(PhaseCleanup, command, receipt, ""); err != nil {
			s.recordInvariant(PhaseCleanup, "umpire.runtime.invariant.environment-cleanup-receipt")
			status = "failed"
		}
		statuses = append(statuses, status)
	}
	status := combinePhaseStatuses(statuses)
	if s.invariant != nil && len(statuses) == 0 {
		status = "failed"
	}
	s.finishPhase(PhaseCleanup, started, status)
	cancel()
}

func (s *engineState) consumeReceipt(
	phase Phase,
	expected Command,
	receipt Receipt,
	control string,
) error {
	if receipt.command != expected || receipt.facts == nil ||
		receipt.acquiredResources == nil || receipt.releasedResources == nil {
		return errors.New("receipt is not bound to the expected command")
	}
	switch receipt.status {
	case ReceiptAccepted, ReceiptRejected, ReceiptUnsupported, ReceiptFailed, ReceiptCanceled:
	default:
		return errors.New("receipt status is invalid")
	}
	for _, resource := range receipt.acquiredResources {
		key := resourceKey(resource)
		if _, duplicate := s.acquiredResources[key]; duplicate {
			return fmt.Errorf("resource %q was acquired more than once", key)
		}
		s.acquiredResources[key] = struct{}{}
		s.liveResources[key] = resource
	}
	for _, resource := range receipt.releasedResources {
		key := resourceKey(resource)
		if _, live := s.liveResources[key]; !live {
			return fmt.Errorf("resource %q was released without acquisition", key)
		}
		delete(s.liveResources, key)
	}
	if control != "" {
		fact, err := controlReceiptFact(expected, control)
		if err != nil {
			return err
		}
		outcome, err := s.evidence.appendControlReceipt(phase, fact)
		if err != nil || outcome != appendRetained {
			return fmt.Errorf("control receipt could not be retained")
		}
		factID := fact.definitionID
		attempt := artifactv2.ControlAttempt{
			OccurrenceDefinitionID:  expected.occurrenceDefinitionID,
			ActionDefinitionID:      expected.actionDefinitionID,
			Attempt:                 artifactv2.NaturalFromUint64(expected.attempt),
			ReceiptFactDefinitionID: &factID,
			Status:                  control,
		}
		if control != "accepted" {
			code := "umpire.runtime.control." + control
			attempt.Code = &code
		}
		s.controlAttempts = append(s.controlAttempts, attempt)
	}
	for _, fact := range receipt.facts {
		outcome, err := s.evidence.append(phase, fact)
		if err != nil {
			return err
		}
		if outcome == appendRejected {
			return fmt.Errorf("evidence fact was rejected")
		}
	}
	return nil
}

func (s *engineState) buildOutput() (Output, error) {
	if len(s.controlAttempts) == 0 {
		program := s.request.Program()
		occurrence := program.Occurrence()
		s.controlAttempts = []artifactv2.ControlAttempt{{
			OccurrenceDefinitionID: occurrence.definitionID,
			ActionDefinitionID:     occurrence.actionDefinitionID,
			Attempt:                artifactv2.NaturalFromUint64(s.request.attempt),
			Status:                 "not-attempted",
		}}
	}
	observationStatus := s.phaseOutcomes[2].Status
	historyStatus := "partial"
	if observationStatus == "succeeded" {
		historyStatus = "closed"
	} else if observationStatus == "failed" {
		historyStatus = "failed"
	}
	controlSourceStatus := "closed"
	if s.controlAttempts[0].Status == "not-attempted" {
		controlSourceStatus = "partial"
	}
	cleanupStatus := s.phaseOutcomes[4].Status
	cleanupSourceStatus := "partial"
	if cleanupStatus == "succeeded" && len(s.liveResources) == 0 {
		cleanupSourceStatus = "closed"
	} else if cleanupStatus == "failed" {
		cleanupSourceStatus = "failed"
	}
	for _, closure := range []struct{ source, status string }{
		{EvidenceSourceCleanup, cleanupSourceStatus},
		{EvidenceSourceControlReceipt, controlSourceStatus},
		{EvidenceSourceHistory, historyStatus},
		{EvidenceSourceParticipantOutput, "closed"},
	} {
		if err := s.evidence.closeSource(closure.source, closure.status); err != nil {
			s.recordInvariant(PhaseCleanup, "umpire.runtime.invariant.source-closure")
			return Output{}, s.invariant
		}
	}
	sources, facts, gaps := s.evidence.materialize()
	experiment := s.request.Experiment()
	configuration := s.request.RuntimeConfiguration()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	cleanup := cleanupOutcome(cleanupStatus, uint64(len(s.liveResources)))
	run := artifactv2.ExperimentRun{
		FormatVersion: artifactv2.ExperimentRunFormat, RunIdentity: s.request.runIdentity,
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Attempt:              artifactv2.NaturalFromUint64(s.request.attempt),
		PhaseOutcomes:        s.phaseOutcomes, ControlAttempts: s.controlAttempts,
		SourceClosures: sourceClosures(sources), Cleanup: cleanup,
		Limits: slices.Clone(configuration.PhaseLimits), KnownGaps: cloneEngineGaps(gaps),
		Provenance: engineProvenance(),
	}
	run.OperationalStatus = operationalStatus(run)
	run.BehaviorFingerprint, err = behaviorFingerprint(struct {
		RunIdentity       string                      `json:"runIdentity"`
		PhaseOutcomes     []artifactv2.PhaseOutcome   `json:"phaseOutcomes"`
		ControlAttempts   []artifactv2.ControlAttempt `json:"controlAttempts"`
		SourceClosures    []artifactv2.SourceClosure  `json:"sourceClosures"`
		Cleanup           artifactv2.CleanupOutcome   `json:"cleanup"`
		OperationalStatus string                      `json:"operationalStatus"`
	}{run.RunIdentity, run.PhaseOutcomes, run.ControlAttempts, run.SourceClosures, run.Cleanup, run.OperationalStatus})
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	run, err = artifactv2.SealExperimentRun(run)
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	rawEvidence := artifactv2.RawEvidence{
		FormatVersion: artifactv2.RawEvidenceFormat, RunIdentity: s.request.runIdentity,
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Run:                  artifactv2.ExperimentRunArtifactBinding(run), CaptureStatus: captureStatus(sources),
		Sources: sources, Facts: facts, KnownGaps: cloneEngineGaps(gaps), Provenance: engineProvenance(),
	}
	rawEvidence.BehaviorFingerprint, err = behaviorFingerprint(struct {
		RunIdentity string                         `json:"runIdentity"`
		Sources     []artifactv2.RawEvidenceSource `json:"sources"`
		Facts       []artifactv2.RawEvidenceFact   `json:"facts"`
		KnownGaps   []artifactv2.KnownGap          `json:"knownGaps"`
	}{rawEvidence.RunIdentity, rawEvidence.Sources, rawEvidence.Facts, rawEvidence.KnownGaps})
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	rawEvidence, err = artifactv2.SealRawEvidence(rawEvidence)
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	executable, ok := s.request.input.AdmittedSet().Executable()
	if !ok {
		return Output{}, s.admissionInvariant()
	}
	admitted, err := executable.AdmitExecution(run, rawEvidence)
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	return Output{admitted: admitted, run: run, rawEvidence: rawEvidence}, nil
}

func (s *engineState) finishPhase(phase Phase, started time.Time, status string) {
	index := slices.Index(phaseOrder[:], phase)
	if index < 0 || s.phaseOutcomes[index].Status != "not-started" {
		s.recordInvariant(phase, "umpire.runtime.invariant.phase-transition")
		return
	}
	start := artifactv2.NaturalFromUint64(uint64(started.UnixMilli()))
	finished := artifactv2.NaturalFromUint64(uint64(time.Now().UnixMilli()))
	outcome := artifactv2.PhaseOutcome{
		Phase: string(phase), Status: status,
		StartedAtUnixMillis: &start, FinishedAtUnixMillis: &finished,
	}
	if status != "succeeded" {
		code := "umpire.runtime.phase." + string(phase) + "." + status
		outcome.Code = &code
	}
	s.phaseOutcomes[index] = outcome
}

func (s *engineState) recordInvariant(phase Phase, code string) {
	if s.invariant == nil {
		s.invariant = &InvariantError{phase: phase, code: code, executionOccurred: s.executionStarted}
	}
}

func (s *engineState) admissionInvariant() error {
	return &InvariantError{
		phase: PhaseCleanup, code: "umpire.runtime.invariant.artifact-admission",
		executionOccurred: s.executionStarted,
	}
}

func terminalPhaseStatus(ctx context.Context, receipt ReceiptStatus) string {
	switch receipt {
	case ReceiptRejected, ReceiptUnsupported, ReceiptFailed:
		return "failed"
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "timed-out"
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "canceled"
	}
	switch receipt {
	case ReceiptAccepted:
		return "succeeded"
	case ReceiptCanceled:
		return "canceled"
	}
	return "failed"
}

func combinePhaseStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "succeeded"
	}
	for _, status := range statuses {
		if status == "failed" {
			return "failed"
		}
	}
	for _, status := range statuses {
		if status == "timed-out" {
			return "timed-out"
		}
	}
	for _, status := range statuses {
		if status == "canceled" {
			return "canceled"
		}
	}
	return "succeeded"
}

func controlStatus(receipt ReceiptStatus, phaseStatus string) string {
	if phaseStatus == "timed-out" || phaseStatus == "canceled" {
		return "canceled"
	}
	return string(receipt)
}

func controlReceiptFact(command Command, status string) (Fact, error) {
	fields := make([]FactField, 0, 4)
	for _, field := range []struct{ definitionID, value string }{
		{artifactv2.ControlReceiptActionFieldDefinitionID, command.actionDefinitionID},
		{artifactv2.ControlReceiptAttemptFieldDefinitionID, fmt.Sprintf("%d", command.attempt)},
		{artifactv2.ControlReceiptOccurrenceFieldDefinitionID, command.occurrenceDefinitionID},
		{artifactv2.ControlReceiptStatusFieldDefinitionID, status},
	} {
		value, err := NewFactField(field.definitionID, field.value)
		if err != nil {
			return Fact{}, err
		}
		fields = append(fields, value)
	}
	return NewFact(
		controlReceiptFactID(command), EvidenceSourceControlReceipt,
		artifactv2.ControlReceiptKindDefinitionID, []string{}, fields,
	)
}

func controlReceiptFactID(command Command) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{
		"umpire.runtime.control-receipt/v1", command.runIdentity,
		command.occurrenceDefinitionID, fmt.Sprintf("%d", command.attempt),
	}, "\n")))
	return "umpire.runtime.fact.control-receipt." + hex.EncodeToString(digest[:])
}

func notStartedPhaseOutcomes() []artifactv2.PhaseOutcome {
	outcomes := make([]artifactv2.PhaseOutcome, len(phaseOrder))
	for index, phase := range phaseOrder {
		outcomes[index] = artifactv2.PhaseOutcome{Phase: string(phase), Status: "not-started"}
	}
	return outcomes
}

func totalPhaseDuration(limits []PhaseLimit) time.Duration {
	var total time.Duration
	for _, limit := range limits {
		total += limit.duration
	}
	return total
}

func resourceKey(resource Resource) string {
	return string(resource.kind) + "/" + resource.identity
}

func sourceClosures(sources []artifactv2.RawEvidenceSource) []artifactv2.SourceClosure {
	closures := make([]artifactv2.SourceClosure, len(sources))
	for index, source := range sources {
		closures[index] = artifactv2.SourceClosure{
			SourceDefinitionID: source.SourceDefinitionID, Status: source.Status,
			RecordCount: source.FactCount, ByteCount: source.ByteCount,
		}
	}
	return closures
}

func cleanupOutcome(status string, openHandles uint64) artifactv2.CleanupOutcome {
	outcome := artifactv2.CleanupOutcome{OpenHandleCount: artifactv2.NaturalFromUint64(openHandles)}
	if status == "succeeded" && openHandles == 0 {
		outcome.Status = "complete"
		return outcome
	}
	if status == "failed" {
		outcome.Status = "failed"
	} else {
		outcome.Status = "incomplete"
	}
	code := "umpire.runtime.cleanup." + outcome.Status
	outcome.Code = &code
	return outcome
}

func operationalStatus(run artifactv2.ExperimentRun) string {
	for _, phase := range run.PhaseOutcomes {
		if phase.Status == "failed" {
			return "failed"
		}
	}
	for _, control := range run.ControlAttempts {
		if control.Status == "rejected" || control.Status == "unsupported" || control.Status == "failed" {
			return "failed"
		}
	}
	for _, source := range run.SourceClosures {
		if source.Status == "failed" {
			return "failed"
		}
	}
	if run.Cleanup.Status == "failed" {
		return "failed"
	}
	for _, phase := range run.PhaseOutcomes {
		if phase.Status != "succeeded" {
			return "incomplete"
		}
	}
	for _, control := range run.ControlAttempts {
		if control.Status != "accepted" {
			return "incomplete"
		}
	}
	for _, source := range run.SourceClosures {
		if source.Status != "closed" {
			return "incomplete"
		}
	}
	if run.Cleanup.Status != "complete" || len(run.KnownGaps) != 0 {
		return "incomplete"
	}
	return "succeeded"
}

func captureStatus(sources []artifactv2.RawEvidenceSource) string {
	for _, source := range sources {
		if source.Status == "failed" {
			return "failed"
		}
	}
	for _, source := range sources {
		if source.Status == "partial" {
			return "partial"
		}
	}
	return "closed"
}

func behaviorFingerprint(value any) (string, error) {
	encoded, err := artifact.CanonicalPretty(value)
	if err != nil {
		return "", err
	}
	return artifactv2.BehaviorFingerprint(encoded), nil
}

func engineProvenance() artifactv2.Provenance {
	return artifactv2.Provenance{
		SourceDefinitionIDs: []string{"umpire.runtime.engine"},
		SourceLocations: []artifactv2.SourceLocation{{
			Path: "tools/umpire/runtime/engine.go", Line: artifactv2.NaturalFromUint64(1),
			Column: artifactv2.NaturalFromUint64(1), Provenance: "runtime-engine",
		}},
	}
}

func cloneEngineGaps(gaps []artifactv2.KnownGap) []artifactv2.KnownGap {
	cloned := slices.Clone(gaps)
	for index := range cloned {
		cloned[index].Subject = cloneEngineString(cloned[index].Subject)
		cloned[index].Detail = cloneEngineString(cloned[index].Detail)
	}
	return cloned
}

func cloneExperimentRun(run artifactv2.ExperimentRun) artifactv2.ExperimentRun {
	cloned := run
	cloned.PhaseOutcomes = slices.Clone(run.PhaseOutcomes)
	for index := range cloned.PhaseOutcomes {
		cloned.PhaseOutcomes[index].StartedAtUnixMillis = cloneEngineNatural(run.PhaseOutcomes[index].StartedAtUnixMillis)
		cloned.PhaseOutcomes[index].FinishedAtUnixMillis = cloneEngineNatural(run.PhaseOutcomes[index].FinishedAtUnixMillis)
		cloned.PhaseOutcomes[index].Code = cloneEngineString(run.PhaseOutcomes[index].Code)
	}
	cloned.ControlAttempts = slices.Clone(run.ControlAttempts)
	for index := range cloned.ControlAttempts {
		cloned.ControlAttempts[index].ReceiptFactDefinitionID = cloneEngineString(run.ControlAttempts[index].ReceiptFactDefinitionID)
		cloned.ControlAttempts[index].Code = cloneEngineString(run.ControlAttempts[index].Code)
	}
	cloned.SourceClosures = slices.Clone(run.SourceClosures)
	cloned.Cleanup.Code = cloneEngineString(run.Cleanup.Code)
	cloned.Limits = slices.Clone(run.Limits)
	cloned.KnownGaps = cloneEngineGaps(run.KnownGaps)
	cloned.Provenance.SourceDefinitionIDs = slices.Clone(run.Provenance.SourceDefinitionIDs)
	cloned.Provenance.SourceLocations = slices.Clone(run.Provenance.SourceLocations)
	return cloned
}

func cloneRawEvidence(document artifactv2.RawEvidence) artifactv2.RawEvidence {
	cloned := document
	cloned.Sources = slices.Clone(document.Sources)
	cloned.Facts = slices.Clone(document.Facts)
	for index := range cloned.Facts {
		cloned.Facts[index].CausalFactDefinitionIDs = slices.Clone(document.Facts[index].CausalFactDefinitionIDs)
		cloned.Facts[index].Fields = slices.Clone(document.Facts[index].Fields)
	}
	cloned.KnownGaps = cloneEngineGaps(document.KnownGaps)
	cloned.Provenance.SourceDefinitionIDs = slices.Clone(document.Provenance.SourceDefinitionIDs)
	cloned.Provenance.SourceLocations = slices.Clone(document.Provenance.SourceLocations)
	return cloned
}

func cloneEngineNatural(value *artifactv2.Natural) *artifactv2.Natural {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneEngineString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
