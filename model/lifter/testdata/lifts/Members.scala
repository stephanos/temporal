// Typed composition selectors beside the production compositions of temporal/standaloneactivity,
// which declare their members, syncs and replaced machines by field, derive each later design by
// `withMember` and select their Scenarios' composed classes with `synced` and `own`. The lifter's
// tests lift those Queries for their exact composed keys, members, syncs and replacement targets;
// lift the cross-entity claim here, which names the attempt-start sync by the worker's side where
// production names it by the activity's; and lift `switchQueries`, whose field, sync and action
// names carry the separators of the composed keys and whose two members bind actions spelled alike,
// for the exact keys they select; and lift `flickedBothOnce`, over a sync named after its first
// member's action and a composition `withMember` derives from it.
package fixture.members

import temporal.standaloneactivity.{six, standaloneActivity, Paths}
import temporal.standaloneactivity.StandaloneActivityState
import temporal.standaloneactivity.SystemFamily.given
import temporal.worker.Phase as WorkerPhase
import umpire.*

// ### The cross-entity claim, about every attempt start the sync pairs

val startedByPollingWorkerTyped: Property[StandaloneActivityState] =
  standaloneActivity.property
    .whenAction(standaloneActivity.synced(_.worker -> temporal.worker.serve)) holds
    (_.state.worker.phase == WorkerPhase.polling)

val stoppedWorkerStartsNothingTyped: Query =
  query verify startedByPollingWorkerTyped in Paths.stoppedBeforeRetry limits six total 3456

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
  val tap = action(Party("fixture"))
  val flick = action(Party("fixture"))

object Right:
  val tap = action(Party("fixture"))
  val flick = action(Party("fixture"))

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

// ### A sync named after its first member's action, kept by the composition derived from it

val flicks = compose[Switches](_.left_side -> leftSwitch, _.right_side -> rightSwitch)
  .sync(_.left_side -> Left.flick, _.right_side -> Right.flick)
  .ends(_ => true)
val flickOnly = leftSwitch.restrict(Left.flick)
val leftFlicks = flicks.withMember(_.left_side -> flickOnly)
val bothFlick = leftFlicks.scenario.actions(leftFlicks.synced(_.right_side -> Right.flick))
val flickedBoth =
  leftFlicks.property.whenAction(leftFlicks.synced(_.left_side -> Left.flick)) holds (after =>
    after.state.left_side.on == after.state.right_side.on
  )
val flickedBothOnce: Query = query verify flickedBoth in bothFlick limits switchLimits total 4

val switchQueries: Vector[Query] = Vector(
  query("turnedOn") verify turnedOn in switchSchedule limits switchLimits total 16,
  query("tappedBoth") verify tappedBoth in switchSchedule limits switchLimits total 16
)
