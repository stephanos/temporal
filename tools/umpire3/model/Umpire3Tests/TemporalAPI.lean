import Temporal.API.Nexus

namespace Umpire3.Temporal.API.Nexus.Tests

def validRequest : Generated.RequestCancelNexusOperationExecutionRequest where
  namespaceName := "namespace"
  operationId := "operation"
  runId := "run"
  identity := "client"
  requestId := "request"
  reason := "test"

def validStartRequest : Generated.StartNexusOperationExecutionRequest where
  namespaceName := "namespace"
  identity := "client"
  requestId := "request"
  operationId := "operation"
  endpoint := "endpoint"
  service := "service"
  operation := "operation"
  scheduleToCloseTimeout := some { seconds := 10, nanos := 0 }
  scheduleToStartTimeout := none
  startToCloseTimeout := some { seconds := 1, nanos := 0 }
  input := none
  idReusePolicy := { number := 0 }
  idConflictPolicy := { number := 0 }
  searchAttributes := none
  nexusHeader := []
  userMetadata := none

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

example : durationValid { seconds := 0, nanos := 0 } = true := by rfl

example : durationValid { seconds := 0, nanos := -1 } = false := by rfl

example : interpretStart validStartRequest = .accepted {
    namespaceName := ⟨"namespace", by decide⟩
    operationId := ⟨"operation", by decide⟩
    endpoint := ⟨"endpoint", by decide⟩
    service := ⟨"service", by decide⟩
    operation := ⟨"operation", by decide⟩
    requestId := ⟨"request", by decide⟩
    scheduleToCloseTimeout := some { seconds := 10, nanos := 0 }
    scheduleToStartTimeout := none
    startToCloseTimeout := some { seconds := 1, nanos := 0 }
    hasSensitiveInput := false
  } := by rfl

example : interpretStart { validStartRequest with endpoint := "" } =
    .rejected .missingEndpoint := by rfl

example : interpretStart {
    validStartRequest with startToCloseTimeout := some { seconds := 0, nanos := -1 }
  } = .rejected .invalidStartToCloseTimeout := by rfl

end Umpire3.Temporal.API.Nexus.Tests
