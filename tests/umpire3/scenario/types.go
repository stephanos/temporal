package scenario

import (
	"fmt"
	"time"

	"go.temporal.io/server/tests/umpire3/protocol"
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
	Identifier string              `json:"identifier"`
	Kind       protocol.EntityKind `json:"kind"`
}

type Scenario struct {
	Identifier string
	Target     protocol.TargetID
	Resources  []Resource
	Root       Term
}

type Symbol struct {
	Name string
	Type protocol.SemanticTypeID
}

func (s Symbol) Value() protocol.Value {
	name := s.Name
	return protocol.Value{Type: protocol.ValueSymbol, Text: &name}
}

type Projection struct {
	ProducerAction string
	Name           string
	Type           protocol.SemanticTypeID
}

func Project(producerAction, name string, valueType protocol.SemanticTypeID) Projection {
	return Projection{ProducerAction: producerAction, Name: name, Type: valueType}
}

type ActionOption func(*actionIntent)

func WithArgument(name string, value protocol.Value) ActionOption {
	return func(intent *actionIntent) {
		intent.arguments = append(intent.arguments, protocol.NamedValue{Name: name, Value: value})
	}
}

func WithResponse(mode protocol.ResponseMode) ActionOption {
	return func(intent *actionIntent) {
		intent.responseMode = mode
	}
}

func WithBoundedBlock(duration time.Duration) ActionOption {
	return func(intent *actionIntent) {
		intent.responseMode = protocol.ResponseBlocking
		intent.maxBlockNanos = int64(duration)
	}
}

type FaultIntent struct {
	identifier string
	kind       protocol.FaultKind
	arguments  []protocol.NamedValue
	configured *protocol.Fault
	source     Source
}

func Fault(identifier string, kind protocol.FaultKind, options ...ActionOption) FaultIntent {
	return FaultAt(Source{}, identifier, kind, options...)
}

func FaultAt(source Source, identifier string, kind protocol.FaultKind, options ...ActionOption) FaultIntent {
	intent := actionIntent{}
	for _, option := range options {
		option(&intent)
	}
	return FaultIntent{identifier: identifier, kind: kind, arguments: intent.arguments, source: source}
}

func ConfiguredFault(fault protocol.Fault) FaultIntent {
	clone := fault
	clone.Arguments = append([]protocol.NamedValue(nil), fault.Arguments...)
	clone.RequiredCapabilities = append([]string(nil), fault.RequiredCapabilities...)
	clone.Scope.Resources = append([]string(nil), fault.Scope.Resources...)
	clone.Scope.Endpoints = append([]string(nil), fault.Scope.Endpoints...)
	clone.Scope.TaskQueues = append([]string(nil), fault.Scope.TaskQueues...)
	clone.Scope.Services = append([]string(nil), fault.Scope.Services...)
	clone.Scope.Routes = append([]string(nil), fault.Scope.Routes...)
	clone.Scope.Participants = append([]string(nil), fault.Scope.Participants...)
	clone.Scope.Attempts = append([]int(nil), fault.Scope.Attempts...)
	return FaultIntent{
		identifier: fault.Identifier, kind: protocol.FaultKind(fault.Kind),
		arguments: append([]protocol.NamedValue(nil), fault.Arguments...), configured: &clone,
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
	identifier    string
	kind          protocol.ActionKind
	arguments     []protocol.NamedValue
	responseMode  protocol.ResponseMode
	maxBlockNanos int64
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
	property    protocol.PropertyID
	children    []Term
	fault       FaultIntent
	repeatCount int
}

func ActionAt(source Source, identifier string, kind protocol.ActionKind, options ...ActionOption) Term {
	intent := actionIntent{identifier: identifier, kind: kind, responseMode: protocol.ResponseSynchronous}
	for _, option := range options {
		option(&intent)
	}
	return Term{kind: nodeAction, source: source, action: intent}
}

func BindAt(source Source, symbol Symbol, projection Projection) Term {
	return Term{kind: nodeBind, source: source, bind: bindIntent{symbol: symbol, projection: projection}}
}

func RequireAt(source Source, property protocol.PropertyID) Term {
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
	Status          ModelReplayStatus     `json:"status"`
	CanonicalModel  string                `json:"canonicalModel,omitempty"`
	Variant         string                `json:"variant,omitempty"`
	LiveOnlyActions []protocol.ActionKind `json:"liveOnlyActions"`
	Reason          string                `json:"reason,omitempty"`
}

type Explain struct {
	FormatVersion    string                        `json:"formatVersion"`
	Scenario         string                        `json:"scenario"`
	ScenarioDigest   string                        `json:"scenarioDigest"`
	CatalogHash      string                        `json:"catalogHash"`
	Target           protocol.TargetID             `json:"target"`
	Property         protocol.PropertyID           `json:"property"`
	AddedActionKinds []string                      `json:"addedActionKinds"`
	Constraints      []protocol.OrderConstraint    `json:"constraints"`
	Identities       []IdentityRecord              `json:"identities"`
	Paths            [][]string                    `json:"paths"`
	Omissions        []protocol.ProjectionOmission `json:"omissions"`
	ModelReplay      ModelReplay                   `json:"modelReplay"`
	Enumeration      Enumeration                   `json:"enumeration"`
}

type Suite struct {
	FormatVersion  string                `json:"formatVersion"`
	ScenarioDigest string                `json:"scenarioDigest"`
	Experiments    []protocol.Experiment `json:"experiments"`
	Digests        []string              `json:"digests"`
	Explain        Explain               `json:"explain"`
}
