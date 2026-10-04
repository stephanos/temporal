// The compositions of temporal/standaloneactivity declared again with typed selectors: members, syncs
// and the replaced machine by field, each later design derived from the first by `withMember`, and
// the Scenarios and the `whenAction` that name composed classes by key there selected with
// `synced` and `own`. The lifter's tests lift these beside the production declarations and require
// the same compositions, keys and Property, but for names and positions; and lift `switchQueries`,
// whose field, sync and action names carry the separators of the composed keys and whose two members
// bind actions spelled alike, for the exact keys they select.
package fixture.members

import temporal.standaloneactivity.{
  ackLoss,
  acknowledge,
  addActivityTask,
  admissionEnds,
  answerDelivery,
  attemptStart,
  control,
  crash,
  currentRecord,
  deliver,
  dispatch,
  dispatchQueue,
  eight,
  enqueue,
  five,
  forgetfulQueue,
  idleOverMatching,
  idleOverQueue,
  lossyMatchingQueue,
  matchingQueue,
  persistTask,
  seven,
  six,
  staleRecord,
  standaloneActivity,
  stoppedBeforeRetry,
  syncMatch,
  three,
  volatileQueue,
  Active,
  Control,
  OverMatching,
  OverQueue,
  StandaloneActivityState,
  SystemFamily
}
import temporal.worker.Phase as WorkerPhase
import umpire.*

given Family = SystemFamily

// ### The design over the opaque queue, and the stale record over it

val currentOverQueueTyped: Composition[OverQueue] =
  compose[OverQueue](_.activity -> currentRecord, _.queue -> dispatchQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .ends(s => admissionEnds(s.activity))

val staleOverQueueTyped: Composition[OverQueue] =
  currentOverQueueTyped.withMember(_.activity -> staleRecord)

def typedOverQueueQueries(c: Composition[OverQueue]): Vector[Query] =
  val oneActive =
    c.property("atMostOneActive") holds (after => after.state.activity.active != Active.two)
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .starts(idleOverQueue)
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.activity, control(Control.pause)),
      c.synced(_.queue -> deliver)
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .starts(idleOverQueue)
    .actions(
      c.synced(_.queue -> enqueue),
      c.synced(_.activity -> attemptStart),
      c.own(_.activity, control(Control.pause))
    )
  val duplicate = c
    .scenario("duplicateDelivery")
    .starts(idleOverQueue)
    .actions(
      c.synced(_.activity -> dispatch),
      c.synced(_.activity -> attemptStart),
      c.synced(_.activity -> attemptStart)
    )
  Vector(
    query(s"${c.name}.staleDelivery") verify oneActive in stale limits three,
    query(s"${c.name}.admittedBeforePause") verify oneActive in prePause limits three,
    query(s"${c.name}.duplicateDelivery") verify oneActive in duplicate limits three
  )

val currentOverQueueTypedQueries: Vector[Query] = typedOverQueueQueries(currentOverQueueTyped)
val staleOverQueueTypedQueries: Vector[Query] = typedOverQueueQueries(staleOverQueueTyped)

// ### The designs over the detailed queues, each derived from the first by its replaced member

val currentOverMatchingTyped: Composition[OverMatching] =
  compose[OverMatching](_.activity -> currentRecord, _.queue -> matchingQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .replaces(_.queue, dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val staleOverMatchingTyped: Composition[OverMatching] =
  currentOverMatchingTyped.withMember(_.activity -> staleRecord)
val currentOverForgetfulTyped: Composition[OverMatching] =
  currentOverMatchingTyped.withMember(_.queue -> forgetfulQueue)
val currentOverVolatileTyped: Composition[OverMatching] =
  currentOverMatchingTyped.withMember(_.queue -> volatileQueue)
// The lossy provider refines the interface that allows storage loss, so it stands in for that one.
val currentOverLossyMatchingTyped: Composition[OverMatching] =
  currentOverMatchingTyped.withMember(_.queue -> lossyMatchingQueue)

def typedOverMatchingQueries(c: Composition[OverMatching]): Vector[Query] =
  val oneActive =
    c.property("atMostOneActive") holds (after => after.state.activity.active != Active.two)
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .starts(idleOverMatching)
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.own(_.activity, control(Control.pause)),
      c.synced(_.activity -> attemptStart)
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .starts(idleOverMatching)
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.synced(_.activity -> attemptStart),
      c.own(_.activity, control(Control.pause))
    )
  val lostAck = c
    .scenario("deliveredAgainAfterLostAck")
    .starts(idleOverMatching)
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.synced(_.activity -> attemptStart),
      c.own(_.queue, ackLoss),
      c.synced(_.queue -> deliver)
    )
  val crashAfterCommit = c
    .scenario("crashAfterAdmissionCommit")
    .starts(idleOverMatching)
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, syncMatch),
      c.synced(_.activity -> attemptStart),
      c.own(_.queue, crash),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, syncMatch),
      c.synced(_.activity -> attemptStart)
    )
  Vector(
    query(s"${c.name}.staleDelivery") verify oneActive in stale limits five,
    query(s"${c.name}.admittedBeforePause") verify oneActive in prePause limits five,
    query(s"${c.name}.deliveredAgainAfterLostAck") verify oneActive in lostAck limits seven,
    query(s"${c.name}.crashAfterAdmissionCommit") verify oneActive in crashAfterCommit limits eight
  )

val currentOverMatchingTypedQueries: Vector[Query] =
  typedOverMatchingQueries(currentOverMatchingTyped)
val staleOverMatchingTypedQueries: Vector[Query] = typedOverMatchingQueries(staleOverMatchingTyped)
val currentOverLossyMatchingTypedQueries: Vector[Query] =
  typedOverMatchingQueries(currentOverLossyMatchingTyped)

// ### The cross-entity claim, about every attempt start the sync pairs

val startedByPollingWorkerTyped: Property[StandaloneActivityState] =
  standaloneActivity
    .property("startedByPollingWorkerTyped")
    .whenAction(standaloneActivity.synced(_.worker -> temporal.worker.serve)) holds
    (_.state.worker.phase == WorkerPhase.polling)

val stoppedWorkerStartsNothingTyped: Query =
  query verify startedByPollingWorkerTyped in stoppedBeforeRetry limits six

// ### Separators in composed keys, and actions spelled alike in two members

enum Level derives Finite:
  case low, high

enum Switched derives Finite:
  case accepted

final case class Switch(on: Boolean) derives Finite

final case class Switches(left_side: Switch, right_side: Switch)

val turnOn = action("turn-on", Party("fixture")).input[Level]("level")

/** The left switch's actions, spelled as the right switch's are. */
object Left:
  val tap = action("tap", Party("fixture"))
  val flick = action("flick", Party("fixture"))

object Right:
  val tap = action("tap", Party("fixture"))
  val flick = action("flick", Party("fixture"))

def toggle(s: Switch): List[Step[Switch, Switched, Nothing]] = List(
  Step(Switched.accepted, Switch(!s.on))
)

def turnOnStep(s: Switch, level: Level): List[Step[Switch, Switched, Nothing]] = List(
  Step(Switched.accepted, Switch(true))
)

val leftSwitch = machine[Switch, Switched, Nothing] {
  starts(Switch(false))
  ends(_ => true)
  steps(Left.tap ~> toggle, Left.flick ~> toggle, turnOn ~> turnOnStep)
}

val rightSwitch = machine[Switch, Switched, Nothing] {
  starts(Switch(false))
  ends(_ => true)
  steps(Right.tap ~> toggle, Right.flick ~> toggle)
}

val switches = compose[Switches](_.left_side -> leftSwitch, _.right_side -> rightSwitch)
  .sync("tap_both-ways", _.left_side -> Left.tap, _.right_side -> Right.tap)
  .ends(_ => true)

val switchSchedule = switches.scenario.actions(
  switches.synced(_.right_side -> Right.tap),
  switches.own(_.left_side, turnOn(Level.high)),
  switches.own(_.left_side, Left.flick),
  switches.own(_.right_side, Right.flick)
)
val turnedOn =
  switches.property.whenAction(switches.own(_.left_side, turnOn)) holds (_.state.left_side.on)
val tappedBoth =
  switches.property.whenAction(switches.synced(_.left_side -> Left.tap)) holds (after =>
    after.state.left_side.on == after.state.right_side.on
  )
val switchLimits = Limits(steps = 4, actions = 4, search = 64)

val switchQueries: Vector[Query] = Vector(
  query("turnedOn") verify turnedOn in switchSchedule limits switchLimits,
  query("tappedBoth") verify tappedBoth in switchSchedule limits switchLimits
)
