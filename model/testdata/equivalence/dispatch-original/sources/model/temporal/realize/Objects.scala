// A realization object: one realization of one machine, typed by it, its declarations in named
// sections, as a machine object holds one machine (fn-133 Part C).
//
//   object Standalone extends Realizes(ActivitySystem):
//     object controller extends Controller(perform(…), onPath(…)(…), …)
//     object workers extends Workers(attempts)
//     object evidence extends Evidences(answered(Fact.statusScheduled, startActivity), …)
//     object serverSteps extends ServerSteps(…)
//     object controls extends Controls(dispatchHold)
//
// The sections come in that order, each at most once, and an object holds nothing else. The
// realization is named after its object with the first letter lowered, and its Definition ID is
// the object's qualified name. What its machine and the kit already know is derived, not stated:
// the operation is its machine's entity; its roles are the WorkflowService and the Case's task
// queue, and each other kit role its scripts' calls, activations, faults, controls and protobuf
// literals name; a delivery step is each class an activity script starts with, a timer step each
// deadline timer of an input its `deadlines` declaration sets, at `deadlineMs`, and the machine's
// backoff timer, at `firstRetryBackoffMs`, where an activity script runs the attempts it retries.
// A `serverSteps` section adds steps or overrides a derived one; an override equal to the derived
// step is refused. The lifter (model/irgen/Realizations.scala) reads the object and writes the
// record `temporalRealization` writes.
package temporal.realize

import framework.Machine
import framework.realize.*

// The header of a realization object of `machine`: what is not derived.
abstract class Realizes[S, O, F <: AnyRef](
    val machine: Machine[S, O, F],
    val learned: Vector[Learned] = Vector.empty,
    val observations: Vector[Observed] = Vector(correlated),
    val requiredSettings: Vector[RequiredSetting] = Vector.empty
):
  // What the server answered a call of the controller, where it succeeded, the kind named after
  // `fact`, a fact of the machine. The kit's `answered`, typed by the machine.
  def answered(fact: F | EveryValue, call: Command | Instruction, confirms: Taking*) =
    temporal.realize.answered(fact, call, confirms*)

  // `answered` for a kind of its own, `kind`, that confirms `records`, a fact of the machine.
  def answeredAs(
      kind: Fact,
      records: F | EveryValue,
      call: Command | Instruction,
      confirms: Taking*
  ) =
    temporal.realize.answeredAs(kind, records, call, confirms*)

  // The kit's `delivered`, of `fact`, a fact of the machine.
  def delivered(
      fact: F | EveryValue,
      attempts: Script,
      attempt: Long,
      after: Command | Instruction,
      confirms: Taking*
  ) = temporal.realize.delivered(fact, attempts, attempt, after, confirms*)

// A realization object derived from `base`, of `machine`: the base's sections with the changes its
// `changes` section makes to the controller, as a `Derived` machine is its base with rules
// changed. Its header is the base's.
abstract class DerivesFrom[S, O, F <: AnyRef](
    val base: Realizes[?, ?, ?],
    machine: Machine[S, O, F]
) extends Realizes[S, O, F](machine)

// The controller: the calls and controls of a Case, in the order its path makes them.
abstract class Controller(val items: Item*)

// The scripts the system's own activations run, such as an activity's attempts.
abstract class Workers(val scripts: Script*)

// The evidence each fact of the machine is confirmed by.
abstract class Evidences(val items: (Evidence | EvidenceRef[?, ?] | TypedEvidence[?])*)

// The server steps the realization states beyond, or instead of, the derived ones.
abstract class ServerSteps(
    val steps: (ServerStep | ActivityExternalSettlement[?, ?] |
      ActivityExternalSettlement.Scheduled | ActivityResetSettlement)*
)

// The actuators a run needs beyond its commands.
abstract class Controls(val actuators: Actuator*)

// The changes a derived realization makes to its base's controller, each at the base's item whose
// command is the one named.
abstract class Changes(val changes: Change*)

// One change: items inserted after the item whose command is `step`, or that item replaced.
final case class Change(step: Command | Instruction, replaces: Boolean)(val items: Item*)

// Inserts `items` after the base controller's item whose command is `after`.
def inserting(after: Command | Instruction)(items: Item*): Change =
  Change(after, replaces = false)(items*)

// Replaces the base controller's item whose command is `step` with `items`.
def replacing(step: Command | Instruction)(items: Item*): Change =
  Change(step, replaces = true)(items*)

// Every value of a fact case with fields, named by its companion: `everyValue(Fact.statusTimedOut)`.
// A machine's fact type holds the case's values, not its companion, so a typed declaration takes
// the companion only through this, and does not check that the case is the machine's.
opaque type EveryValue <: AnyRef = AnyRef

// The fact case `cases` names with every value of it.
def everyValue(cases: AnyRef): EveryValue = cases
