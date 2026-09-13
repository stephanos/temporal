import Umpire.Case.Correlated
import Umpire.Property.Tests.Correlated.Fixtures

/-! Independent small traces exercise actual checked lowering and typed portable observations. -/
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

def identity (ordinal : Nat) : CorrelatedIdentity := {
  scope := #[{ field_id := "test.run", value := some { value := some (.text_value "run-1") } }]
  evidence_source := "test.source", ordinal := Int64.ofInt ordinal }

def evidence (ordinal : Nat) (kind : String) (operation := "a") (parents : List Nat := []) : CorrelatedEvidence := {
  identity := some (identity ordinal)
  operation, kind := "test." ++ kind
  parents := parents.toArray.map identity }

structure Scenario where
  name : String
  bound : Nat
  events : List CorrelatedEvidence
  expected : Nat
  endpoint : TraceEnding := .«partial»
  incomplete : Bool := false

def scenarios : List Scenario := [
  ⟨"zero-response", 0, [evidence 0 "both"], 2, .«partial», false⟩,
  ⟨"zero-missing", 0, [evidence 0 "request"], 3, .«partial», false⟩,
  ⟨"trigger", 2, [evidence 0 "request"], 0, .«partial», false⟩,
  ⟨"deadline", 1, [evidence 0 "request", evidence 1 "reply"], 2, .«partial», false⟩,
  ⟨"late", 1, [evidence 0 "request", evidence 1 "tick", evidence 2 "reply"], 3, .«partial», false⟩,
  ⟨"multiple", 2, [evidence 0 "request", evidence 1 "request", evidence 2 "reply"], 2, .«partial», false⟩,
  ⟨"interleaved", 1, [evidence 0 "request", evidence 1 "tick" "b", evidence 2 "reply"], 2, .«partial», false⟩,
  ⟨"self-loop", 1, [evidence 0 "request", evidence 1 "tick"], 3, .«partial», false⟩,
  ⟨"closed", 3, [evidence 0 "request"], 3, .final, false⟩,
  ⟨"incomplete-close", 3, [evidence 0 "request"], 0, .final, true⟩,
  ⟨"pending-cause", 1, [evidence 1 "reply" "a" [0]], 0, .final, false⟩,
  ⟨"causal-chunks", 1, [evidence 1 "reply" "a" [0], evidence 0 "request", evidence 1 "reply" "a" [0]], 2, .«partial», false⟩,
  ⟨"poll-stutter", 1, [evidence 0 "request", evidence 1 "poll", evidence 2 "reply"], 2, .«partial», false⟩,
  ⟨"violation-incomplete", 0, [evidence 0 "request"], 3, .final, true⟩,
  -- Silence is not vacuous truth: a capability that admitted no evidence observed nothing, so it
  -- reports unresolved rather than the empty-obligation satisfaction the total model trace has.
  ⟨"unobserved", 1, [], 0, .«partial», false⟩]

/-- Lower one scenario's Case beside the correlated ceilings its capability is decoded under, which
the Case does not carry. -/
def loweredCase (scenario : Scenario) :
    Except String (temporal.server.api.testpilot.v1.Case × CorrelatedLimits) := do
  let target ← targetResult.mapError (fun _ => "target")
  let property ← (property target scenario.bound scenario.endpoint).mapError (fun _ => "property")
  let plan ← (plan target).mapError (fun _ => "projection")
  let compiled ← (Case.Projection.Correlated.compile plan property
    { transitions := 32, obligations := 16, work := 1000000000 }).mapError (fun _ => "correlated compile")
  let lowered ← (Correlated.lower plan compiled "evidence").mapError (fun error => error.construct)
  let binding : Provenance.DefinitionBinding :=
    ⟨property.id.value, property.behaviorFingerprint.render, .property⟩
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
    properties := [lowered.contractLowering]
  }).mapError (·.construct)
  pure (artifact, lowered.limits)

def compiledCase (scenario : Scenario) : Except String temporal.server.api.testpilot.v1.Case :=
  (·.1) <$> loweredCase scenario

/-- Exercise the same checked Contract through ordinary RPC response projection and Run recording.
The test Driver supplies the declared typed evidence; this is qualification, not an Implementation Link. -/
def runnableCase (scenario : Scenario) : Except String temporal.server.api.testpilot.v1.Case := do
  let artifact ← compiledCase scenario
  let some program := artifact.program | throw "missing program"
  let nodes := scenario.events.toArray.mapIdx fun index _ =>
    Testpilot.Authoring.Program.node ("read." ++ toString index)
      (Testpilot.Authoring.Program.invokeRpc "source" "/test.correlated.Source/Read" #[]
        #[Testpilot.Authoring.Program.responseRead (Testpilot.Authoring.Path.make #[])
          .READ_CARDINALITY_ONE #[Testpilot.Authoring.Program.observationTarget "evidence"]])
      (Testpilot.Authoring.Program.instructionLimits 1000 1)
      (guard := if index == 0 then none else
        some (Testpilot.Authoring.Expr.literal (Testpilot.Authoring.Value.boolean true)))
      (outcome := some (Testpilot.Authoring.Program.outcome #[
        Testpilot.Authoring.Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS
          (Testpilot.Authoring.Types.singular (Testpilot.Authoring.Types.enumeration
            "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"))]))
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
  let initial ← compiled.start scope
  let run ← scenario.events.zipIdx.foldlM (fun run (event, index) => run.observe (index + 2) event) initial
  pure (run.close.answers scenario.incomplete)

#guard scenarios.all (fun scenario => (evaluate scenario).toOption == some [scenario.expected])

end Umpire.Case.CorrelatedFixtures
