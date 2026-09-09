import Temporal.Feature.Nexus3.TypedNexus

namespace Temporal.Feature.Nexus3.Tests.TypedNexus
open Temporal.Feature.Nexus3.TypedNexus

#guard historyBinding.isOk
#guard (scheduleCommand firstCommandId).isOk
#guard scheduledEventDeclaration.isOk

#eval match checked with
  | .ok _ => "ok"
  | .error (.operation error) => "operation: " ++ reprStr error
  | .error (.field error) => "field: " ++ reprStr error
  | .error (.target _) => "target"
  | .error (.property error) => "property: " ++ reprStr error
  | .error (.scoped error) => "scoped: " ++ reprStr error
  | .error (.inconsistent reason) => "inconsistent: " ++ reason

#eval match typedNexusCase with
  | .ok output => "ok: " ++ output.case_id
  | .error failure => "error: " ++ failure.construct ++ " / " ++ failure.sourceDefinitionId

end Temporal.Feature.Nexus3.Tests.TypedNexus
