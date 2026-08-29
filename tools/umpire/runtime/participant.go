package runtime

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

// Occurrence is one checked action occurrence accepted by a participant program.
type Occurrence struct {
	definitionID       string
	actionDefinitionID string
	position           uint64
}

// NewOccurrence checks one occurrence without interpreting its action.
func NewOccurrence(definitionID string, actionDefinitionID string, position uint64) (Occurrence, error) {
	if !validIdentity(definitionID) {
		return Occurrence{}, preflightError(PreflightOccurrence, "occurrence")
	}
	if !validIdentity(actionDefinitionID) {
		return Occurrence{}, preflightError(PreflightAction, "occurrence-action")
	}
	if position == 0 {
		return Occurrence{}, preflightError(PreflightOccurrence, "occurrence-position")
	}
	return Occurrence{
		definitionID: definitionID, actionDefinitionID: actionDefinitionID, position: position,
	}, nil
}

func (o Occurrence) DefinitionID() string       { return o.definitionID }
func (o Occurrence) ActionDefinitionID() string { return o.actionDefinitionID }
func (o Occurrence) Position() uint64           { return o.position }

// CommandKind is one of the four closed participant operations.
type CommandKind string

const (
	CommandPrepare CommandKind = "prepare"
	CommandRealize CommandKind = "realize"
	CommandObserve CommandKind = "observe"
	CommandCleanup CommandKind = "cleanup"
)

var commandOrder = [...]CommandKind{
	CommandPrepare,
	CommandRealize,
	CommandObserve,
	CommandCleanup,
}

const commandIsolate CommandKind = "isolate"

// Program is checked inert participant metadata, never executable behavior.
type Program struct {
	definitionID            string
	version                 uint64
	behaviorFingerprint     string
	targetDefinitionIDs     []string
	actionDefinitionIDs     []string
	occurrences             []Occurrence
	capabilityDefinitionIDs []string
}

// NewProgram checks the one closed program shape without adding a callback or plugin surface.
func NewProgram(
	definitionID string,
	version uint64,
	behaviorFingerprint string,
	targetDefinitionIDs []string,
	actionDefinitionIDs []string,
	occurrences []Occurrence,
	capabilityDefinitionIDs []string,
) (Program, error) {
	if !validIdentity(definitionID) || version != 2 || !artifactv2.ValidDigest(behaviorFingerprint) {
		return Program{}, preflightError(PreflightParticipant, "program")
	}
	if len(targetDefinitionIDs) != 1 {
		return Program{}, preflightError(PreflightTarget, "program-targets")
	}
	if err := validateIdentitySet(targetDefinitionIDs, PreflightTarget, "program-targets"); err != nil {
		return Program{}, err
	}
	if len(actionDefinitionIDs) != 1 {
		return Program{}, preflightError(PreflightAction, "program-actions")
	}
	if err := validateIdentitySet(actionDefinitionIDs, PreflightAction, "program-actions"); err != nil {
		return Program{}, err
	}
	if len(occurrences) != 1 || occurrences[0].definitionID == "" || occurrences[0].position != 1 {
		return Program{}, preflightError(PreflightOccurrence, "program-occurrences")
	}
	if occurrences[0].actionDefinitionID != actionDefinitionIDs[0] {
		return Program{}, preflightError(PreflightAction, "program-occurrence-action")
	}
	if len(capabilityDefinitionIDs) == 0 ||
		len(capabilityDefinitionIDs) > artifact.MaximumJSONArrayItems {
		return Program{}, preflightError(PreflightCapability, "program-capabilities")
	}
	if err := validateIdentitySet(
		capabilityDefinitionIDs, PreflightCapability, "program-capabilities",
	); err != nil {
		return Program{}, err
	}
	return Program{
		definitionID:            definitionID,
		version:                 version,
		behaviorFingerprint:     behaviorFingerprint,
		targetDefinitionIDs:     slices.Clone(targetDefinitionIDs),
		actionDefinitionIDs:     slices.Clone(actionDefinitionIDs),
		occurrences:             slices.Clone(occurrences),
		capabilityDefinitionIDs: slices.Clone(capabilityDefinitionIDs),
	}, nil
}

func (p Program) DefinitionID() string        { return p.definitionID }
func (p Program) Version() uint64             { return p.version }
func (p Program) BehaviorFingerprint() string { return p.behaviorFingerprint }
func (p Program) Occurrence() Occurrence {
	if len(p.occurrences) == 0 {
		return Occurrence{}
	}
	return p.occurrences[0]
}

func (p Program) TargetDefinitionIDs() []string {
	return slices.Clone(p.targetDefinitionIDs)
}

func (p Program) ActionDefinitionIDs() []string {
	return slices.Clone(p.actionDefinitionIDs)
}

func (p Program) Occurrences() []Occurrence {
	return slices.Clone(p.occurrences)
}

func (p Program) CapabilityDefinitionIDs() []string {
	return slices.Clone(p.capabilityDefinitionIDs)
}

func (p Program) CommandKinds() []CommandKind {
	return slices.Clone(commandOrder[:])
}

func (p Program) clone() Program {
	cloned := p
	cloned.targetDefinitionIDs = slices.Clone(p.targetDefinitionIDs)
	cloned.actionDefinitionIDs = slices.Clone(p.actionDefinitionIDs)
	cloned.occurrences = slices.Clone(p.occurrences)
	cloned.capabilityDefinitionIDs = slices.Clone(p.capabilityDefinitionIDs)
	return cloned
}

// Command binds one closed program operation to a checked request.
type Command struct {
	kind                   CommandKind
	phase                  Phase
	programDefinitionID    string
	programFingerprint     string
	runIdentity            string
	targetDefinitionID     string
	actionDefinitionID     string
	occurrenceDefinitionID string
	attempt                uint64
	limit                  PhaseLimit
}

func (c Command) Kind() CommandKind                  { return c.kind }
func (c Command) Phase() Phase                       { return c.phase }
func (c Command) ProgramDefinitionID() string        { return c.programDefinitionID }
func (c Command) ProgramBehaviorFingerprint() string { return c.programFingerprint }
func (c Command) RunIdentity() string                { return c.runIdentity }
func (c Command) TargetDefinitionID() string         { return c.targetDefinitionID }
func (c Command) ActionDefinitionID() string         { return c.actionDefinitionID }
func (c Command) OccurrenceDefinitionID() string     { return c.occurrenceDefinitionID }
func (c Command) Attempt() uint64                    { return c.attempt }
func (c Command) Limit() PhaseLimit                  { return c.limit }

// ResourceKind is one closed tracked-handle category.
type ResourceKind string

const (
	ResourceEnvironment ResourceKind = "environment"
	ResourceConnection  ResourceKind = "connection"
	ResourceWorker      ResourceKind = "worker"
	ResourceParticipant ResourceKind = "participant"
)

// Resource is an immutable tracked handle identity, never the live handle itself.
type Resource struct {
	kind     ResourceKind
	identity string
}

// NewResource checks one bounded resource identity.
func NewResource(kind ResourceKind, identity string) (Resource, error) {
	switch kind {
	case ResourceEnvironment, ResourceConnection, ResourceWorker, ResourceParticipant:
	default:
		return Resource{}, fmt.Errorf("invalid resource kind")
	}
	if !validIdentity(identity) {
		return Resource{}, fmt.Errorf("invalid resource identity")
	}
	return Resource{kind: kind, identity: identity}, nil
}

func (r Resource) Kind() ResourceKind { return r.kind }
func (r Resource) Identity() string   { return r.identity }

// FactField is one bounded retained mechanical string field.
type FactField struct {
	definitionID string
	value        string
}

// NewFactField checks one field identity and bounded value.
func NewFactField(definitionID string, value string) (FactField, error) {
	if !validIdentity(definitionID) {
		return FactField{}, fmt.Errorf("invalid fact field identity")
	}
	if !validFactValue(value) {
		return FactField{}, fmt.Errorf("fact field value exceeds limit")
	}
	return FactField{definitionID: definitionID, value: value}, nil
}

func validFactValue(value string) bool {
	if len(value) == 0 || len(value) > MaximumFactValueBytes {
		return false
	}
	for _, character := range []byte(value) {
		if !asciiAlphaNumeric(character) && character != '.' && character != '-' &&
			character != '_' && character != ':' {
			return false
		}
	}
	return true
}

func (f FactField) DefinitionID() string { return f.definitionID }
func (f FactField) Value() string        { return f.value }

// Fact is one immutable, bounded participant fact awaiting source-local ordering.
type Fact struct {
	definitionID        string
	sourceDefinitionID  string
	kindDefinitionID    string
	causalDefinitionIDs []string
	fields              []FactField
}

// NewFact checks deterministic field and causal-reference order before retaining a fact.
func NewFact(
	definitionID string,
	sourceDefinitionID string,
	kindDefinitionID string,
	causalDefinitionIDs []string,
	fields []FactField,
) (Fact, error) {
	if !validIdentity(definitionID) || !validIdentity(sourceDefinitionID) ||
		!validIdentity(kindDefinitionID) {
		return Fact{}, fmt.Errorf("invalid fact identity")
	}
	if len(causalDefinitionIDs) > artifact.MaximumJSONArrayItems {
		return Fact{}, fmt.Errorf("fact causal identities exceed limit")
	}
	if err := validateIdentitySet(causalDefinitionIDs, PreflightCapability, "fact-causes"); err != nil {
		return Fact{}, fmt.Errorf("invalid fact causal identities")
	}
	if fields == nil || len(fields) > artifact.MaximumFieldsPerEvidenceFact {
		return Fact{}, fmt.Errorf("fact fields exceed limit")
	}
	if !slices.IsSortedFunc(fields, func(left, right FactField) int {
		return strings.Compare(left.definitionID, right.definitionID)
	}) {
		return Fact{}, fmt.Errorf("fact fields are not in canonical order")
	}
	payloadBytes := 0
	for index, field := range fields {
		if field.definitionID == "" {
			return Fact{}, fmt.Errorf("fact has unchecked field")
		}
		if index > 0 && field.definitionID == fields[index-1].definitionID {
			return Fact{}, fmt.Errorf("fact repeats field")
		}
		payloadBytes += len(field.definitionID) + len(field.value)
	}
	if payloadBytes > artifact.MaximumEvidenceFactPayloadBytes {
		return Fact{}, fmt.Errorf("fact payload exceeds limit")
	}
	return Fact{
		definitionID: definitionID, sourceDefinitionID: sourceDefinitionID,
		kindDefinitionID:    kindDefinitionID,
		causalDefinitionIDs: slices.Clone(causalDefinitionIDs),
		fields:              slices.Clone(fields),
	}, nil
}

func (f Fact) DefinitionID() string       { return f.definitionID }
func (f Fact) SourceDefinitionID() string { return f.sourceDefinitionID }
func (f Fact) KindDefinitionID() string   { return f.kindDefinitionID }

func (f Fact) CausalDefinitionIDs() []string {
	return slices.Clone(f.causalDefinitionIDs)
}

func (f Fact) Fields() []FactField {
	return slices.Clone(f.fields)
}

// ReceiptStatus is one terminal participant-command status.
type ReceiptStatus string

const (
	ReceiptAccepted    ReceiptStatus = "accepted"
	ReceiptRejected    ReceiptStatus = "rejected"
	ReceiptUnsupported ReceiptStatus = "unsupported"
	ReceiptFailed      ReceiptStatus = "failed"
	ReceiptCanceled    ReceiptStatus = "canceled"
)

// Receipt is a command-bound immutable terminal result.
type Receipt struct {
	command           Command
	status            ReceiptStatus
	facts             []Fact
	acquiredResources []Resource
	releasedResources []Resource
}

// NewReceipt validates bounded canonical fact and resource collections.
func NewReceipt(
	command Command,
	status ReceiptStatus,
	facts []Fact,
	acquiredResources []Resource,
	releasedResources []Resource,
) (Receipt, error) {
	if command.runIdentity == "" {
		return Receipt{}, fmt.Errorf("receipt requires a checked command")
	}
	switch status {
	case ReceiptAccepted, ReceiptRejected, ReceiptUnsupported, ReceiptFailed, ReceiptCanceled:
	default:
		return Receipt{}, fmt.Errorf("invalid receipt status")
	}
	if facts == nil || uint64(len(facts)) > command.limit.maxRecords {
		return Receipt{}, fmt.Errorf("receipt facts exceed phase limit")
	}
	if err := validateFacts(facts, command.limit.maxBytes); err != nil {
		return Receipt{}, err
	}
	if err := validateResources(acquiredResources); err != nil {
		return Receipt{}, err
	}
	if err := validateResources(releasedResources); err != nil {
		return Receipt{}, err
	}
	return Receipt{
		command: command, status: status,
		facts:             cloneFacts(facts),
		acquiredResources: slices.Clone(acquiredResources),
		releasedResources: slices.Clone(releasedResources),
	}, nil
}

func (r Receipt) Command() Command      { return r.command }
func (r Receipt) Status() ReceiptStatus { return r.status }
func (r Receipt) Facts() []Fact         { return cloneFacts(r.facts) }

func (r Receipt) AcquiredResources() []Resource {
	return slices.Clone(r.acquiredResources)
}

func (r Receipt) ReleasedResources() []Resource {
	return slices.Clone(r.releasedResources)
}

func validateFacts(facts []Fact, maxBytes uint64) error {
	if !slices.IsSortedFunc(facts, func(left, right Fact) int {
		return strings.Compare(left.definitionID, right.definitionID)
	}) {
		return fmt.Errorf("receipt facts are not in canonical order")
	}
	var payloadBytes uint64
	for index, fact := range facts {
		if fact.definitionID == "" {
			return fmt.Errorf("receipt has unchecked fact")
		}
		if index > 0 && fact.definitionID == facts[index-1].definitionID {
			return fmt.Errorf("receipt repeats fact")
		}
		payloadBytes += uint64(len(fact.definitionID) + len(fact.sourceDefinitionID) + len(fact.kindDefinitionID))
		for _, field := range fact.fields {
			payloadBytes += uint64(len(field.definitionID) + len(field.value))
		}
	}
	if payloadBytes > maxBytes {
		return fmt.Errorf("receipt fact bytes exceed phase limit")
	}
	return nil
}

func validateResources(resources []Resource) error {
	if resources == nil {
		return fmt.Errorf("resource collection must not be nil")
	}
	if len(resources) > artifact.MaximumJSONArrayItems {
		return fmt.Errorf("resource collection exceeds limit")
	}
	if !slices.IsSortedFunc(resources, compareResource) {
		return fmt.Errorf("resources are not in canonical order")
	}
	for index, resource := range resources {
		if resource.identity == "" {
			return fmt.Errorf("resource collection has unchecked value")
		}
		if index > 0 && compareResource(resource, resources[index-1]) == 0 {
			return fmt.Errorf("resource collection has duplicate")
		}
	}
	return nil
}

func compareResource(left, right Resource) int {
	if comparison := strings.Compare(string(left.kind), string(right.kind)); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.identity, right.identity)
}

func cloneFacts(facts []Fact) []Fact {
	cloned := slices.Clone(facts)
	for index := range cloned {
		cloned[index].causalDefinitionIDs = slices.Clone(
			facts[index].causalDefinitionIDs,
		)
		cloned[index].fields = slices.Clone(facts[index].fields)
	}
	return cloned
}

// EnvironmentFactory acquires one isolated environment during preparation.
type EnvironmentFactory interface {
	Prepare(context.Context, CheckedRunRequest, Command) (Environment, Receipt)
}

// Environment owns the bounded isolation and cleanup phases.
type Environment interface {
	Isolate(context.Context, Command) Receipt
	Cleanup(context.Context, Command) Receipt
}

// Participant executes only the four closed command kinds.
type Participant interface {
	Prepare(context.Context, Environment, Command) Receipt
	Realize(context.Context, Environment, Command) Receipt
	Observe(context.Context, Environment, Command) Receipt
	Cleanup(context.Context, Environment, Command) Receipt
}
