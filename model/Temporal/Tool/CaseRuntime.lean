import Temporal.CaseRuntime
import Umpire.Case.ProtoJSON

private def render (compiled : Except Umpire.Case.Compiler.LoweringError Umpire.Case) : IO Unit :=
  match compiled with
  | .ok output => IO.print (Umpire.Case.ProtoJSON.canonical output)
  | .error failure => throw (IO.userError (reprStr failure))

def main (arguments : List String) : IO Unit :=
  match arguments with
  | ["get-system-info"] => render Temporal.CaseRuntime.getSystemInfoCase
  | ["async-nexus"] => render Temporal.CaseRuntime.asyncNexusCase
  | ["conformance-satisfied"] => render Temporal.CaseRuntime.conformanceSatisfiedCase
  | ["conformance-violated"] => render Temporal.CaseRuntime.conformanceViolatedCase
  | ["conformance-inconclusive"] => render Temporal.CaseRuntime.conformanceInconclusiveCase
  | ["conformance-static-preparation-rejection"] =>
      render Temporal.CaseRuntime.conformanceStaticRejectionCase
  | ["conformance-cleanup-failure-after-proved-violation"] =>
      render Temporal.CaseRuntime.conformanceCleanupFailureCase
  | ["conformance-cross-run-isolation"] =>
      render Temporal.CaseRuntime.conformanceCrossRunIsolationCase
  | _ => throw (IO.userError "expected a supported Case fixture name")
