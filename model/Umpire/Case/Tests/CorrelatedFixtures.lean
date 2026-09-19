import Umpire.Case.Correlated
import Umpire.Property.Tests.Correlated.Fixtures
import Umpire.Property.Tests.Correlated.Structured

/-! Independent small traces exercise actual checked lowering and typed portable observations.

Two Targets carry them. The atomic one has a single state and pins the obligation semantics; the
structured one keeps a phase and an attempt count per operation, so its rows carry state fields on
the wire and its rule reads one, and several operations under their own keys pin that each is
tracked apart. Both evaluators -- this module's `#guard` and the Go facade test that replays the
generated fixture -- answer every scenario the same way. -/
namespace Umpire.Case.CorrelatedFixtures
open Umpire.Property.CorrelatedTests
open temporal.server.api.testpilot.v1

def plan (target : TestTarget) := Case.Projection.check target {
  id := id "test.projection"
  scopeFields := [id "test.run"]
  operationField := id "test.operation"
  sources := [id "test.source"]
  rules := [
    { kind := id "test.request", meaning := .confirmed none [(request, result false)] },
    { kind := id "test.both", meaning := .confirmed none [(both, result true)] },
    { kind := id "test.reply", meaning := .confirmed none [(reply, result true)] },
    { kind := id "test.tick", meaning := .confirmed none [(tick, result false)] },
    { kind := id "test.poll", meaning := .irrelevant }]
  limits := {
    events := 16, buffered := 8, keys := 8, support := 256
    work := 1000000000, eventSize := 512 }
} () state

/-! The evidence a test Driver supplies names the scope field, source and kinds by the Case-local names
the compiled Case gives `test.run`, `test.source` and each `test.<kind>`. -/

/-- The Run scope under the Case-local name of `test.run`. -/
def caseScope : List (Shared.SemanticData.Name × String) := [(⟨"run"⟩, "run-1")]

def identity (ordinal : Nat) : CorrelatedIdentity := {
  scope := #[{ field_id := "run", value := some { value := some (.text_value "run-1") } }]
  evidence_source := "source", ordinal := Int64.ofInt ordinal }

def evidence (ordinal : Nat) (kind : String) (operation := "a") (parents : List Nat := []) : CorrelatedEvidence := {
  identity := some (identity ordinal)
  operation, kind
  parents := parents.toArray.map identity }

/-- Which Target a scenario runs over. -/
inductive Family where
  | atomic
  | structured
  deriving BEq, Repr

structure Scenario where
  name : String
  bound : Nat
  events : List CorrelatedEvidence
  expected : Nat
  endpoint : TraceEnding := .«partial»
  incomplete : Bool := false
  family : Family := .atomic

def scenarios : List Scenario := [
  ⟨"zero-response", 0, [evidence 0 "both"], 2, .«partial», false, .atomic⟩,
  ⟨"zero-missing", 0, [evidence 0 "request"], 3, .«partial», false, .atomic⟩,
  ⟨"trigger", 2, [evidence 0 "request"], 0, .«partial», false, .atomic⟩,
  ⟨"deadline", 1, [evidence 0 "request", evidence 1 "reply"], 2, .«partial», false, .atomic⟩,
  ⟨"late", 1, [evidence 0 "request", evidence 1 "tick", evidence 2 "reply"], 3, .«partial», false, .atomic⟩,
  ⟨"multiple", 2, [evidence 0 "request", evidence 1 "request", evidence 2 "reply"], 2, .«partial», false, .atomic⟩,
  ⟨"interleaved", 1, [evidence 0 "request", evidence 1 "tick" "b", evidence 2 "reply"], 2, .«partial», false, .atomic⟩,
  ⟨"self-loop", 1, [evidence 0 "request", evidence 1 "tick"], 3, .«partial», false, .atomic⟩,
  ⟨"closed", 3, [evidence 0 "request"], 3, .final, false, .atomic⟩,
  ⟨"incomplete-close", 3, [evidence 0 "request"], 0, .final, true, .atomic⟩,
  ⟨"pending-cause", 1, [evidence 1 "reply" "a" [0]], 0, .final, false, .atomic⟩,
  ⟨"causal-chunks", 1, [evidence 1 "reply" "a" [0], evidence 0 "request", evidence 1 "reply" "a" [0]], 2, .«partial», false, .atomic⟩,
  ⟨"poll-stutter", 1, [evidence 0 "request", evidence 1 "poll", evidence 2 "reply"], 2, .«partial», false, .atomic⟩,
  ⟨"violation-incomplete", 0, [evidence 0 "request"], 3, .final, true, .atomic⟩,
  -- Silence is not vacuous truth: a capability that admitted no evidence observed nothing, so it
  -- reports unresolved rather than the empty-obligation satisfaction the total model trace has.
  ⟨"unobserved", 1, [], 0, .«partial», false, .atomic⟩,
  -- The structured Target: after a fault the operation's `attempts` field is one, read apart from
  -- the state that carries it. A first fault satisfies it and a second (`faultAgain`, the recorder
  -- saying which fault this was) violates it; two operations that each fault once are two
  -- operations, not one that faulted twice.
  ⟨"structured-first-fault", 0, [evidence 0 "request", evidence 1 "fault"], 2, .«partial», false,
    .structured⟩,
  ⟨"structured-second-fault", 0,
    [evidence 0 "request", evidence 1 "fault", evidence 2 "faultAgain"], 3, .«partial», false,
    .structured⟩,
  ⟨"structured-two-operations", 0,
    [evidence 0 "request", evidence 1 "request" "b", evidence 2 "fault", evidence 3 "fault" "b"], 2,
    .«partial», false, .structured⟩,
  ⟨"structured-interleaved-violation", 0,
    [evidence 0 "request", evidence 1 "request" "b", evidence 2 "fault", evidence 3 "fault" "b",
      evidence 4 "reply" "b", evidence 5 "faultAgain"], 3, .«partial», false, .structured⟩]

/-! ### The structured Target's projection

The same shape as the atomic one, with each evidence kind confirming the steps its action can take
from any open count, and the state fields attached at the lowering. -/

namespace Structured

private def sid := Umpire.Property.CorrelatedStructuredTests.id
private def stable := Umpire.Property.CorrelatedStructuredTests.table

/-- The one step an evidence kind confirms. A kind's outputs are the steps the evidence confirms in
sequence, each continuing from the operation's own state, so a kind names one result: the recorder
says which fault this was, and the projector checks that the operation is where that step starts. -/
private def confirms (action : ModelValue) (phase : String) (attempts : Nat)
    (outcome : ModelValue) (facts : List ModelValue := []) :
    Shared.CorrelatedProjection.Meaning ModelValue (Umpire.Step ModelValue ModelValue ModelValue) :=
  .confirmed none [(action, Umpire.Property.CorrelatedStructuredTests.step phase attempts outcome facts)]

def plan (target : Umpire.Property.CorrelatedStructuredTests.TestTarget) :=
  Case.Projection.check target {
    id := sid "test.projection"
    scopeFields := [sid "test.run"]
    operationField := sid "test.operation"
    sources := [sid "test.source"]
    rules := [
      { kind := sid "test.request"
        meaning := confirms Umpire.Property.CorrelatedStructuredTests.request "open" 0
          Umpire.Property.CorrelatedStructuredTests.quiet },
      { kind := sid "test.fault"
        meaning := confirms Umpire.Property.CorrelatedStructuredTests.fault "open" 1
          Umpire.Property.CorrelatedStructuredTests.quiet
          [Umpire.Property.CorrelatedStructuredTests.pendingAttempts] },
      { kind := sid "test.faultAgain"
        meaning := confirms Umpire.Property.CorrelatedStructuredTests.fault "open" 2
          Umpire.Property.CorrelatedStructuredTests.quiet
          [Umpire.Property.CorrelatedStructuredTests.pendingAttempts] },
      { kind := sid "test.reply"
        meaning := confirms Umpire.Property.CorrelatedStructuredTests.reply "done" 1
          Umpire.Property.CorrelatedStructuredTests.response }]
    limits := {
      events := 16, buffered := 8, keys := 8, support := 256
      work := 1000000000, eventSize := 512 }
  } () (Umpire.Property.CorrelatedStructuredTests.stateOf "open" 0)

end Structured

/-- Lower one scenario's Case beside the correlated ceilings its capability is decoded under, which
the Case does not carry. -/
def loweredCase (scenario : Scenario) :
    Except String (temporal.server.api.testpilot.v1.Case × CorrelatedLimits) := do
  let (lowering, binding, source) ← match scenario.family with
    | .atomic => do
        let target ← targetResult.mapError (fun _ => "target")
        let property ← (property target scenario.bound scenario.endpoint).mapError
          (fun _ => "property")
        let plan ← (plan target).mapError (fun _ => "projection")
        let compiled ← (Case.Projection.Correlated.compile plan property
          { transitions := 32, obligations := 16, work := 1000000000 }).mapError
          (fun _ => "correlated compile")
        let lowered ← (Correlated.lower plan compiled "evidence").mapError
          (fun error => error.construct)
        pure ((lowered.contractLowering, lowered.limits),
          (⟨property.id.value, property.behaviorFingerprint.render, .property⟩ :
            Provenance.DefinitionBinding), source)
    | .structured => do
        let target ← Umpire.Property.CorrelatedStructuredTests.targetResult.mapError
          (fun _ => "target")
        let property ← (Umpire.Property.CorrelatedStructuredTests.property target
          scenario.endpoint).mapError (fun _ => "property")
        let plan ← (Structured.plan target).mapError (fun _ => "projection")
        let compiled ← (Case.Projection.Correlated.compile plan property
          { transitions := 32, obligations := 16, work := 1000000000 }).mapError
          (fun _ => "correlated compile")
        let lowered ← (Correlated.lower plan compiled "evidence"
          (stateFields := Umpire.Property.CorrelatedStructuredTests.stateFields)).mapError
          (fun error => error.construct)
        pure ((lowered.contractLowering, lowered.limits),
          (⟨property.id.value, property.behaviorFingerprint.render, .property⟩ :
            Provenance.DefinitionBinding),
          Umpire.Property.CorrelatedStructuredTests.source)
  let program := Testpilot.Authoring.Program.make "correlated.program" #[] #[]
    #[Testpilot.Authoring.Program.observation "evidence" (Testpilot.Authoring.Types.singular
      (Testpilot.Authoring.Types.messageType "temporal.server.api.testpilot.v1.CorrelatedEvidence"))]
    #[Testpilot.Authoring.Program.controller "controller" #[]]
    (Testpilot.Authoring.Program.cleanup "cleanup" #[])
  let artifact ← (Compiler.compile {
    version := { major := 1 }
    caseId := "correlated." ++ scenario.name
    producerId := "umpire.correlated.fixtures"
    definitions := [binding]
    sources := [source]
    knownGaps := []
    program
    contractId := "correlated"
    properties := [lowering.1]
  }).mapError (·.construct)
  pure (artifact, lowering.2)

def compiledCase (scenario : Scenario) : Except String temporal.server.api.testpilot.v1.Case :=
  (·.1) <$> loweredCase scenario

/-- Exercise the same checked Contract through ordinary RPC response reads and Run recording.
The test Driver supplies the declared typed evidence; this is qualification, not an Implementation Link. -/
def runnableCase (scenario : Scenario) : Except String temporal.server.api.testpilot.v1.Case := do
  let artifact ← compiledCase scenario
  let some program := artifact.program | throw "missing program"
  let nodes := scenario.events.toArray.mapIdx fun index _ =>
    Testpilot.Authoring.Program.node ("read." ++ toString index)
      (Testpilot.Authoring.Program.invokeRpc "source" "/test.correlated.Source/Read" #[]
        #[Testpilot.Authoring.Program.responseRead (Testpilot.Authoring.Path.make #[])
          .READ_CARDINALITY_ONE #[Testpilot.Authoring.Program.observationTarget "evidence"]])
      (guard := if index == 0 then none else
        some (Testpilot.Authoring.Expr.literal (Testpilot.Authoring.Value.boolean true)))
  pure { artifact with program := some { program with
    roles := #[Testpilot.Authoring.Program.role "source" .ROLE_KIND_ENDPOINT]
    entrypoints := #[Testpilot.Authoring.Program.controller "controller" nodes] } }

def capability (scenario : Scenario) : Except String CorrelatedContract := do
  let artifact ← compiledCase scenario
  let some contract := artifact.contract | throw "missing contract"
  let some capability := contract.«correlated» | throw "missing capability"
  pure capability

/-- The correlated ceilings the scenario's capability is decoded under. -/
def ceilings (scenario : Scenario) : Except String CorrelatedLimits := (·.2) <$> loweredCase scenario

def evaluate (scenario : Scenario) : Except String (List Nat) := do
  let compiled ← Testpilot.Correlated.decode (← ceilings scenario) (← capability scenario)
  let initial ← compiled.start caseScope
  let run ← scenario.events.zipIdx.foldlM (fun run (event, index) => run.observe (index + 2) event) initial
  pure (run.close.answers scenario.incomplete)

#guard scenarios.all (fun scenario => (evaluate scenario).toOption == some [scenario.expected])

end Umpire.Case.CorrelatedFixtures
