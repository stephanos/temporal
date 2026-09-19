import Temporal.Case.Realization.Nexus

/-!
# fn-85's early proof point

The question the proof point asks is whether binding a Model's **action classes** to instructions,
and letting the Producer place them from a Query's path, reproduces what a hand-written whole-Program
template produces. If it does not, the party-to-entrypoint design is wrong and everything fn-85 builds
on it is wrong with it.

The comparison is the asynchronous Nexus Program, which is the one checked-in Case whose side effects
span all three entrypoints: the caller workflow schedules, the handler replies, and -- the case that
decides the design -- the handler's *completion* runs as a controller instruction over a handle slot.

`Temporal.Case.Template.nexusOperation` writes that Program directly.
`Temporal.Case.Realization.asyncNexus` writes none: it declares the same scaffolding, binds the three
side effects to action classes, and lets `Umpire.Case.Producer.assembleProgram` order them from the
path `[schedule, handlerReply, complete]`. The guards below require the two to be equal -- the same
roles, slots, observations, entrypoints, instruction ids, guards, limits and dependency edges --
everywhere but in the three bound instructions themselves: since fn-85 .8 the realization binds each
class to the typed worker instruction that carries its API message, where the template keeps the
untyped ones until fn-86 removes them, so the comparison erases those three instructions' messages
on both sides and pins the typed shapes separately. Since fn-85 .10 one realization plan serves
every Query of the caller set, which costs two scaffolding differences the comparison also sets
aside and pins on their own: the controller waits for the workflow to close before it reads history,
because a synchronous or failed operation gives it nothing else to wait on, and the workflow
finishes with a literal under a guard that runs it whether or not the operation succeeded.

What this does not yet compare is the whole Case. A Case's Contract is derived from the checked
Property's clauses and the Scenario's action order, so re-authoring the Model's two waits
(`awaitStart`, `awaitSuccess`) as three side effects changes the correlated clauses, their occurrence
bound and their evidence rules by construction. That comparison needs the Model re-authored through
the `entity`, `action`, `machine` and `observation` commands, which is fn-85 .2 and .3; this module
pins the half the design rests on, and the receipt records the rest.
-/

namespace Temporal.Case.Tests.ProofPoint

open Umpire
open temporal.server.api.testpilot.v1 hiding ModelValue

/-- The identity the comparison runs under. Both sides derive every other id from the fixture name,
so one name fixes the Program id and the workflow type on both. -/
private def identity : Umpire.Case.Producer.Identity :=
  Umpire.Case.Producer.Identity.ofFixture "temporal.case" "async-nexus"

private def service : String := "umpire.case.service"
private def operation : String := "complete"

/-- The path Query 2 takes: the caller schedules the operation, the handler accepts it
asynchronously, and the handler completes it. -/
private def path : List DefinitionId := [
  Temporal.Case.Realization.Nexus.scheduleAction,
  Temporal.Case.Realization.Nexus.handlerReplyAction,
  Temporal.Case.Realization.Nexus.completeAction]

/-- The Program the hand-written template produces. The template binds no action, so its own path is
irrelevant to it. -/
private def templateProgram : Except Umpire.Case.Compiler.Error Program :=
  (Temporal.Case.Template.nexusOperation service operation .async).program identity

/-- The Program the realization assembles from the path. -/
private def assembledProgram : Except Umpire.Case.Compiler.Error Program :=
  (Temporal.Case.Realization.asyncNexus service operation).program identity (path := path)

/- Both sides assemble. A rejection here is an unbound or unplaced action, not a difference. -/
#guard templateProgram.toOption.isSome
#guard assembledProgram.toOption.isSome

/-- The wire bytes of one assembled Program, or `none` when the assembly rejected. The generated
`Program` carries no `BEq`, and `Testpilot.ProtoJSON.canonical` needs `IO`, so the comparison goes
through the protobuf library's own pure encoder: two values of one generated type encode to equal
bytes exactly when they are equal, which is the equality a fixture comparison means anyway. -/
private def wire (program : Except Umpire.Case.Compiler.Error Program) : Option ByteArray :=
  match program with
  | .ok program => (Protobuf.encode program).toOption
  | .error _ => none

/- Both sides encode, so neither guard below passes by comparing two failures. -/
#guard (wire templateProgram).isSome
#guard (wire assembledProgram).isSome

/-- The instruction ids the three bound classes land on, whose messages the two sides spell
differently: untyped in the template, typed in the realization. -/
private def boundIds : List String :=
  ["start-nexus-operation", "respond-async", "complete-nexus-operation"]

/-- The one instruction the realization spells differently for every path: its finish returns a
literal under an unconditional guard, where the template's returns the awaited payload. -/
private def finishId : String := "finish-workflow"

/-- The one node the realization adds on every path: the close-event read before the history read.
-/
private def awaitCloseId : String := "await-close"

/-- A Program with the bound instructions' messages erased and the two path-serving differences set
aside, so what remains is the scaffolding, the placement and every coordinate a Contract reads back
through. -/
private def scaffolding (program : Except Umpire.Case.Compiler.Error Program) :
    Except Umpire.Case.Compiler.Error Program :=
  program.map fun program =>
    { program with entrypoints := program.entrypoints.map fun entrypoint =>
        { entrypoint with instructions :=
            (entrypoint.instructions.filter (·.instruction_id != awaitCloseId)).map fun node =>
              if boundIds.contains node.instruction_id then { node with instruction := none }
              else if node.instruction_id == finishId then
                { node with instruction := none, guard := none }
              else node } }

/-- The instruction ids of one entrypoint of the assembled Program, in order. -/
private def assembledIds (entrypointId : String) : List String :=
  match assembledProgram with
  | .ok program =>
      ((program.entrypoints.find? (·.entrypoint_id == entrypointId)).map fun entrypoint =>
        entrypoint.instructions.toList.map (·.instruction_id)).getD []
  | .error _ => []

/- The two differences set aside, pinned: the close-event read sits between the completion and the
history read, and the finish runs under an unconditional guard. -/
#guard assembledIds "controller" ==
  ["start-workflow", "await-completion-authority", "complete-nexus-operation", "await-close",
    "history"]
#guard (match assembledProgram with
  | .ok program =>
      ((program.entrypoints.find? (·.entrypoint_id == "workflow")).bind fun entrypoint =>
        (entrypoint.instructions.find? (·.instruction_id == finishId)).map (·.guard.isSome))
  | .error _ => none) == some true

/- **The proof point.** Outside the three bound instructions, the assembled Program is byte-identical
to the template's, so the path and the class bindings carry everything the template wrote by hand:
the same roles, slots, observations, entrypoints, instruction ids, guards, limits and dependency
edges. The bound instructions differ only in spelling: the realization's carry the typed messages. -/
#guard (wire (scaffolding templateProgram)).map (·.toList) ==
  (wire (scaffolding assembledProgram)).map (·.toList)
#guard (wire templateProgram).map (·.toList) != (wire assembledProgram).map (·.toList)

/-- The typed message a bound instruction of the assembled Program carries, named by its arm. -/
private def carriedArm (entrypointId instructionId : String) : String :=
  match assembledProgram with
  | .ok program =>
      match program.entrypoints.find? (·.entrypoint_id == entrypointId) with
      | some entrypoint =>
          match entrypoint.instructions.find? (·.instruction_id == instructionId) with
          | some node =>
              match node.instruction.bind (·.instruction) with
              | some (.workflow_command carried) =>
                  match carried.command.bind (·.attributes) with
                  | some (.schedule_nexus_operation_command_attributes attributes) =>
                      s!"schedule {attributes.endpoint} {attributes.service} {attributes.operation}"
                  | _ => "command?"
              | some (.nexus_handler_reply reply) =>
                  match reply.reply with
                  | some (.response { variant := some (.async_success _), .. }) =>
                      s!"async-reply {reply.handle_slot_id}"
                  | _ => "reply?"
              | some (.nexus_operation_completion completion) =>
                  match completion.result with
                  | some (.payload _) => s!"complete-payload {completion.handle_slot_id}"
                  | _ => "completion?"
              | _ => "untyped"
          | none => "missing"
      | none => "missing"
  | .error _ => "rejected"

/- The three bound instructions carry the typed messages of their classes, on the template's own
endpoint role, service, operation and handle slot. -/
#guard carriedArm "workflow" "start-nexus-operation" ==
  "schedule temporal.nexus-endpoint umpire.case.service complete"
#guard carriedArm "handler" "respond-async" == "async-reply completion-authority"
#guard carriedArm "controller" "complete-nexus-operation" == "complete-payload completion-authority"

/-! ### What the equality covers

The guard above is whole-Program equality, which is the check that matters. These name the parts a
reader would otherwise have to take on trust, so a future change that breaks one of them fails with
the part named rather than with "the Programs differ". -/

private def entrypointIds (program : Except Umpire.Case.Compiler.Error Program) : List String :=
  match program with
  | .ok program => program.entrypoints.toList.map (·.entrypoint_id)
  | .error _ => []

private def instructionIds
    (program : Except Umpire.Case.Compiler.Error Program)
    (entrypointId : String) : List String :=
  match program with
  | .ok program =>
      match program.entrypoints.find? (·.entrypoint_id == entrypointId) with
      | some entrypoint => entrypoint.instructions.toList.map (·.instruction_id)
      | none => []
  | .error _ => []

/- The three entrypoints, in the order the plan declares them. -/
#guard entrypointIds assembledProgram == ["controller", "workflow", "handler"]

/- The controller interleaves scaffolding and one action: it starts the workflow, waits for the
authority, performs the completion, waits for the workflow to close, then reads history. Reproducing
this sequence from item order is what the design claims. -/
#guard instructionIds assembledProgram "controller" ==
  ["start-workflow", "await-completion-authority", "complete-nexus-operation", "await-close",
    "history"]

/- The workflow schedules the operation, then waits and finishes as scaffolding. -/
#guard instructionIds assembledProgram "workflow" ==
  ["start-nexus-operation", "await-nexus-operation", "finish-workflow"]

/- The handler's only instruction is its reply. -/
#guard instructionIds assembledProgram "handler" == ["respond-async"]

/- Each entrypoint's sequence matches the template's, which is where the dependency edges come from:
after fn-87 an instruction depends on the one before it in its entrypoint unless it says otherwise.
The one addition is the close-event read every path of the caller set needs. -/
#guard (instructionIds assembledProgram "controller").filter (· != awaitCloseId) ==
  instructionIds templateProgram "controller"
#guard instructionIds assembledProgram "workflow" == instructionIds templateProgram "workflow"
#guard instructionIds assembledProgram "handler" == instructionIds templateProgram "handler"

/-! ### The rejections the assembly owns

A realization that binds a class no entrypoint places, and a path that performs a class no binding
covers, are both realization errors rather than silently smaller Programs. -/

private def unplacedRealization : Umpire.Case.Producer.Realization :=
  let realized := Temporal.Case.Realization.asyncNexus service operation
  { realized with plan := { realized.plan with entrypoints := [] } }

/- A bound action that no entrypoint places rejects: a Program missing a side effect would still
run, and would prove nothing. -/
#guard (unplacedRealization.program identity (path := path)).toOption.isNone

private def unboundRealization : Umpire.Case.Producer.Realization :=
  let realized := Temporal.Case.Realization.asyncNexus service operation
  { realized with actions := [] }

/- A path whose action an `actions` item names but no binding covers rejects. -/
#guard (unboundRealization.program identity (path := path)).toOption.isNone

private def twicePlacedRealization : Umpire.Case.Producer.Realization :=
  let realized := Temporal.Case.Realization.asyncNexus service operation
  { realized with
    plan := { realized.plan with
      entrypoints := realized.plan.entrypoints ++ [
        { activate := fun _ nodes => Testpilot.Authoring.Program.controller "second" nodes
          items := [.actions [Temporal.Case.Realization.Nexus.completeAction]] }] } }

/- A class two `actions` items name rejects: emitting its nodes twice would give them the same
instruction ids, which is a realization mistake and not a larger Program. -/
#guard (twicePlacedRealization.program identity (path := path)).toOption.isNone

/- An action performed twice on one path gets two distinct instruction ids, so preparation never
sees a duplicate. The retry Query of fn-85 .11 is the first Model that needs it. -/
#guard instructionIds
    ((Temporal.Case.Realization.asyncNexus service operation).program identity
      (path := path ++ [Temporal.Case.Realization.Nexus.completeAction])) "controller" ==
  ["start-workflow", "await-completion-authority", "complete-nexus-operation",
    "complete-nexus-operation-2", "await-close", "history"]

end Temporal.Case.Tests.ProofPoint
