import Temporal.CaseRuntime
import Temporal.CaseRuntime.TestpilotProtoJSON

private def renderTestpilot
    (compiled : Except Umpire.Case.Compiler.LoweringError Umpire.Case) : IO Unit :=
  match compiled with
  | .ok output =>
      match Temporal.CaseRuntime.TestpilotProtoJSON.canonical output with
      | .ok encoded => IO.print encoded
      | .error failure => throw (IO.userError failure)
  | .error failure => throw (IO.userError (reprStr failure))

def main (arguments : List String) : IO Unit :=
  match arguments with
  | ["get-system-info"] => renderTestpilot Temporal.CaseRuntime.getSystemInfoCase
  | ["async-nexus"] => renderTestpilot Temporal.CaseRuntime.asyncNexusCase
  | ["conformance-satisfied"] => renderTestpilot Temporal.CaseRuntime.conformanceSatisfiedCase
  | ["conformance-violated"] => renderTestpilot Temporal.CaseRuntime.conformanceViolatedCase
  | ["conformance-inconclusive"] => renderTestpilot Temporal.CaseRuntime.conformanceInconclusiveCase
  | ["conformance-static-preparation-rejection"] =>
      renderTestpilot Temporal.CaseRuntime.conformanceStaticRejectionCase
  | ["conformance-cleanup-failure-after-proved-violation"] =>
      renderTestpilot Temporal.CaseRuntime.conformanceCleanupFailureCase
  | ["conformance-cross-run-isolation"] =>
      renderTestpilot Temporal.CaseRuntime.conformanceCrossRunIsolationCase
  | _ => throw (IO.userError "expected a supported Case fixture name")
