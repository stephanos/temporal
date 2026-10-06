// Capabilities sections (fn-134.2), each Property a declared capability's companion brings lifted
// as a Property, a Scenario and a Query named `<machine>.<property>`, the Property with the def it
// was expanded from as its origin:
//   - a machine's own section, whose Queries are bounded in its `queries` section, one override
//     among them, and a Query of its own reading a generated Property with `claim`;
//   - a Property reading two capabilities (Holdable's, reading Takeable's `working` too), brought
//     to the machines that declare both and not to Chore, which declares Holdable alone;
//   - a Property waived with `except` and one replaced with `overriding`, which keeps its origin;
//   - a shared set two compositions extend, `PairCapabilities(this)`, whose capabilities read the
//     composition it is given and whose waiver both inherit, one adding a waiver of its own.
// The kinds are the fixture's own, each a case class whose companion defines its Properties.
package fixture.capabilitysections

import umpire.*

enum Phase derives Finite:
  case queued, running, held, done

// A task's state, named apart from the machine object `Task`.
final case class TaskState(phase: Phase) derives Finite

enum Answer derives Finite:
  case ok, gone

given Ok[Answer] = Ok(Answer.ok)

enum Note derives Finite:
  case started, heldNote, released, finished

object client extends Actor
object worker extends Actor

val take = action(worker)
val finish = action(worker)
val hold = action(client)
val release = action(client)

// The task has a terminal status set: `status` reads a state's status, `done` says which statuses
// close it, and a closed task answers `refused`.
final case class Sealable[S, P, O](status: S => P, done: P => Boolean, refused: O)
    extends CapabilityOf[S, O, Nothing]

object Sealable extends CapabilityKind:
  // No step leaves a done status.
  def sealedStaysSealed[S, P](m: Declares[S])(status: S => P, done: P => Boolean): Property[S] =
    m.property holdsAcross ((before, after) =>
      !done(status(before)) || status(after.state) == status(before)
    )

  // Every step from a done status keeps the state and answers `refused`.
  def sealedIsRefused[S, P](m: Declares[S])(
      status: S => P,
      done: P => Boolean,
      refused: m.Outcome
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !done(status(before)) || (after.state == before && after.outcome == refused)
    )

// The task is held by `hold` and released by `release`; `held` says where it is.
final case class Holdable[S](
    hold: ClassRef | Composed,
    release: ClassRef | Composed,
    held: S => Boolean
) extends CapabilityOf[S, Nothing, Nothing]

object Holdable extends CapabilityKind:
  // No step from a held state lands in a working one: it reads Takeable's `working`, so a machine
  // declaring both is brought it.
  def heldIsNotTaken[S](m: Declares[S])(held: S => Boolean, working: S => Boolean): Property[S] =
    m.property.never(s => working(s.state)).from(held)

// The task's work is taken by `take`; `working` says where a worker holds it.
final case class Takeable[S](take: ClassRef | Composed, working: S => Boolean)
    extends CapabilityOf[S, Nothing, Nothing]

object Takeable extends CapabilityKind

// The tasks' status sets and step functions; a control of a done task is answered `gone`.
object Tasks:
  def phase(t: TaskState): Phase = t.phase
  def done(p: Phase): Boolean = p == Phase.done
  def held(t: TaskState): Boolean = t.phase == Phase.held
  def working(t: TaskState): Boolean = t.phase == Phase.running
  def over(t: TaskState): Boolean = done(t.phase)

  private def gone(t: TaskState): List[Step[TaskState, Answer, Note]] = List(Step(Answer.gone, t))

  def take(t: TaskState): List[Step[TaskState, Answer, Note]] =
    if t.phase == Phase.queued then enter(TaskState(Phase.running), Note.started) else disabled
  def finish(t: TaskState): List[Step[TaskState, Answer, Note]] =
    if t.phase == Phase.running then enter(TaskState(Phase.done), Note.finished) else disabled
  def hold(t: TaskState): List[Step[TaskState, Answer, Note]] =
    if done(t.phase) then gone(t)
    else if t.phase.in(Phase.queued, Phase.running) then enter(TaskState(Phase.held), Note.heldNote)
    else disabled
  def release(t: TaskState): List[Step[TaskState, Answer, Note]] =
    if done(t.phase) then gone(t)
    else if t.phase == Phase.held then enter(TaskState(Phase.queued), Note.released)
    else disabled

  // The chore's answer to a done task: it keeps the state, whatever it answers.
  def refusedOrKept[S, P](m: Declares[S])(
      status: S => P,
      done: P => Boolean,
      refused: m.Outcome
  ): Property[S] =
    m.property holdsAcross ((before, after) => !done(status(before)) || after.state == before)

val three = Limits(steps = 3, actions = 3, search = 512)
val two = Limits(steps = 2, actions = 2, search = 256)

// 4 states; 4 action classes.
object Task extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)

  object rules
      extends Bindings(
        take ~> Tasks.take,
        finish ~> Tasks.finish,
        hold ~> Tasks.hold,
        release ~> Tasks.release
      )

  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)
    val holdable: Capability = Holdable(hold = hold, release = release, held = Tasks.held)
    val takeable: Capability = Takeable(take = take, working = Tasks.working)

  // Free verify Queries: 4 states x 4 classes x 3 steps = 48, sealedIsRefused's x 2 steps = 32.
  object queries:
    capabilities.bound(three, Sealable.sealedIsRefused -> two)

    // Pinned: 4 states x min(3 steps, 2 scheduled) = 8.
    val heldThenTaken = scenario.actions(hold, take)
    val heldStaysUntaken =
      query verify capabilities.claim(Holdable.heldIsNotTaken) in heldThenTaken limits three

// Its twin without a worker: it declares Holdable without Takeable, so it is brought no
// heldIsNotTaken; it waives sealedStaysSealed and replaces sealedIsRefused.
object Chore extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)

  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)

  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)
    val holdable: Capability = Holdable(hold = hold, release = release, held = Tasks.held)
    except(Sealable.sealedStaysSealed, because = "a fixture's waiver: the chore keeps no status")
    overriding(
      Sealable.sealedIsRefused -> Tasks.refusedOrKept,
      because = "a fixture's override: the chore answers a done task as it likes"
    )

  // 4 states x 2 classes x 3 steps = 24.
  object queries:
    capabilities.bound(three)

// ### A shared set: two compositions' capabilities, read through their left member

// Two tasks' state, named apart from the composition objects.
final case class PairState(left: TaskState, right: TaskState) derives Finite

object Pairs:
  def over(p: PairState): Boolean = Tasks.over(p.left) && Tasks.over(p.right)

// What a pair of tasks can do, as its left task: written once, with its waiver, for each pair.
abstract class PairCapabilities(c: Composition[PairState])(using
    Declaring[PairState, String, String]
) extends Capabilities:
  val sealable: Capability =
    Sealable(status = through(_.left, Tasks.phase), done = Tasks.done, refused = "gone")
  val holdable: Capability = Holdable(
    hold = c.own(_.left, hold),
    release = c.own(_.left, release),
    held = through(_.left, Tasks.held)
  )
  val takeable: Capability =
    Takeable(take = c.own(_.left, take), working = through(_.left, Tasks.working))
  except(
    Sealable.sealedIsRefused,
    because = "a fixture's shared waiver: a pair answers a done task with its member's outcome"
  )

// 16 states x (4 + 2) classes x 2 steps = 192.
object TaskPair extends Composition[PairState](_.left -> Task, _.right -> Chore):
  def end(p: State) = Pairs.over(p)
  object syncs extends Syncs
  object capabilities extends PairCapabilities(this)
  object queries:
    capabilities.bound(two)

// 16 states x (4 + 4) classes x 2 steps = 256.
object MirrorPair extends Composition[PairState](_.left -> Task, _.right -> Task):
  def end(p: State) = Pairs.over(p)
  object syncs extends Syncs
  object capabilities extends PairCapabilities(this):
    except(
      Holdable.heldIsNotTaken,
      because =
        "a fixture's own waiver: the mirror's right task may be taken while the left is held"
    )
  object queries:
    capabilities.bound(two)
