// Typed composition selectors beside the production compositions of features/activity/standalone,
// which declare their members, syncs and replaced machines by field, derive each later design by
// `withMember` and select their Scenarios' composed classes with `synced` and `own`. The lifter's
// tests lift those Queries for their exact composed keys, members, syncs and replacement targets;
// lift `switchQueries` here, whose field, sync and action names carry the separators of the composed
// keys and whose two members bind actions spelled alike, for the exact keys they select; and lift
// `flickedBothOnce`, over a sync named after its first member's action, which a composition
// `withMember` derives from it keeps and `synced` selects from either member.
package fixture.members

import framework.*

// ### Separators in composed keys, and actions spelled alike in two members

enum Level derives Finite:
  case low, high

enum Switched derives Finite:
  case accepted

final case class Switch(on: Boolean) derives Finite

// The two switches' state, named apart from the composition object `Switches`.
final case class SwitchesState(left_side: Switch, right_side: Switch)

val turnOn = action("turn-on", Actor("fixture")).input[Level]("level")

// The left switch's actions, spelled as the right switch's are.
object Left:
  val tap = action(Actor("fixture"))
  val flick = action(Actor("fixture"))

object Right:
  val tap = action(Actor("fixture"))
  val flick = action(Actor("fixture"))

def toggle(s: Switch): List[Step[Switch, Switched, Nothing]] = List(
  Step(Switched.accepted, Switch(!s.on))
)

def turnOnStep(s: Switch, level: Level): List[Step[Switch, Switched, Nothing]] = List(
  Step(Switched.accepted, Switch(true))
)

object LeftSwitch extends Machine[Switch, Switched, Nothing]:
  val init = Switch(false)
  def end(switch: State) = true

  object rules extends Bindings(Left.tap ~> toggle, Left.flick ~> toggle, turnOn ~> turnOnStep)

object RightSwitch extends Machine[Switch, Switched, Nothing]:
  val init = Switch(false)
  def end(switch: State) = true

  object rules extends Bindings(Right.tap ~> toggle, Right.flick ~> toggle)

object Switches
    extends Composition[SwitchesState](_.left_side -> LeftSwitch, _.right_side -> RightSwitch):
  def end(switches: State) = true
  object syncs extends Syncs:
    sync("tap_both-ways", _.left_side -> Left.tap, _.right_side -> Right.tap)

val switchSchedule = Switches.scenario.actions(
  Switches.synced(_.right_side -> Right.tap),
  Switches.own(_.left_side, turnOn(Level.high)),
  Switches.own(_.left_side, Left.flick),
  Switches.own(_.right_side, Right.flick)
)
val turnedOn =
  Switches.property.whenAction(Switches.own(_.left_side, turnOn)) holds (_.state.left_side.on)
val tappedBoth =
  Switches.property.whenAction(Switches.synced(_.left_side -> Left.tap)) holds (after =>
    after.state.left_side.on == after.state.right_side.on
  )
val switchLimits = Limits(steps = 4, actions = 4, search = 64)

// ### A sync named after its first member's action, kept by the composition derived from it

object Flicks
    extends Composition[SwitchesState](_.left_side -> LeftSwitch, _.right_side -> RightSwitch):
  def end(switches: State) = true
  object syncs extends Syncs:
    sync(_.left_side -> Left.flick, _.right_side -> Right.flick)

object FlickOnly extends Derived(LeftSwitch.restrict(Left.flick))
object LeftFlicks extends Composition(Flicks.withMember(_.left_side -> FlickOnly))
val bothFlick = LeftFlicks.scenario.actions(LeftFlicks.synced(_.right_side -> Right.flick))
val flickedBoth =
  LeftFlicks.property.whenAction(LeftFlicks.synced(_.left_side -> Left.flick)) holds (after =>
    after.state.left_side.on == after.state.right_side.on
  )
val flickedBothOnce: Query = query verify flickedBoth in bothFlick limits switchLimits total 4

val switchQueries: Vector[Query] = Vector(
  query("turnedOn") verify turnedOn in switchSchedule limits switchLimits total 16,
  query("tappedBoth") verify tappedBoth in switchSchedule limits switchLimits total 16
)
