package scenario

import (
	"fmt"
	"time"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const (
	SuiteFormatVersion   = "umpire3/compiler-suite/v1"
	ExplainFormatVersion = "umpire3/compiler-explain/v3"
)

type ErrorCategory string

const (
	ErrorInvalidIntent          ErrorCategory = "invalid-intent"
	ErrorCycle                  ErrorCategory = "cycle"
	ErrorAmbiguousProducer      ErrorCategory = "ambiguous-producer"
	ErrorMissingProjection      ErrorCategory = "missing-projection"
	ErrorRebind                 ErrorCategory = "rebind"
	ErrorTypeMismatch           ErrorCategory = "type-mismatch"
	ErrorIncompleteEnumeration  ErrorCategory = "incomplete-enumeration"
	ErrorLimitExceeded          ErrorCategory = "limit-exceeded"
	ErrorSemanticallyImpossible ErrorCategory = "semantically-impossible"
)

type Source struct {
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

type Error struct {
	Category ErrorCategory
	Source   Source
	Detail   string
}

func (e *Error) Error() string {
	location := ""
	if e.Source.File != "" {
		location = e.Source.File
		if e.Source.Line > 0 {
			location += fmt.Sprintf(":%d", e.Source.Line)
		}
		location += ": "
	}
	return location + string(e.Category) + ": " + e.Detail
}

type Limits struct {
	MaxPaths       int
	MaxActions     int
	MaxStates      int
	MaxMemoryBytes int64
	MaxTime        time.Duration
}

type Resource struct {
	Identifier string                     `json:"identifier"`
	Kind       protocolcatalog.EntityKind `json:"kind"`
}

type Scenario struct {
	Identifier string
	Target     protocolcatalog.TargetID
	Resources  []Resource
	Root       Term
}

type Symbol struct {
	Name string
	Type protocolcatalog.SemanticTypeID
}

func (s Symbol) Value() protocolexperiment.Value {
	name := s.Name
	return protocolexperiment.Value{Type: protocolexperiment.ValueSymbol, Text: &name}
}

type Projection struct {
	ProducerAction string
	Name           string
	Type           protocolcatalog.SemanticTypeID
}

func Project(producerAction, name string, valueType protocolcatalog.SemanticTypeID) Projection {
	return Projection{ProducerAction: producerAction, Name: name, Type: valueType}
}

type ActionOption func(*actionIntent)

type Value struct {
	value protocolexperiment.Value
}

type Field struct {
	name  string
	value Value
}

func String(value string) Value {
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &value}}
}

func Integer(value int64) Value {
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueInteger, Integer: &value}}
}

func Boolean(value bool) Value {
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueBoolean, Boolean: &value}}
}

func Duration(value time.Duration) Value {
	nanoseconds := int64(value)
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueDuration, Integer: &nanoseconds}}
}

func Enum(name string, number int64) Value {
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueEnum, Text: &name, Integer: &number}}
}

func BytesDigest(value string) Value {
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueBytesDigest, Text: &value}}
}

func SymbolValue(symbol Symbol) Value {
	return Value{value: symbol.Value()}
}

func List(values ...Value) Value {
	elements := make([]protocolexperiment.Value, len(values))
	for index, value := range values {
		elements[index] = value.value
	}
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueList, Elements: elements}}
}

func Named(name string, value Value) Field {
	return Field{name: name, value: value}
}

func Record(fields ...Field) Value {
	values := make([]protocolexperiment.NamedValue, len(fields))
	for index, field := range fields {
		values[index] = protocolexperiment.NamedValue{Name: field.name, Value: field.value.value}
	}
	return Value{value: protocolexperiment.Value{Type: protocolexperiment.ValueRecord, Fields: values}}
}

type Outcome string

const (
	Applied          Outcome = "applied"
	Suppressed       Outcome = "suppressed"
	Rejected         Outcome = "rejected"
	Retried          Outcome = "retried"
	FaultIntercepted Outcome = "fault-intercepted"
)

func WithArgument(name string, value protocolexperiment.Value) ActionOption {
	return func(intent *actionIntent) {
		intent.arguments = append(intent.arguments, protocolexperiment.NamedValue{Name: name, Value: value})
	}
}

func WithValue(name string, value Value) ActionOption {
	return WithArgument(name, value.value)
}

func withStringArgument(name, value string) ActionOption {
	return WithArgument(name, protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &value})
}

func withIdentityArgument(name string, symbol Symbol) ActionOption {
	return WithArgument(name, symbol.Value())
}

func WithResponse(mode protocolexperiment.ResponseMode) ActionOption {
	return func(intent *actionIntent) {
		intent.responseMode = mode
	}
}

func WithBoundedBlock(duration time.Duration) ActionOption {
	return func(intent *actionIntent) {
		intent.responseMode = protocolexperiment.ResponseBlocking
		intent.maxBlockNanos = int64(duration)
	}
}

func WithOutcomes(outcomes ...protocolexperiment.ActionOutcome) ActionOption {
	return func(intent *actionIntent) {
		intent.allowedOutcomes = append([]protocolexperiment.ActionOutcome(nil), outcomes...)
	}
}

func Outcomes(outcomes ...Outcome) ActionOption {
	return func(intent *actionIntent) {
		intent.allowedOutcomes = make([]protocolexperiment.ActionOutcome, len(outcomes))
		for index, outcome := range outcomes {
			intent.allowedOutcomes[index] = protocolexperiment.ActionOutcome(outcome)
		}
	}
}

type FaultOption func(*FaultIntent)

type FaultIntent struct {
	identifier string
	kind       protocolcatalog.FaultKind
	arguments  []protocolexperiment.NamedValue
	scope      protocolexperiment.FaultScope
	occurrence protocolexperiment.FaultOccurrence
	configured *protocolexperiment.Fault
	source     Source
}

func Fault(identifier string, kind protocolcatalog.FaultKind, options ...FaultOption) FaultIntent {
	return FaultAt(Source{}, identifier, kind, options...)
}

func FaultAt(source Source, identifier string, kind protocolcatalog.FaultKind, options ...FaultOption) FaultIntent {
	intent := FaultIntent{identifier: identifier, kind: kind, source: source}
	for _, option := range options {
		option(&intent)
	}
	return intent
}

func OnResources(resources ...Resource) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Resources = make([]string, len(resources))
		for index, resource := range resources {
			intent.scope.Resources[index] = resource.Identifier
		}
	}
}

func OnEndpoints(endpoints ...string) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Endpoints = append([]string(nil), endpoints...)
	}
}

func OnTaskQueues(taskQueues ...string) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.TaskQueues = append([]string(nil), taskQueues...)
	}
}

func OnServices(services ...string) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Services = append([]string(nil), services...)
	}
}

func OnRoutes(routes ...string) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Routes = append([]string(nil), routes...)
	}
}

func OnParticipants(participants ...string) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Participants = append([]string(nil), participants...)
	}
}

func OnAttempts(attempts ...int) FaultOption {
	return func(intent *FaultIntent) {
		intent.scope.Attempts = append([]int(nil), attempts...)
	}
}

func AtOccurrence(first, count int) FaultOption {
	return func(intent *FaultIntent) {
		intent.occurrence = protocolexperiment.FaultOccurrence{First: first, Count: count}
	}
}

func WithFaultValue(name string, value Value) FaultOption {
	return func(intent *FaultIntent) {
		intent.arguments = append(intent.arguments, protocolexperiment.NamedValue{Name: name, Value: value.value})
	}
}

func ConfiguredFault(fault protocolexperiment.Fault) FaultIntent {
	clone := fault
	clone.Arguments = append([]protocolexperiment.NamedValue(nil), fault.Arguments...)
	clone.RequiredCapabilities = append([]string(nil), fault.RequiredCapabilities...)
	clone.Scope.Resources = append([]string(nil), fault.Scope.Resources...)
	clone.Scope.Endpoints = append([]string(nil), fault.Scope.Endpoints...)
	clone.Scope.TaskQueues = append([]string(nil), fault.Scope.TaskQueues...)
	clone.Scope.Services = append([]string(nil), fault.Scope.Services...)
	clone.Scope.Routes = append([]string(nil), fault.Scope.Routes...)
	clone.Scope.Participants = append([]string(nil), fault.Scope.Participants...)
	clone.Scope.Attempts = append([]int(nil), fault.Scope.Attempts...)
	return FaultIntent{
		identifier: fault.Identifier, kind: protocolcatalog.FaultKind(fault.Kind),
		arguments: append([]protocolexperiment.NamedValue(nil), fault.Arguments...), configured: &clone,
	}
}

type nodeKind uint8

const (
	nodeInvalid nodeKind = iota
	nodeAction
	nodeBind
	nodeRequire
	nodeOnePath
	nodeAllPaths
	nodeAnyOrder
	nodeBefore
	nodeDuring
	nodeRepeat
)

type actionIntent struct {
	identifier      string
	kind            protocolcatalog.ActionKind
	allowedOutcomes []protocolexperiment.ActionOutcome
	arguments       []protocolexperiment.NamedValue
	responseMode    protocolexperiment.ResponseMode
	maxBlockNanos   int64
}

type bindIntent struct {
	symbol     Symbol
	projection Projection
}

type Term struct {
	kind        nodeKind
	source      Source
	action      actionIntent
	bind        bindIntent
	property    protocolcatalog.PropertyID
	children    []Term
	fault       FaultIntent
	repeatCount int
}

func ActionAt(source Source, identifier string, kind protocolcatalog.ActionKind, options ...ActionOption) Term {
	intent := actionIntent{identifier: identifier, kind: kind, responseMode: protocolexperiment.ResponseSynchronous}
	for _, option := range options {
		option(&intent)
	}
	return Term{kind: nodeAction, source: source, action: intent}
}

func BindAt(source Source, symbol Symbol, projection Projection) Term {
	return Term{kind: nodeBind, source: source, bind: bindIntent{symbol: symbol, projection: projection}}
}

func RequireAt(source Source, property protocolcatalog.PropertyID) Term {
	return Term{kind: nodeRequire, source: source, property: property}
}

func OnePathAt(source Source, children ...Term) Term {
	return Term{kind: nodeOnePath, source: source, children: append([]Term(nil), children...)}
}

func AllPathsAt(source Source, children ...Term) Term {
	return Term{kind: nodeAllPaths, source: source, children: append([]Term(nil), children...)}
}

func AnyOrderAt(source Source, children ...Term) Term {
	return Term{kind: nodeAnyOrder, source: source, children: append([]Term(nil), children...)}
}

func BeforeAt(source Source, before, after Term) Term {
	return Term{kind: nodeBefore, source: source, children: []Term{before, after}}
}

func DuringAt(source Source, fault FaultIntent, body Term) Term {
	return Term{kind: nodeDuring, source: source, fault: fault, children: []Term{body}}
}

func RepeatAt(source Source, count int, body Term) Term {
	return Term{kind: nodeRepeat, source: source, repeatCount: count, children: []Term{body}}
}

type Enumeration struct {
	Mode        string `json:"mode"`
	States      int    `json:"states"`
	Paths       int    `json:"paths"`
	MaxPaths    int    `json:"maxPaths"`
	MaxActions  int    `json:"maxActions"`
	MaxStates   int    `json:"maxStates"`
	MemoryBytes int64  `json:"memoryBytes"`
}

type IdentityRecord struct {
	Symbol          string   `json:"symbol"`
	Type            string   `json:"type"`
	ProducerAction  string   `json:"producerAction"`
	Projection      string   `json:"projection"`
	ConsumerActions []string `json:"consumerActions"`
}

type ModelReplayStatus string

const (
	ModelReplayChecked      ModelReplayStatus = "checked"
	ModelReplayNotSupported ModelReplayStatus = "not-supported"
)

type ModelReplay struct {
	Status          ModelReplayStatus            `json:"status"`
	CanonicalModel  string                       `json:"canonicalModel,omitempty"`
	Variant         string                       `json:"variant,omitempty"`
	LiveOnlyActions []protocolcatalog.ActionKind `json:"liveOnlyActions"`
	Reason          string                       `json:"reason,omitempty"`
}

type Explain struct {
	FormatVersion    string                               `json:"formatVersion"`
	Scenario         string                               `json:"scenario"`
	ScenarioDigest   string                               `json:"scenarioDigest"`
	CatalogHash      string                               `json:"catalogHash"`
	Target           protocolcatalog.TargetID             `json:"target"`
	Property         protocolcatalog.PropertyID           `json:"property"`
	AddedActionKinds []string                             `json:"addedActionKinds"`
	Constraints      []protocolexperiment.OrderConstraint `json:"constraints"`
	Identities       []IdentityRecord                     `json:"identities"`
	Paths            [][]string                           `json:"paths"`
	Omissions        []protocolcatalog.ProjectionOmission `json:"omissions"`
	ModelReplay      ModelReplay                          `json:"modelReplay"`
	Enumeration      Enumeration                          `json:"enumeration"`
}

type Suite struct {
	FormatVersion  string                          `json:"formatVersion"`
	ScenarioDigest string                          `json:"scenarioDigest"`
	Experiments    []protocolexperiment.Experiment `json:"experiments"`
	Digests        []string                        `json:"digests"`
	Explain        Explain                         `json:"explain"`
}
