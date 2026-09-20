import Temporal.API
import Umpire.Command

/-!
# Resolving an action's `schema:` against the generated API

An action may name the protobuf message that types its payload. `Umpire` stores the name and asks a
platform whether it resolves; this module is Temporal's answer.

The generated schema carries no global message table: a descriptor closure hangs off each RPC, so a
message is reachable through a method that carries it. The roots below are the methods whose
closures cover the messages a Model names -- the workflow task the caller's commands travel on, the
three Nexus task RPCs the handler's replies travel on, and the one unary call a Model invokes
directly. A name outside those closures rejects, and
a Model that needs a message from elsewhere adds the root that carries it, which keeps the admitted
set tied to the generated API rather than to a list maintained here.

Only the verdict leaves this module. A Model that stored the descriptor would put the closure inside
every Case identity derived from it, and `Umpire.Command.Action` holds the name alone for that
reason.

**What the check decides.** It decides whether a name resolves. It does not check an action's class
members or `examples:` against the message's fields: the members the design's own examples name
(`BadRequest`, `Internal`) are values of `temporal.api.nexus.v1.HandlerError.error_type`, which the
generated schema types as a `string` -- so there is nothing in the descriptor to check them against.
A member becomes checkable once an action's payload declares typed fields, which is fn-85 `.8`.
-/

namespace Temporal.Case.Schema

open Temporal.API

private def workflowTask := Temporal.Api.Workflowservice.V1.WorkflowService.respondWorkflowTaskCompleted
private def workflowTaskReference : MethodReference workflowTask := by constructor

private def nexusTask := Temporal.Api.Workflowservice.V1.WorkflowService.pollNexusTaskQueue
private def nexusTaskReference : MethodReference nexusTask := by constructor

private def nexusReply := Temporal.Api.Workflowservice.V1.WorkflowService.respondNexusTaskCompleted
private def nexusReplyReference : MethodReference nexusReply := by constructor

private def nexusFailure := Temporal.Api.Workflowservice.V1.WorkflowService.respondNexusTaskFailed
private def nexusFailureReference : MethodReference nexusFailure := by constructor

private def systemInfo := Temporal.Api.Workflowservice.V1.WorkflowService.getSystemInfo
private def systemInfoReference : MethodReference systemInfo := by constructor

/-- Every message name reachable from the roots, in descriptor order. -/
def admitted : List String :=
  let closures := [
    workflowTaskReference.schema.request, workflowTaskReference.schema.response,
    nexusTaskReference.schema.request, nexusTaskReference.schema.response,
    nexusReplyReference.schema.request, nexusReplyReference.schema.response,
    nexusFailureReference.schema.request, nexusFailureReference.schema.response,
    systemInfoReference.schema.request, systemInfoReference.schema.response]
  (closures.flatMap (·.nodes.map (·.name))).eraseDups

/-- How many characters the generated descriptor of one message is. Nothing in Umpire needs this to
resolve a name; a test reads it to say what a stored descriptor would have cost. -/
def descriptorSize (fullName : String) : Nat :=
  let closures := [
    workflowTaskReference.schema.request, workflowTaskReference.schema.response,
    nexusTaskReference.schema.request, nexusTaskReference.schema.response,
    nexusReplyReference.schema.request, nexusReplyReference.schema.response,
    nexusFailureReference.schema.request, nexusFailureReference.schema.response,
    systemInfoReference.schema.request, systemInfoReference.schema.response]
  match (closures.flatMap (·.nodes)).find? (·.name == fullName) with
  | some node => node.descriptor.length
  | none => 0

/-- Whether one name is a generated message. -/
def resolves (fullName : String) : Bool := admitted.contains fullName

/-- The verdict on one action's `schema:` line. The first unresolvable alternative is the one
reported, because an author corrects one name at a time. -/
def check (fullNames : List String) : Except String Unit :=
  match fullNames.find? (!resolves ·) with
  | some unknown =>
      .error s!"'{unknown}' does not resolve to a protobuf message the generated API carries"
  | none => .ok ()

initialize Umpire.Command.installSchemaCheck check

end Temporal.Case.Schema
