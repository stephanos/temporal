import Umpire.Property.Correlated
import Umpire.Case.Projection.Correlated
import Umpire.Shared.Test

/-!
A finite Target whose states are structured: an `open` or `done` phase and an attempt count, each a
field the Contract reads apart from the state that carries them. Several operations run the same
machine under their own keys, so a rule about one operation's attempt count is answered by that
operation's steps and no other's.

The rule reads a field: after a `fault`, the operation's `attempts` field is `1`. It holds on an
operation's first fault and fails on its second, which is what tells operations apart -- two
operations that each fault once satisfy it, one operation that faults twice does not.
-/

namespace Umpire.Property.CorrelatedStructuredTests

open Property.Correlated

def id := DefinitionId.of
def source : SourceLocation := { path := "Umpire/Property/Tests/Correlated/Structured.lean" }
def value (key payload : String) : ModelValue := ⟨id key, payload⟩

/-- The attempt count saturates at two: a third fault is a second one over again. -/
def attemptBound : Nat := 2

/-- A state, spelled from its two fields the way a machine spells one. -/
def stateOf (phase : String) (attempts : Nat) : ModelValue :=
  value "test.state" (phase ++ "-" ++ toString attempts)

/-- The fields that state holds, each under the field's own definition. -/
def fieldsOf (phase : String) (attempts : Nat) : List ModelValue :=
  [value "test.phase" phase, value "test.attempts" (toString attempts)]

def counts : List Nat := List.range (attemptBound + 1)
def phases : List String := ["open", "done"]

def request := value "test.request" "request"
def fault := value "test.fault" "fault"
def reply := value "test.reply" "reply"
def quiet := value "test.outcome" "quiet"
def response := value "test.outcome" "response"
def pendingAttempts := value "test.fact" "pendingAttempts"

def step (phase : String) (attempts : Nat) (outcome : ModelValue) (facts : List ModelValue := []) :
    Step ModelValue ModelValue ModelValue :=
  { state := stateOf phase attempts, outcome, facts }

def kinds : List (String × DefinitionKind) := [
  ("test.target", .target), ("test.kernel", .machine), ("test.provider", .provider),
  ("test.capability", .capability), ("test.state", .state), ("test.phase", .state),
  ("test.attempts", .state), ("test.request", .action), ("test.fault", .action),
  ("test.reply", .action), ("test.outcome", .outcome), ("test.fact", .fact)]
def definitions : List DefinitionMetadata := kinds.map fun (name, kind) =>
  Shared.Test.definitionMetadata name kind source (name ++ "/v1")
def provider : Provider (fun _ => True) := {
  id := id "test.provider"
  source
  contract := {
    id := id "test.capability"
    behaviorVersion := "correlated-structured-test/v1"
    requiredLaws := [] }
  meanings := (kinds.drop 4).map fun (name, kind) =>
    { definitionId := id name, kind, behaviorVersion := name ++ "/meaning-v1" }
  lawProofs := []
}

/-- Every state with the fields it holds, which is what the lowering attaches to each row. -/
def stateFields : List (ModelValue × List ModelValue) :=
  phases.flatMap fun phase => counts.map fun attempts =>
    (stateOf phase attempts, fieldsOf phase attempts)

def table : FiniteTable Unit ModelValue ModelValue ModelValue ModelValue := {
  setups := [⟨(), "default"⟩]
  states := phases.flatMap fun phase => counts.map fun attempts =>
    ⟨stateOf phase attempts, phase ++ "-" ++ toString attempts⟩
  actions := [⟨request, "request"⟩, ⟨fault, "fault"⟩, ⟨reply, "reply"⟩]
  outcomes := [⟨quiet, "quiet"⟩, ⟨response, "response"⟩]
  facts := [⟨pendingAttempts, "pendingAttempts"⟩]
  initial := [⟨(), [stateOf "open" 0]⟩]
  -- A request leaves an open operation where it is; a fault raises its attempt count, saturating
  -- at the bound; a reply finishes it at the count it reached. A finished operation takes no step.
  transitions := counts.flatMap fun attempts => [
    ⟨"open-" ++ toString attempts ++ "-request", stateOf "open" attempts, request,
      [step "open" attempts quiet]⟩,
    ⟨"open-" ++ toString attempts ++ "-fault", stateOf "open" attempts, fault,
      [step "open" (min (attempts + 1) attemptBound) quiet [pendingAttempts]]⟩,
    ⟨"open-" ++ toString attempts ++ "-reply", stateOf "open" attempts, reply,
      [step "done" attempts response]⟩]
}

def targetResult := table.checkTypedModel {
  id := id "test.target"
  source
  definitions
  requiredCapabilities := [id "test.capability"]
  metadata := { id := id "test.kernel", source }
} (Providers.empty.provide provider)

#guard targetResult.isOk

abbrev TestTarget := CheckedModel (fun _ => True) Unit ModelValue ModelValue ModelValue ModelValue
def context (target : TestTarget) := PropertyCheckContext.ofTarget target

/-- After a fault, the operation's `attempts` field is one: a claim about a field, read apart from
the state that carries it, on the trigger step itself. -/
def clause (ending : TraceEnding := .«partial») : PropertyCorrelatedClause := {
  id := id "test.correlated.attempt"
  source
  trigger := .atom { field := .selectedAction, reference := id "test.fault" }
  response := .atom {
    field := .resultingState
    reference := id "test.attempts"
    constraint := .equals (.text "1") }
  scope := [id "test.run"]
  key := id "test.operation"
  bound := 0
  ending
}

def declaration (ending : TraceEnding := .«partial») : Property := {
  id := id "test.property"
  source
  requires := [id "test.capability"]
  clauses := []
  correlatedRules := [clause ending]
}

def property (target : TestTarget) (ending : TraceEnding := .«partial») :=
  Property.check (context target) (declaration ending)

#guard (targetResult.toOption.map fun target => (property target).isOk) == some true

end Umpire.Property.CorrelatedStructuredTests
