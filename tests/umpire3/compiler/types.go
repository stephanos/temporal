package compiler

import (
	"fmt"
	"time"

	"go.temporal.io/server/tests/umpire3/protocol"
)

const (
	SuiteFormatVersion   = "umpire3/compiler-suite/v1"
	ExplainFormatVersion = "umpire3/compiler-explain/v1"
)

type ErrorCategory string

const (
	ErrorInvalidIntent         ErrorCategory = "invalid-intent"
	ErrorCycle                 ErrorCategory = "cycle"
	ErrorAmbiguousProducer     ErrorCategory = "ambiguous-producer"
	ErrorMissingProjection     ErrorCategory = "missing-projection"
	ErrorRebind                ErrorCategory = "rebind"
	ErrorTypeMismatch          ErrorCategory = "type-mismatch"
	ErrorIncompleteEnumeration ErrorCategory = "incomplete-enumeration"
	ErrorLimitExceeded         ErrorCategory = "limit-exceeded"
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
	Root       Node
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

type Node struct {
	kind        nodeKind
	source      Source
	action      actionIntent
	bind        bindIntent
	property    protocol.PropertyID
	children    []Node
	fault       FaultIntent
	repeatCount int
}

func Action(identifier string, kind protocol.ActionKind, options ...ActionOption) Node {
	return ActionAt(Source{}, identifier, kind, options...)
}

func ActionAt(source Source, identifier string, kind protocol.ActionKind, options ...ActionOption) Node {
	intent := actionIntent{identifier: identifier, kind: kind, responseMode: protocol.ResponseSynchronous}
	for _, option := range options {
		option(&intent)
	}
	return Node{kind: nodeAction, source: source, action: intent}
}

func Bind(symbol Symbol, projection Projection) Node {
	return BindAt(Source{}, symbol, projection)
}

func BindAt(source Source, symbol Symbol, projection Projection) Node {
	return Node{kind: nodeBind, source: source, bind: bindIntent{symbol: symbol, projection: projection}}
}

func Require(property protocol.PropertyID) Node {
	return RequireAt(Source{}, property)
}

func RequireAt(source Source, property protocol.PropertyID) Node {
	return Node{kind: nodeRequire, source: source, property: property}
}

func OnePath(children ...Node) Node {
	return OnePathAt(Source{}, children...)
}

func OnePathAt(source Source, children ...Node) Node {
	return Node{kind: nodeOnePath, source: source, children: append([]Node(nil), children...)}
}

func AllPaths(children ...Node) Node {
	return AllPathsAt(Source{}, children...)
}

func AllPathsAt(source Source, children ...Node) Node {
	return Node{kind: nodeAllPaths, source: source, children: append([]Node(nil), children...)}
}

func AnyOrder(children ...Node) Node {
	return AnyOrderAt(Source{}, children...)
}

func AnyOrderAt(source Source, children ...Node) Node {
	return Node{kind: nodeAnyOrder, source: source, children: append([]Node(nil), children...)}
}

func Before(before, after Node) Node {
	return BeforeAt(Source{}, before, after)
}

func BeforeAt(source Source, before, after Node) Node {
	return Node{kind: nodeBefore, source: source, children: []Node{before, after}}
}

func During(fault FaultIntent, body Node) Node {
	return DuringAt(Source{}, fault, body)
}

func DuringAt(source Source, fault FaultIntent, body Node) Node {
	return Node{kind: nodeDuring, source: source, fault: fault, children: []Node{body}}
}

func Repeat(count int, body Node) Node {
	return RepeatAt(Source{}, count, body)
}

func RepeatAt(source Source, count int, body Node) Node {
	return Node{kind: nodeRepeat, source: source, repeatCount: count, children: []Node{body}}
}

type Enumeration struct {
	Mode        string `json:"mode"`
	Complete    bool   `json:"complete"`
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
	Enumeration      Enumeration                   `json:"enumeration"`
}

type Suite struct {
	FormatVersion  string                `json:"formatVersion"`
	ScenarioDigest string                `json:"scenarioDigest"`
	Experiments    []protocol.Experiment `json:"experiments"`
	Digests        []string              `json:"digests"`
	Explain        Explain               `json:"explain"`
}
