package runtimeengine

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
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

type (
	Output             = umpireruntime.Output
	InvariantError     = umpireruntime.InvariantError
	Phase              = umpireruntime.Phase
	PhaseLimit         = umpireruntime.PhaseLimit
	CheckedRunRequest  = umpireruntime.CheckedRunRequest
	EnvironmentFactory = umpireruntime.EnvironmentFactory
	Environment        = umpireruntime.Environment
	Participant        = umpireruntime.Participant
	Command            = umpireruntime.Command
	CommandKind        = umpireruntime.CommandKind
	Receipt            = umpireruntime.Receipt
	ReceiptStatus      = umpireruntime.ReceiptStatus
	Resource           = umpireruntime.Resource
	ResourceKind       = umpireruntime.ResourceKind
	Occurrence         = umpireruntime.Occurrence
	Fact               = umpireruntime.Fact
	FactField          = umpireruntime.FactField
	Correlation        = umpireruntime.Correlation
	CorrelationKind    = umpireruntime.CorrelationKind
)

const (
	PhasePreparation = umpireruntime.PhasePreparation
	PhaseRealization = umpireruntime.PhaseRealization
	PhaseObservation = umpireruntime.PhaseObservation
	PhaseIsolation   = umpireruntime.PhaseIsolation
	PhaseCleanup     = umpireruntime.PhaseCleanup

	CommandPrepare = umpireruntime.CommandPrepare
	CommandRealize = umpireruntime.CommandRealize
	CommandObserve = umpireruntime.CommandObserve
	CommandCleanup = umpireruntime.CommandCleanup

	ReceiptAccepted      = umpireruntime.ReceiptAccepted
	ReceiptRejected      = umpireruntime.ReceiptRejected
	ReceiptUnsupported   = umpireruntime.ReceiptUnsupported
	ReceiptFailed        = umpireruntime.ReceiptFailed
	ReceiptCanceled      = umpireruntime.ReceiptCanceled
	ResourceEnvironment  = umpireruntime.ResourceEnvironment
	ResourceParticipant  = umpireruntime.ResourceParticipant
	CorrelationWorkflow  = umpireruntime.CorrelationWorkflow
	CorrelationOperation = umpireruntime.CorrelationOperation

	MaximumIdentityBytes = umpireruntime.MaximumIdentityBytes
	commandIsolate       = CommandKind("isolate")
)

var phaseOrder = [...]Phase{
	PhasePreparation,
	PhaseRealization,
	PhaseObservation,
	PhaseIsolation,
	PhaseCleanup,
}

var commandOrder = [...]CommandKind{
	CommandPrepare,
	CommandRealize,
	CommandObserve,
	CommandCleanup,
}

var engineAdmission = newExecutionAdmission()

type executionAdmission struct {
	slot chan struct{}
}

func newExecutionAdmission() *executionAdmission {
	admission := &executionAdmission{slot: make(chan struct{}, 1)}
	admission.slot <- struct{}{}
	return admission
}

func (a *executionAdmission) acquire(ctx context.Context) error {
	if ctx == nil {
		return errors.New("umpire runtime execution admission requires a context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.slot:
		if err := ctx.Err(); err != nil {
			a.release()
			return err
		}
		return nil
	}
}

func (a *executionAdmission) release() {
	a.slot <- struct{}{}
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
	deadline := time.Now().Add(limit.Duration())
	if invocationDeadline.Before(deadline) {
		deadline = invocationDeadline
	}
	return context.WithDeadline(parent, deadline)
}

type engineState struct {
	request             CheckedRunRequest
	evidence            *evidenceAccumulator
	phaseOutcomes       []artifactv2.PhaseOutcome
	controlAttempts     []artifactv2.ControlAttempt
	participantReceipts map[CommandKind]bool
	liveResources       map[string]Resource
	acquiredResources   map[string]struct{}
	invariant           *InvariantError
	executionStarted    bool
}

// Run executes one already-checked request and returns only admitted in-memory artifacts.
func Run(
	ctx context.Context,
	request CheckedRunRequest,
	factory EnvironmentFactory,
	participant Participant,
) (Output, error) {
	if err := engineAdmission.acquire(ctx); err != nil {
		return Output{}, err
	}
	defer engineAdmission.release()
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
		return Output{}, umpireruntime.NewInvariantError(
			"", "umpire.runtime.invariant.request", false,
		)
	}
	state := &engineState{
		request: request, evidence: newEvidenceAccumulator(request),
		phaseOutcomes:       notStartedPhaseOutcomes(),
		controlAttempts:     []artifactv2.ControlAttempt{},
		participantReceipts: make(map[CommandKind]bool),
		liveResources:       make(map[string]Resource),
		acquiredResources:   make(map[string]struct{}),
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
		factoryStatus = terminalPhaseStatus(preparationContext, factoryReceipt.Status())
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
				state.participantReceipts[CommandPrepare] = true
				preparationStatuses = append(preparationStatuses,
					terminalPhaseStatus(preparationContext, participantReceipt.Status()))
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
		parent, PhaseRealization, s.request.PhaseLimits()[1], false, invocationDeadline,
	)
	started := time.Now()
	receipt := participant.Realize(realizationContext, environment, realizationCommand)
	status := "failed"
	if receipt.Command() == realizationCommand {
		status = terminalPhaseStatus(realizationContext, receipt.Status())
	}
	realizationControlStatus := ""
	if receipt.ControlAttempted() {
		realizationControlStatus = controlStatus(receipt.Status(), status)
	}
	if err := s.consumeReceipt(PhaseRealization, realizationCommand, receipt, realizationControlStatus); err != nil {
		s.recordInvariant(PhaseRealization, "umpire.runtime.invariant.realization-receipt")
		status = "failed"
	} else {
		s.participantReceipts[CommandRealize] = true
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
		parent, PhaseObservation, s.request.PhaseLimits()[2], false, invocationDeadline,
	)
	started = time.Now()
	observationReceipt := participant.Observe(observationContext, environment, observationCommand)
	observationStatus := terminalPhaseStatus(observationContext, observationReceipt.Status())
	if err := s.consumeReceipt(PhaseObservation, observationCommand, observationReceipt, ""); err != nil {
		s.recordInvariant(PhaseObservation, "umpire.runtime.invariant.observation-receipt")
		observationStatus = "failed"
	} else {
		s.participantReceipts[CommandObserve] = true
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
	if command.RunIdentity() == "" {
		s.recordInvariant(PhaseIsolation, "umpire.runtime.invariant.isolation-command")
		return
	}
	ctx, cancel := contexts.phaseContext(
		context.Background(), PhaseIsolation, s.request.PhaseLimits()[3], true, invocationDeadline,
	)
	started := time.Now()
	receipt := environment.Isolate(ctx, command)
	status := terminalPhaseStatus(ctx, receipt.Status())
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
		context.Background(), PhaseCleanup, s.request.PhaseLimits()[4], true, invocationDeadline,
	)
	started := time.Now()
	statuses := []string{}
	if environment != nil && participantPrepared {
		receipt := participant.Cleanup(ctx, environment, command)
		status := terminalPhaseStatus(ctx, receipt.Status())
		if err := s.consumeReceipt(PhaseCleanup, command, receipt, ""); err != nil {
			s.recordInvariant(PhaseCleanup, "umpire.runtime.invariant.participant-cleanup-receipt")
			status = "failed"
		} else {
			s.participantReceipts[CommandCleanup] = true
		}
		statuses = append(statuses, status)
	}
	if environment != nil {
		receipt := environment.Cleanup(ctx, command)
		status := terminalPhaseStatus(ctx, receipt.Status())
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
	if receipt.Command() != expected || receipt.Facts() == nil ||
		receipt.AcquiredResources() == nil || receipt.ReleasedResources() == nil {
		return errors.New("receipt is not bound to the expected command")
	}
	switch receipt.Status() {
	case ReceiptAccepted, ReceiptRejected, ReceiptUnsupported, ReceiptFailed, ReceiptCanceled:
	default:
		return errors.New("receipt status is invalid")
	}
	if receipt.ControlAttempted() != (control != "") {
		return errors.New("receipt control-attempt state is invalid")
	}
	for _, resource := range receipt.AcquiredResources() {
		key := resourceKey(resource)
		if _, duplicate := s.acquiredResources[key]; duplicate {
			return fmt.Errorf("resource %q was acquired more than once", key)
		}
		s.acquiredResources[key] = struct{}{}
		s.liveResources[key] = resource
	}
	for _, resource := range receipt.ReleasedResources() {
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
		factID := fact.DefinitionID()
		attempt := artifactv2.ControlAttempt{
			OccurrenceDefinitionID:  expected.OccurrenceDefinitionID(),
			ActionDefinitionID:      expected.ActionDefinitionID(),
			Attempt:                 artifactv2.NaturalFromUint64(expected.Attempt()),
			ReceiptFactDefinitionID: &factID,
			Status:                  control,
		}
		if control != "accepted" {
			code := "umpire.runtime.control." + control
			attempt.Code = &code
		}
		s.controlAttempts = append(s.controlAttempts, attempt)
	}
	for _, fact := range receipt.Facts() {
		outcome, err := s.evidence.append(phase, fact)
		if err != nil {
			return err
		}
		if outcome == appendRejected {
			return fmt.Errorf("evidence fact was rejected")
		}
	}
	if receipt.HistoryCapacity() {
		if err := s.evidence.markSourceCapacity(EvidenceSourceHistory); err != nil {
			return err
		}
	}
	return nil
}

func (s *engineState) buildOutput() (Output, error) {
	if len(s.controlAttempts) == 0 {
		program := s.request.Program()
		occurrence := program.Occurrence()
		s.controlAttempts = []artifactv2.ControlAttempt{{
			OccurrenceDefinitionID: occurrence.DefinitionID(),
			ActionDefinitionID:     occurrence.ActionDefinitionID(),
			Attempt:                artifactv2.NaturalFromUint64(s.request.Attempt()),
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
	participantSourceStatus := "closed"
	for _, command := range commandOrder {
		if !s.participantReceipts[command] {
			participantSourceStatus = "partial"
			break
		}
	}
	for _, closure := range []struct{ source, status string }{
		{EvidenceSourceCleanup, cleanupSourceStatus},
		{EvidenceSourceControlReceipt, controlSourceStatus},
		{EvidenceSourceHistory, historyStatus},
		{EvidenceSourceParticipantOutput, participantSourceStatus},
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
		FormatVersion: artifactv2.ExperimentRunFormat, RunIdentity: s.request.RunIdentity(),
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Attempt:              artifactv2.NaturalFromUint64(s.request.Attempt()),
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
		FormatVersion: artifactv2.RawEvidenceFormat, RunIdentity: s.request.RunIdentity(),
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
	executable, ok := s.request.AdmittedSet().Executable()
	if !ok {
		return Output{}, s.admissionInvariant()
	}
	admitted, err := executable.AdmitExecution(run, rawEvidence)
	if err != nil {
		return Output{}, s.admissionInvariant()
	}
	return umpireruntime.NewOutput(admitted, run, rawEvidence), nil
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
		s.invariant = umpireruntime.NewInvariantError(phase, code, s.executionStarted)
	}
}

func (s *engineState) admissionInvariant() error {
	return umpireruntime.NewInvariantError(
		PhaseCleanup, "umpire.runtime.invariant.artifact-admission", s.executionStarted,
	)
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
		{artifactv2.ControlReceiptActionFieldDefinitionID, command.ActionDefinitionID()},
		{artifactv2.ControlReceiptAttemptFieldDefinitionID, fmt.Sprintf("%d", command.Attempt())},
		{artifactv2.ControlReceiptOccurrenceFieldDefinitionID, command.OccurrenceDefinitionID()},
		{artifactv2.ControlReceiptStatusFieldDefinitionID, status},
	} {
		value, err := umpireruntime.NewFactField(field.definitionID, field.value)
		if err != nil {
			return Fact{}, err
		}
		fields = append(fields, value)
	}
	return umpireruntime.NewFact(
		controlReceiptFactID(command), EvidenceSourceControlReceipt,
		artifactv2.ControlReceiptKindDefinitionID, []string{}, fields,
	)
}

func controlReceiptFactID(command Command) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{
		"umpire.runtime.control-receipt/v1", command.RunIdentity(),
		command.OccurrenceDefinitionID(), fmt.Sprintf("%d", command.Attempt()),
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
		total += limit.Duration()
	}
	return total
}

func resourceKey(resource Resource) string {
	return string(resource.Kind()) + "/" + resource.Identity()
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
			Path: "tools/umpire/internal/runtimeengine/engine.go", Line: artifactv2.NaturalFromUint64(1),
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

func cloneEngineString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
