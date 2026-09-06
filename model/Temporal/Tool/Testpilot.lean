import Temporal.Testpilot
import Temporal.Testpilot.TestpilotProtoJSON

private def renderTestpilot
    (compiled : Except Umpire.Case.Compiler.LoweringError Umpire.Case) : IO Unit :=
  match compiled with
  | .ok output =>
      match Temporal.Testpilot.TestpilotProtoJSON.canonical output with
      | .ok encoded => IO.print encoded
      | .error failure => throw (IO.userError failure)
  | .error failure => throw (IO.userError (reprStr failure))

def main (arguments : List String) : IO Unit :=
  match arguments with
  | ["get-system-info"] => renderTestpilot Temporal.Testpilot.getSystemInfoCase
  | ["async-nexus"] => renderTestpilot Temporal.Testpilot.asyncNexusCase
  | ["conformance-satisfied"] => renderTestpilot Temporal.Testpilot.conformanceSatisfiedCase
  | ["conformance-violated"] => renderTestpilot Temporal.Testpilot.conformanceViolatedCase
  | ["conformance-inconclusive"] => renderTestpilot Temporal.Testpilot.conformanceInconclusiveCase
  | ["conformance-static-preparation-rejection"] =>
      renderTestpilot Temporal.Testpilot.conformanceStaticRejectionCase
  | ["conformance-cleanup-failure-after-proved-violation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCleanupFailureCase
  | ["conformance-cross-run-isolation"] =>
      renderTestpilot Temporal.Testpilot.conformanceCrossRunIsolationCase
  | _ => throw (IO.userError "expected a supported Case fixture name")
