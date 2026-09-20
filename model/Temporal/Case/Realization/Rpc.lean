import Temporal.Case.Evidence
import Temporal.Case.Support

/-!
# The unary-call realization

A controller invokes one workflow-service method and reads a field of its response into an
observation; no workflow runs. It is the realization of a Model whose one side effect is the call:
the `getSystemInfo` action class is bound to the `GetSystemInfo` invocation. What confirms the
step is the Run Event the runtime records when the instruction completes, keyed by its protocol
code -- a Run makes the one call, so the one call is the operation -- because a unary call leaves
no history to read back.
-/

namespace Temporal.Case.Realization.Rpc

open Umpire
open Testpilot.Authoring
open Temporal.Case.Support
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue

def projectionId : DefinitionId := .of "temporal.system.info.projection"
def evidenceSourceId : DefinitionId := .of "temporal.system.info.source.run"
def runFieldId : DefinitionId := .of "temporal.system.info.scope.run"
def callFieldId : DefinitionId := .of "temporal.system.info.scope.call"
def completedEvidenceKindId : DefinitionId :=
  .of "temporal.system.info.evidence.instructionCompleted"

/-- The action class the realization binds, stated as the fallback a binding is read by where the
Model's vocabulary has no member of the binding's key. -/
def getSystemInfoAction : DefinitionId := .of "temporal.system.info.action.getSystemInfo"

/-- The observation the call's `server_version` is read into. -/
def serverVersionObservation := "server-version"

/-- The instruction's completion, as the runtime records it: the one Run Event kind a unary call
leaves behind, keyed by the protocol code the outcome carries. -/
def completedSource : Umpire.Case.Producer.EvidenceSource := {
  eventKind := "instructionCompleted"
  recorded := .runEvent .RUN_EVENT_KIND_INSTRUCTION_COMPLETED
  operationKeyPath := field "protocol_code"
  kindId := completedEvidenceKindId
  sourceId := evidenceSourceId }

/-- The call: the one action class, reading the server version out of its response. -/
def getSystemInfoBinding : Umpire.Case.Producer.ActionBinding := {
  action := getSystemInfoAction
  key := "getSystemInfo"
  instructionId := "get-system-info"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.invokeRpc workflowServiceRole getSystemInfoMethod #[]
        #[project (field "server_version") serverVersionObservation])
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

def plan : Umpire.Case.Producer.ProgramPlan := {
  roles := #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT]
  observations := #[
    Program.observation serverVersionObservation textType,
    Program.observation correlatedObservation correlatedEvidenceType]
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := [.actions [getSystemInfoAction]] }]
  cleanup := Program.cleanup "cleanup" #[] }

end Temporal.Case.Realization.Rpc

namespace Temporal.Case.Realization

open Umpire

/-- One unary workflow-service call made by a controller, realized from a Model's `getSystemInfo`
class; no workflow entrypoint. -/
def unaryRpc : Umpire.Case.Producer.Realization := {
  plan := Rpc.plan
  actions := [Rpc.getSystemInfoBinding]
  producerId := "temporal.system.info.testpilot"
  producerVersion := "1"
  projectionId := Rpc.projectionId
  scopeField := Rpc.runFieldId
  operationKey := Rpc.callFieldId
  historyObservation := Rpc.serverVersionObservation
  correlatedObservation := Support.correlatedObservation
  sources := [Rpc.completedSource]
  projectionLimits := {
    events := 8, buffered := 4, keys := 2, support := 16
    work := 1000000000, eventSize := 512 }
  runLimits := {
    «transitions» := 4, obligations := 4, work := 100000000, captures := 0 } }

/- The one class is bound to the call, and the call's completion is what confirms it. -/
#guard (unaryRpc.actions.map (·.key)) == ["getSystemInfo"]
#guard (unaryRpc.sources.map (·.eventKind)) == ["instructionCompleted"]

end Temporal.Case.Realization
