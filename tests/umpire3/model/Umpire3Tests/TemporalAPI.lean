import Temporal.API.Nexus

namespace Umpire3.Temporal.API.Nexus.Tests

def validRequest : Generated.RequestCancelNexusOperationExecutionRequest where
  namespaceName := "namespace"
  operationId := "operation"
  runId := "run"
  identity := "client"
  requestId := "request"
  reason := "test"

example : interpretCancel validRequest = .ok {
    namespaceName := "namespace"
    operationId := "operation"
    runId := "run"
    requestId := "request"
    reason := "test"
  } := by rfl

example : interpretCancel { validRequest with namespaceName := "" } = .error .missingNamespace := by
  rfl

example : semanticEffect { validRequest with requestId := "" } = none := by rfl

example {request command} (accepted : interpretCancel request = .ok command) : command.Valid :=
  accepted_is_valid accepted

example {request failure} (rejected : interpretCancel request = .error failure) :
    semanticEffect request = none := rejected_has_no_effect rejected

end Umpire3.Temporal.API.Nexus.Tests
