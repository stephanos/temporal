import Temporal.Testpilot
import Temporal.Feature.Nexus3.Testpilot
import Testpilot.ProtoJSON

private def renderTestpilot
    (compiled : Except Umpire.Case.Compiler.LoweringError
      temporal.server.api.testpilot.v1.Case) : IO Unit :=
  match compiled with
  | .ok output => do
      match ← Testpilot.ProtoJSON.canonical output with
      | .ok encoded => IO.print encoded
      | .error failure => throw (IO.userError (toString failure))
  | .error failure => throw (IO.userError (reprStr failure))

def main (arguments : List String) : IO Unit :=
  match arguments with
  | ["get-system-info"] => renderTestpilot Temporal.Testpilot.getSystemInfoCase
  | ["async-nexus"] => renderTestpilot Temporal.Feature.Nexus3.Testpilot.completionCase
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
