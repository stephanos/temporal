import Temporal.Testpilot
import Temporal.Feature.Nexus.Caller.Model
import Temporal.Feature.Nexus.Control.Model
import Temporal.Feature.Nexus.Pair.Model
import Temporal.Feature.System.Info.Model
import Temporal.Feature.Workflow.Outage.Model
import Temporal.Feature.Workflow.Start.Model
import Testpilot.Examples.Synthetic
import Testpilot.ProtoJSON

/-!
Render the checked-in Cases. Every Case a `case` block declares is enumerated from the Lean
environment rather than from a table maintained here: `--list` prints what exists,
`--render <case-id>` prints its canonical bytes.

The synthetic and conformance Cases stay reachable by their own argument. They carry no Model and
the Go conformance builder names them by expected Verdict, which the registry does not model.
-/

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
  | ["conformance-static-preparation-rejection-expression-context"] =>
      renderTestpilot Temporal.Testpilot.conformanceExpressionContextRejectionCase
  | ["conformance-static-preparation-rejection-command-type"] =>
      renderTestpilot Temporal.Testpilot.conformanceCommandTypeRejectionCase
  | ["conformance-static-preparation-rejection-invalid-duration"] =>
      renderTestpilot Temporal.Testpilot.conformanceInvalidDurationRejectionCase
  | ["conformance-static-preparation-rejection-unsettable-field"] =>
      renderTestpilot Temporal.Testpilot.conformanceUnsettableFieldRejectionCase
  | ["conformance-static-preparation-rejection-reply-not-admitted"] =>
      renderTestpilot Temporal.Testpilot.conformanceReplyRejectionCase
  | ["conformance-static-preparation-rejection-undeclared-evidence"] =>
      renderTestpilot Temporal.Testpilot.conformanceUndeclaredEvidenceRejectionCase
  | ["conformance-static-preparation-rejection-duplicate-evidence"] =>
      renderTestpilot Temporal.Testpilot.conformanceDuplicateEvidenceRejectionCase
  | ["conformance-satisfied-history-evidence"] =>
      renderTestpilot Temporal.Testpilot.conformanceHistoryEvidenceCase
  | ["conformance-satisfied-run-event-evidence"] =>
      renderTestpilot Temporal.Testpilot.conformanceRunEventEvidenceCase
  | ["conformance-satisfied-read-evidence"] =>
      renderTestpilot Temporal.Testpilot.conformanceReadEvidenceCase
  | ["conformance-cleanup-failure-after-proved-violation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCleanupFailureCase
  | ["conformance-cross-run-isolation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCrossRunIsolationCase
  | [fixture] => renderRegisteredFixture fixture
  | _ => throw (IO.userError "expected --list, --render <case-id>, or a Case fixture name")
