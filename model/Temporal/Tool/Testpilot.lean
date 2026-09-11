import Temporal.Testpilot
import Temporal.Feature.Nexus.Success.Model
import Temporal.Feature.Nexus.Success.TypedUnary
import Temporal.Feature.Nexus.Success.TypedNexus
import Testpilot.Examples.Synthetic
import Testpilot.ProtoJSON

/-!
Render the checked-in Cases. Every Case a `case` block declares, and every Case that registers its
value explicitly, is enumerated from the Lean environment rather than from a table maintained here:
`--list` prints what exists, `--render <case-id>` prints its canonical bytes.

The synthetic and conformance Cases stay reachable by their own argument. They carry no Model and
the Go conformance builder names them by expected Verdict, which the registry does not model.
-/

/-! The two typed examples and the two realization-only Cases carry their identities in Lean rather
than in a Model, so they register their existing values. Their Programs, Profiles and Contracts do
not change. -/
register_case Temporal.Testpilot.getSystemInfoCase
  id "temporal.case.get-system-info" fixture "get-system-info"
register_case Temporal.Testpilot.workerOutageCase
  id "temporal.case.worker-outage" fixture "worker-outage"
register_case Temporal.Feature.Nexus.Success.TypedUnary.typedUnaryCase
  id "temporal.case.typed-unary" fixture "typed-unary"
register_case Temporal.Feature.Nexus.Success.TypedNexus.typedNexusCase
  id "temporal.case.typed-nexus" fixture "typed-nexus"

/-- Every registered Case, sorted by Case ID. -/
def registered : List Temporal.Case.Registry.Materialized := registeredCases%

private def renderTestpilot
    (compiled : Except Umpire.Case.Compiler.Error
      temporal.server.api.testpilot.v1.Case) : IO Unit :=
  match compiled with
  | .ok output => do
      match ← Testpilot.ProtoJSON.canonical output with
      | .ok encoded => IO.print encoded
      | .error failure => throw (IO.userError (toString failure))
  | .error failure => throw (IO.userError (reprStr failure))

private def renderSynthetic : IO Unit := do
  match ← Testpilot.Examples.Synthetic.canonical with
  | .ok encoded => IO.print encoded
  | .error failure => throw (IO.userError (toString failure))

private def knownCaseIds : String :=
  ", ".intercalate (registered.map (·.caseId))

private def knownFixtures : String :=
  ", ".intercalate (registered.map (·.fixture))

private def renderRegisteredCase (caseId : String) : IO Unit :=
  match registered.find? (·.caseId == caseId) with
  | some entry => renderTestpilot entry.value
  | none => throw (IO.userError s!"unknown Case '{caseId}'; known: {knownCaseIds}")

private def renderRegisteredFixture (fixture : String) : IO Unit :=
  match registered.find? (·.fixture == fixture) with
  | some entry => renderTestpilot entry.value
  | none => throw (IO.userError s!"unknown Case fixture '{fixture}'; known: {knownFixtures}")

def main (arguments : List String) : IO Unit :=
  match arguments with
  | ["--list"] => registered.forM fun entry => IO.println s!"{entry.caseId} {entry.fixture}"
  | ["--render", caseId] => renderRegisteredCase caseId
  | ["synthetic"] => renderSynthetic
  | ["conformance-satisfied"] => renderTestpilot Temporal.Testpilot.conformanceSatisfiedCase
  | ["conformance-violated"] => renderTestpilot Temporal.Testpilot.conformanceViolatedCase
  | ["conformance-inconclusive"] => renderTestpilot Temporal.Testpilot.conformanceInconclusiveCase
  | ["conformance-static-preparation-rejection"] =>
      renderTestpilot Temporal.Testpilot.conformanceStaticRejectionCase
  | ["conformance-cleanup-failure-after-proved-violation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCleanupFailureCase
  | ["conformance-cross-run-isolation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCrossRunIsolationCase
  | [fixture] => renderRegisteredFixture fixture
  | _ => throw (IO.userError "expected --list, --render <case-id>, or a Case fixture name")
