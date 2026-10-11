// A composition's bounded progress claims (fn-150): predicates of the composed state that read both
// members, here two fields filled by one machine, and assumptions that make composed steps fair --
// a sync, every class or one of its classes, and a member's own action, every class or one -- each
// lifted with its composition and member field. The tests lift `drainsThrough` and `gradedThrough`
// and compare the IR with expected/composedProgress.json.
package fixture.composedprogress

import framework.*; import framework.Finite.given

enum Level derives Finite:
  case empty, filled, sent

final case class Tank(level: Level) derives Finite

enum Outcome derives Finite:
  case filled, sent, drained

enum Grade derives Finite:
  case low, high

val grade = input[Grade]

val fill = action(Actor("pump")).input(grade)
val send = action(Actor("valve"))
val drain = action(Actor("valve"))

type TankStep = Step[Tank, Outcome, Nothing]

def filled(t: Tank, g: Grade): List[TankStep] = // format: off
  if t.level == Level.empty then List(Step(Outcome.filled, Tank(Level.filled))) else Nil
def sent(t: Tank): List[TankStep] =
  if t.level == Level.filled then List(Step(Outcome.sent, Tank(Level.sent))) else Nil
def drained(t: Tank): List[TankStep] =
  if t.level == Level.sent then List(Step(Outcome.drained, Tank(Level.empty))) else Nil
  // format: on
object Tanks extends Machine[Tank, Outcome, Nothing]:
  val init = Tank(Level.empty)
  def end(t: State) = true
  object rules extends Bindings(fill ~> filled, send ~> sent, drain ~> drained)

// Two tanks of one machine, told apart by their fields: the left one's send fills the right one.
final case class Line(left: Tank, right: Tank) derives Finite

object Pipeline extends Composition[Line](_.left -> Tanks, _.right -> Tanks):
  def end(l: State) = true
  object syncs extends Syncs:
    sync("transfer", _.left -> send, _.right -> fill)

// The sync, every class of it, and the right tank's own send, every class of it.
val flowing = assume.fair(Pipeline.synced(_.left -> send), Pipeline.own(_.right, send))

// One class of the sync, named from the side that takes the input, and one class of each own action.
val graded = assume("graded").fair(
  Pipeline.synced(_.right -> fill(Grade.high)),
  Pipeline.own(_.left, fill(Grade.low)),
  Pipeline.own(_.right, drain)
)

val drainsThrough: Progress[Line] =
  Pipeline.leadsTo("drainsThrough")( // format: off
    l => l.left.level == Level.filled && l.right.level == Level.empty,
    l => l.right.level == Level.sent,
    within = 2, // format: on
    flowing
  )

val gradedThrough: Progress[Line] =
  Pipeline.leadsTo("gradedThrough")( // format: off
    l => l.left.level == Level.filled,
    l => l.right.level != Level.empty,
    within = 3, // format: on
    flowing,
    graded
  )
