package runtime

import (
	"slices"
	"strconv"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

// CheckedRunRequest is an immutable complete request that may enter the execution engine.
type CheckedRunRequest struct {
	input        artifact.ExecutableSet
	authority    Authority
	runIdentity  string
	seed         uint64
	attempt      uint64
	correlations []Correlation
	commands     []Command
}

// CheckRequest validates an already-admitted typed input and performs no IO.
func CheckRequest(
	admitted artifact.AdmittedSet,
	authority Authority,
	runIdentity string,
	seed uint64,
	attempt uint64,
) (CheckedRunRequest, error) {
	input, ok := admitted.Executable()
	if !ok {
		return CheckedRunRequest{}, preflightError(PreflightInputSet, "executable-set")
	}
	experiment := input.Experiment()
	configuration := input.RuntimeConfiguration()
	if err := checkProfile(configuration, authority); err != nil {
		return CheckedRunRequest{}, err
	}
	if len(configuration.KnownGaps) != 0 {
		return CheckedRunRequest{}, preflightError(PreflightConfiguration, "configuration-gaps")
	}
	if configuration.ConfigurationDefinitionID != authority.configurationDefinitionID ||
		configuration.BehaviorFingerprint != authority.configurationFingerprint {
		return CheckedRunRequest{}, preflightError(PreflightConfiguration, "configuration-identity")
	}
	program := authority.program
	if len(program.targetDefinitionIDs) != 1 ||
		program.targetDefinitionIDs[0] != experiment.Plan.TargetDefinitionID {
		return CheckedRunRequest{}, preflightError(PreflightTarget, "program-target")
	}
	if len(experiment.Plan.RequestedFaults) != 0 || len(experiment.Plan.RequestedActions) != 1 ||
		len(program.actionDefinitionIDs) != 1 ||
		program.actionDefinitionIDs[0] != experiment.Plan.RequestedActions[0].DefinitionID {
		return CheckedRunRequest{}, preflightError(PreflightAction, "program-action")
	}
	if err := checkOccurrence(experiment, program); err != nil {
		return CheckedRunRequest{}, err
	}
	if len(configuration.ParticipantBindings) != 1 {
		return CheckedRunRequest{}, preflightError(PreflightParticipant, "participant-cardinality")
	}
	binding := configuration.ParticipantBindings[0]
	if binding.ParticipantDefinitionID != authority.participantDefinitionID ||
		binding.ProgramDefinitionID != program.definitionID ||
		binding.ProgramBehaviorFingerprint != program.behaviorFingerprint {
		return CheckedRunRequest{}, preflightError(PreflightParticipant, "participant-binding")
	}
	protocolVersion, ok := naturalUint64(binding.ProtocolVersion)
	if !ok || binding.ProtocolDefinitionID != authority.protocolDefinitionID ||
		protocolVersion != authority.protocolVersion {
		return CheckedRunRequest{}, preflightError(PreflightProtocol, "participant-protocol")
	}
	if !slices.Equal(
		configuration.AuthorityProfile.RequiredCapabilityDefinitionIDs,
		authority.requiredCapabilities,
	) || !slices.Equal(binding.CapabilityDefinitionIDs, program.capabilityDefinitionIDs) {
		return CheckedRunRequest{}, preflightError(PreflightCapability, "configuration-capabilities")
	}
	if err := checkPhaseLimits(configuration.PhaseLimits, authority.phaseLimits); err != nil {
		return CheckedRunRequest{}, err
	}
	if !validIdentity(runIdentity) {
		return CheckedRunRequest{}, preflightError(PreflightRunIdentity, "run-identity")
	}
	correlations, err := deriveCorrelations(runIdentity)
	if err != nil {
		return CheckedRunRequest{}, err
	}
	if seed != 0 || seed != authority.seed {
		return CheckedRunRequest{}, preflightError(PreflightSeed, "seed")
	}
	if attempt != 1 || attempt != authority.attempt {
		return CheckedRunRequest{}, preflightError(PreflightAttempt, "attempt")
	}
	request := CheckedRunRequest{
		input: input, authority: authority.clone(), runIdentity: runIdentity,
		seed: seed, attempt: attempt, correlations: correlations,
	}
	request.commands = buildCommands(request)
	return request, nil
}

func checkProfile(configuration artifactv2.RuntimeConfiguration, authority Authority) error {
	version, ok := naturalUint64(configuration.AuthorityProfile.Version)
	if !ok || authority.definitionID == "" ||
		configuration.AuthorityProfile.DefinitionID != authority.definitionID ||
		version != authority.version ||
		configuration.AuthorityProfile.BehaviorFingerprint != authority.behaviorFingerprint {
		return preflightError(PreflightProfile, "authority-profile")
	}
	return nil
}

func checkOccurrence(experiment artifactv2.Experiment, program Program) error {
	if len(experiment.Plan.LinearExtension) != 1 || len(program.occurrences) != 1 {
		return preflightError(PreflightOccurrence, "program-occurrence")
	}
	actual := experiment.Plan.LinearExtension[0]
	expected := program.occurrences[0]
	position, ok := naturalUint64(actual.Position)
	if !ok || actual.DefinitionID != expected.definitionID ||
		actual.ActionDefinitionID != expected.actionDefinitionID ||
		position != expected.position || actual.AuthoredDefinitionID == nil ||
		*actual.AuthoredDefinitionID != expected.definitionID {
		return preflightError(PreflightOccurrence, "program-occurrence")
	}
	return nil
}

func checkPhaseLimits(actual []artifactv2.PhaseLimit, expected []PhaseLimit) error {
	if len(actual) != len(expected) || len(expected) != len(phaseOrder) {
		return preflightError(PreflightBudget, "phase-limits")
	}
	for index, limit := range actual {
		durationMilliseconds, durationOK := naturalUint64(limit.DurationMilliseconds)
		maxAttempts, attemptsOK := naturalUint64(limit.MaxAttempts)
		maxRecords, recordsOK := naturalUint64(limit.MaxRecords)
		maxBytes, bytesOK := naturalUint64(limit.MaxBytes)
		if !durationOK || !attemptsOK || !recordsOK || !bytesOK ||
			limit.Phase != string(expected[index].phase) ||
			durationMilliseconds != uint64(expected[index].duration/timeMillisecond) ||
			maxAttempts != expected[index].maxAttempts ||
			maxRecords != expected[index].maxRecords ||
			maxBytes != expected[index].maxBytes {
			return preflightError(PreflightBudget, "phase-limits")
		}
	}
	return nil
}

const timeMillisecond = 1_000_000

func buildCommands(request CheckedRunRequest) []Command {
	program := request.authority.program
	operationOrder := [...]CommandKind{
		CommandPrepare,
		CommandRealize,
		CommandObserve,
		commandIsolate,
		CommandCleanup,
	}
	commands := make([]Command, 0, len(operationOrder))
	for _, kind := range operationOrder {
		phaseIndex := commandPhaseIndex(kind)
		commands = append(commands, Command{
			kind:                   kind,
			phase:                  phaseOrder[phaseIndex],
			programDefinitionID:    program.definitionID,
			programFingerprint:     program.behaviorFingerprint,
			runIdentity:            request.runIdentity,
			targetDefinitionID:     program.targetDefinitionIDs[0],
			actionDefinitionID:     program.actionDefinitionIDs[0],
			occurrenceDefinitionID: program.occurrences[0].definitionID,
			attempt:                request.attempt,
			limit:                  request.authority.phaseLimits[phaseIndex],
		})
	}
	return commands
}

func commandPhaseIndex(kind CommandKind) int {
	switch kind {
	case CommandPrepare:
		return 0
	case CommandRealize:
		return 1
	case CommandObserve:
		return 2
	case commandIsolate:
		return 3
	case CommandCleanup:
		return 4
	default:
		return -1
	}
}

func naturalUint64(value artifactv2.Natural) (uint64, bool) {
	parsed, err := strconv.ParseUint(value.String(), 10, 64)
	return parsed, err == nil
}

// AdmittedSet returns the exact immutable input set.
func (r CheckedRunRequest) AdmittedSet() artifact.AdmittedSet {
	return r.input.AdmittedSet()
}

// Experiment returns an immutable copy of the admitted ExperimentSpec value.
func (r CheckedRunRequest) Experiment() artifactv2.Experiment {
	return r.input.Experiment()
}

// RuntimeConfiguration returns an immutable copy of the admitted configuration value.
func (r CheckedRunRequest) RuntimeConfiguration() artifactv2.RuntimeConfiguration {
	return r.input.RuntimeConfiguration()
}

func (r CheckedRunRequest) Authority() Authority { return r.authority.clone() }
func (r CheckedRunRequest) Program() Program     { return r.authority.program.clone() }
func (r CheckedRunRequest) RunIdentity() string  { return r.runIdentity }
func (r CheckedRunRequest) Seed() uint64         { return r.seed }
func (r CheckedRunRequest) Attempt() uint64      { return r.attempt }

func (r CheckedRunRequest) PhaseLimits() []PhaseLimit {
	return slices.Clone(r.authority.phaseLimits)
}

func (r CheckedRunRequest) Correlations() []Correlation {
	return slices.Clone(r.correlations)
}

// Command returns the one request-bound value for a supported participant command kind.
func (r CheckedRunRequest) Command(kind CommandKind) (Command, bool) {
	if kind == commandIsolate {
		return Command{}, false
	}
	for _, command := range r.commands {
		if command.kind == kind {
			return command, true
		}
	}
	return Command{}, false
}

// IsolationCommand returns the request-bound environment isolation operation.
func (r CheckedRunRequest) IsolationCommand() Command {
	for _, command := range r.commands {
		if command.kind == commandIsolate {
			return command
		}
	}
	return Command{}
}
