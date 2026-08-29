// Package runtime owns the bounded, domain-neutral execution and participant contracts.
package runtime

import (
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const (
	// MaximumIdentityBytes is the shared Artifact identity ceiling.
	MaximumIdentityBytes = artifact.MaximumIdentityBytes
	// MaximumFactValueBytes bounds one retained participant fact value.
	MaximumFactValueBytes = artifact.MaximumDiagnosticBytes
)

// Phase is one member of the fixed execution order.
type Phase string

const (
	PhasePreparation Phase = "preparation"
	PhaseRealization Phase = "realization"
	PhaseObservation Phase = "observation"
	PhaseIsolation   Phase = "isolation"
	PhaseCleanup     Phase = "cleanup"
)

var phaseOrder = [...]Phase{
	PhasePreparation,
	PhaseRealization,
	PhaseObservation,
	PhaseIsolation,
	PhaseCleanup,
}

// PhaseLimit is a constructor-checked per-phase budget.
type PhaseLimit struct {
	phase       Phase
	duration    time.Duration
	maxAttempts uint64
	maxRecords  uint64
	maxBytes    uint64
}

var canonicalPhaseLimits = [...]PhaseLimit{
	{phase: PhasePreparation, duration: 30 * time.Second, maxAttempts: 1, maxRecords: 128, maxBytes: 1 << 20},
	{phase: PhaseRealization, duration: 30 * time.Second, maxAttempts: 1, maxRecords: 128, maxBytes: 1 << 20},
	{phase: PhaseObservation, duration: 30 * time.Second, maxAttempts: 1, maxRecords: 3584, maxBytes: 12 << 20},
	{phase: PhaseIsolation, duration: 15 * time.Second, maxAttempts: 1, maxRecords: 128, maxBytes: 1 << 20},
	{phase: PhaseCleanup, duration: 15 * time.Second, maxAttempts: 1, maxRecords: 128, maxBytes: 1 << 20},
}

// NewPhaseLimit admits only the one fixed budget for its phase.
func NewPhaseLimit(
	phase Phase,
	duration time.Duration,
	maxAttempts uint64,
	maxRecords uint64,
	maxBytes uint64,
) (PhaseLimit, error) {
	candidate := PhaseLimit{
		phase: phase, duration: duration, maxAttempts: maxAttempts,
		maxRecords: maxRecords, maxBytes: maxBytes,
	}
	for _, expected := range canonicalPhaseLimits {
		if phase == expected.phase && candidate == expected {
			return candidate, nil
		}
	}
	return PhaseLimit{}, preflightError(PreflightBudget, string(phase))
}

// CanonicalPhaseLimits returns the fixed five phase budgets in execution order.
func CanonicalPhaseLimits() []PhaseLimit {
	return append([]PhaseLimit(nil), canonicalPhaseLimits[:]...)
}

func (l PhaseLimit) Phase() Phase            { return l.phase }
func (l PhaseLimit) Duration() time.Duration { return l.duration }
func (l PhaseLimit) MaxAttempts() uint64     { return l.maxAttempts }
func (l PhaseLimit) MaxRecords() uint64      { return l.maxRecords }
func (l PhaseLimit) MaxBytes() uint64        { return l.maxBytes }

// Authority is an inert checked profile and its one participant contract.
type Authority struct {
	definitionID              string
	version                   uint64
	behaviorFingerprint       string
	configurationDefinitionID string
	configurationFingerprint  string
	requiredCapabilities      []string
	phaseLimits               []PhaseLimit
	seed                      uint64
	attempt                   uint64
	participantDefinitionID   string
	protocolDefinitionID      string
	protocolVersion           uint64
	participantCount          uint64
	programCount              uint64
	program                   Program
}

// NewAuthority validates the complete closed authority contract without performing IO.
func NewAuthority(
	definitionID string,
	version uint64,
	behaviorFingerprint string,
	configurationDefinitionID string,
	configurationFingerprint string,
	requiredCapabilities []string,
	phaseLimits []PhaseLimit,
	seed uint64,
	attempt uint64,
	participantDefinitionID string,
	protocolDefinitionID string,
	protocolVersion uint64,
	participantCount uint64,
	programCount uint64,
	program Program,
) (Authority, error) {
	if !validIdentity(definitionID) || version != 2 || !artifactv2.ValidDigest(behaviorFingerprint) {
		return Authority{}, preflightError(PreflightProfile, "authority-profile")
	}
	if !validIdentity(configurationDefinitionID) || !artifactv2.ValidDigest(configurationFingerprint) {
		return Authority{}, preflightError(PreflightConfiguration, "configuration")
	}
	if err := validateIdentitySet(requiredCapabilities, PreflightCapability, "profile-capabilities"); err != nil {
		return Authority{}, err
	}
	if len(requiredCapabilities) != 3 {
		return Authority{}, preflightError(PreflightCapability, "profile-capabilities")
	}
	if !slices.Equal(phaseLimits, canonicalPhaseLimits[:]) {
		return Authority{}, preflightError(PreflightBudget, "phase-limits")
	}
	if seed != 0 {
		return Authority{}, preflightError(PreflightSeed, "authority-seed")
	}
	if attempt != 1 {
		return Authority{}, preflightError(PreflightAttempt, "authority-attempt")
	}
	if participantCount != 1 || programCount != 1 || program.definitionID == "" {
		return Authority{}, preflightError(PreflightParticipant, "participant-cardinality")
	}
	if !validIdentity(participantDefinitionID) {
		return Authority{}, preflightError(PreflightParticipant, "participant")
	}
	if !validIdentity(protocolDefinitionID) || protocolVersion != 2 {
		return Authority{}, preflightError(PreflightProtocol, "protocol")
	}
	return Authority{
		definitionID:              definitionID,
		version:                   version,
		behaviorFingerprint:       behaviorFingerprint,
		configurationDefinitionID: configurationDefinitionID,
		configurationFingerprint:  configurationFingerprint,
		requiredCapabilities:      append([]string(nil), requiredCapabilities...),
		phaseLimits:               append([]PhaseLimit(nil), phaseLimits...),
		seed:                      seed,
		attempt:                   attempt,
		participantDefinitionID:   participantDefinitionID,
		protocolDefinitionID:      protocolDefinitionID,
		protocolVersion:           protocolVersion,
		participantCount:          participantCount,
		programCount:              programCount,
		program:                   program.clone(),
	}, nil
}

func (a Authority) DefinitionID() string        { return a.definitionID }
func (a Authority) Version() uint64             { return a.version }
func (a Authority) BehaviorFingerprint() string { return a.behaviorFingerprint }
func (a Authority) ConfigurationDefinitionID() string {
	return a.configurationDefinitionID
}
func (a Authority) ConfigurationBehaviorFingerprint() string {
	return a.configurationFingerprint
}
func (a Authority) Seed() uint64                    { return a.seed }
func (a Authority) Attempt() uint64                 { return a.attempt }
func (a Authority) ParticipantDefinitionID() string { return a.participantDefinitionID }
func (a Authority) ProtocolDefinitionID() string    { return a.protocolDefinitionID }
func (a Authority) ProtocolVersion() uint64         { return a.protocolVersion }
func (a Authority) ParticipantCount() uint64        { return a.participantCount }
func (a Authority) ProgramCount() uint64            { return a.programCount }
func (a Authority) Program() Program                { return a.program.clone() }

func (a Authority) RequiredCapabilityDefinitionIDs() []string {
	return append([]string(nil), a.requiredCapabilities...)
}

func (a Authority) PhaseLimits() []PhaseLimit {
	return append([]PhaseLimit(nil), a.phaseLimits...)
}

func (a Authority) clone() Authority {
	cloned := a
	cloned.requiredCapabilities = append([]string(nil), a.requiredCapabilities...)
	cloned.phaseLimits = append([]PhaseLimit(nil), a.phaseLimits...)
	cloned.program = a.program.clone()
	return cloned
}

// CorrelationKind names one closed run-owned identity.
type CorrelationKind string

const (
	CorrelationWorkflow    CorrelationKind = "workflow"
	CorrelationOperation   CorrelationKind = "operation"
	CorrelationTaskQueue   CorrelationKind = "task-queue"
	CorrelationWorker      CorrelationKind = "worker"
	CorrelationParticipant CorrelationKind = "participant"
)

var correlationOrder = [...]CorrelationKind{
	CorrelationWorkflow,
	CorrelationOperation,
	CorrelationTaskQueue,
	CorrelationWorker,
	CorrelationParticipant,
}

// Correlation is one immutable identity derived from a checked run identity.
type Correlation struct {
	kind     CorrelationKind
	identity string
}

func (c Correlation) Kind() CorrelationKind { return c.kind }
func (c Correlation) Identity() string      { return c.identity }

func deriveCorrelations(runIdentity string) ([]Correlation, error) {
	correlations := make([]Correlation, 0, len(correlationOrder))
	for _, kind := range correlationOrder {
		identity := runIdentity + "." + string(kind)
		if !validIdentity(identity) {
			return nil, preflightError(PreflightRunIdentity, "derived-correlation")
		}
		for _, existing := range correlations {
			if existing.identity == identity {
				return nil, preflightError(PreflightDuplicate, "derived-correlation")
			}
		}
		correlations = append(correlations, Correlation{kind: kind, identity: identity})
	}
	return correlations, nil
}

func validateIdentitySet(
	values []string,
	kind PreflightErrorKind,
	subject string,
) error {
	if values == nil || !slices.IsSorted(values) {
		return preflightError(kind, subject)
	}
	for index, value := range values {
		if !validIdentity(value) {
			return preflightError(kind, subject)
		}
		if index > 0 && value == values[index-1] {
			return preflightError(PreflightDuplicate, subject)
		}
	}
	return nil
}

func validIdentity(value string) bool {
	if len(value) == 0 || len(value) > MaximumIdentityBytes {
		return false
	}
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, character := range []byte(segment) {
			if !asciiAlphaNumeric(character) && character != '-' && character != '_' {
				return false
			}
		}
	}
	return true
}

func asciiAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}
