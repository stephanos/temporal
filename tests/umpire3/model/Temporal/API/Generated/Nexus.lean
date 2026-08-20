namespace Umpire3.Temporal.API.Generated

def descriptorFullName : String := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest"
def descriptorHash : String := "sha256:00725f8eb481d0cd2597c8763977a928c8b27a21f8d3cd99d6e416eccdf5da04"

structure RequestCancelNexusOperationExecutionRequest where
  namespaceName : String
  operationId : String
  runId : String
  identity : String
  requestId : String
  reason : String
  deriving DecidableEq, Repr

end Umpire3.Temporal.API.Generated
