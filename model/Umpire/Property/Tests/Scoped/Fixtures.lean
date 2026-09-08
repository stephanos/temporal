import Umpire.Property.Scoped
import Umpire.Observation.Evaluation.Scoped
import Umpire.Shared.Test

/-! Non-cancellation, multi-operation finite Target with labeled self-loops. -/

namespace Umpire.Property.ScopedTests

open Property.Scoped

def id := DefinitionId.of
def source : SourceLocation := { path := "Umpire/Property/Tests/Scoped.lean" }
def value (key payload : String) : ModelValue := ⟨id key, payload⟩
def state := value "test.state" "ready"
def request := value "test.trigger" "request"
def both := value "test.trigger" "both"
def tick := value "test.tick" "tick"
def reply := value "test.reply" "reply"
def quiet := value "test.outcome" "quiet"
def response := value "test.outcome" "response"
def fact := value "test.fact" "response"
def result (responded : Bool) : TransitionResult ModelValue ModelValue ModelValue := {
  resultingState := state
  modelOutcome := if responded then response else quiet
  observations := if responded then [fact, fact] else []
}

def kinds : List (String × DefinitionKind) := [
  ("test.target", .target), ("test.kernel", .kernel), ("test.provider", .provider),
  ("test.capability", .capability), ("test.state", .state), ("test.trigger", .action),
  ("test.tick", .action), ("test.reply", .action), ("test.outcome", .outcome), ("test.fact", .observation)]
def definitions : List DefinitionMetadata := kinds.map fun (name, kind) =>
  Shared.Test.definitionMetadata name kind source (name ++ "/v1")
def provider : CapabilityProvider (fun _ => True) := {
  id := id "test.provider"
  source
  contract := { id := id "test.capability", canonicalBehavior := "scoped-test/v1", requiredLaws := [] }
  meanings := (kinds.drop 4).map fun (name, kind) =>
    { definitionId := id name, kind, canonicalBehavior := name ++ "/meaning-v1" }
  lawWitnesses := []
}

def table : FiniteTable Unit ModelValue ModelValue ModelValue ModelValue := {
  setups := [⟨(), "default"⟩]
  states := [⟨state, "ready"⟩]
  actions := [⟨request, "request"⟩, ⟨both, "both"⟩, ⟨tick, "tick"⟩, ⟨reply, "reply"⟩]
  outcomes := [⟨quiet, "quiet"⟩, ⟨response, "response"⟩]
  facts := [⟨fact, "response"⟩]
  initial := [⟨(), [state]⟩]
  transitions := [
    ⟨"request", state, request, [result false]⟩,
    ⟨"both", state, both, [result true]⟩,
    ⟨"tick", state, tick, [result false]⟩,
    ⟨"reply", state, reply, [result true]⟩]
}

def targetResult := table.checkTarget {
  id := id "test.target"
  source
  definitions
  requiredCapabilities := [id "test.capability"]
  metadata := { id := id "test.kernel", source }
} (TargetComposition.empty.provide provider)

#guard targetResult.isOk

abbrev TestTarget := CheckedTarget (fun _ => True) Unit ModelValue ModelValue ModelValue ModelValue
def context (target : TestTarget) := PropertyCheckContext.ofTarget target

def clause (bound : Nat) (endpoint : PropertyScopedEndpoint := .runtimePrefix) : PropertyScopedClause := {
  id := id "test.scoped.response"
  source
  trigger := .atom { field := .selectedAction, reference := id "test.trigger" }
  response := .atom {
    field := .modelOutcome
    reference := id "test.outcome"
    constraint := .equals (.text "response") }
  scope := [id "test.run"]
  key := id "test.operation"
  clock := .operationTransitions
  bound
  endpoint
}

def declaration (bound : Nat) (endpoint : PropertyScopedEndpoint := .runtimePrefix) : PropertyDeclaration := {
  id := id "test.property"
  source
  requires := [id "test.capability"]
  clauses := []
  scopedClauses := [clause bound endpoint]
}

def property (target : TestTarget) (bound : Nat) (endpoint : PropertyScopedEndpoint := .runtimePrefix) :=
  checkProperty (context target) (.portable (declaration bound endpoint))

def limits : Limits := { transitions := 1000, obligations := 1000, work := 1000000 }
def scope : List (DefinitionId × String) := [(id "test.run", "run-1")]
def step (operation : String) (action : ModelValue) : Transition := {
  scope
  operationField := id "test.operation"
  operation
  priorState := state
  action
  result := result (action == reply || action == both)
}

def evaluate (bound : Nat) (endpoint : PropertyScopedEndpoint) (steps : List Transition)
    (budget : Limits := limits) : Except Error (List (DefinitionId × PropertyEndpointAnswer)) := do
  let .ok target := targetResult | throw .invalidInitialState
  let property ← (property target bound endpoint).mapError Error.property
  let compiled ← compile target property [id "test.run"] (id "test.operation") budget
  let initial ← compiled.start () state scope
  let run ← initial.consumeMany steps
  pure run.close.answers

end Umpire.Property.ScopedTests
