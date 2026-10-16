package fixture.subset

import framework.dsl.*
import framework.dsl.Finite.given

final case class State(flag: Boolean, count: UpTo[2])
type Result = Step[State, Boolean, Boolean]
given Ok[Boolean] = Ok(true)
val press = action(Actor("user"))

def booleanLiteral(s: State): Boolean = true
def integerLiteral(s: State): Int = 1
def textLiteral(s: State): String = "ok"
def reference(s: State): State = s
def field(s: State): Boolean = s.flag
def called(s: State): Boolean = field(s)
def constructed(s: State): State = State(s.flag, UpTo(1))
def wrapped(s: State): State = constructed(s = s: State)
def localValue(s: State): Boolean =
  val value = s.flag
  value
def conditional(s: State): Boolean = if s.flag then false else true
def matched(s: State): Boolean = s.flag match
  case flag @ true => flag
  case false       => false
def listed(s: State): List[Boolean] = List(s.flag, true)
def optional(s: State): Option[Boolean] = if s.flag then Some(true) else None
def stepRecord(s: State): List[Result] = List(Step(true, s))
def equal(s: State): Boolean = s.flag == true
def unequal(s: State): Boolean = s.flag != false
def conjoined(s: State): Boolean = s.flag && true
def disjoined(s: State): Boolean = s.flag || false
def negated(s: State): Boolean = !s.flag
def less(s: State): Boolean = s.count < 2
def lessEqual(s: State): Boolean = s.count <= 2
def greater(s: State): Boolean = s.count > 0
def greaterEqual(s: State): Boolean = s.count >= 0
def added(s: State): Int = s.count + 1
def subtracted(s: State): Int = s.count - 1
def contained(s: State): Boolean = listed(s).contains(s.flag)
def copied(s: State): State = s.copy(flag = true)
def minimum(s: State): Int = (s.count + 1).min(2)
def maximum(s: State): Int = (s.count - 1).max(0)
def concatenated(s: State): List[Boolean] = listed(s) ++ List(false)
def prepended(s: State): List[Boolean] = s.flag +: listed(s)
def converted(s: State): List[Boolean] = listed(s).toList
def contracted(s: State): Boolean =
  ({ require(s.count >= 0); s.flag }).ensuring(_ == true)
def interpolated(s: State): String = s"value=${true}"

object Checked extends Machine[State, Boolean, Boolean]:
  val init = State(false, UpTo(0))
  def end(s: State): Boolean =
    booleanLiteral(s) && integerLiteral(s) == 1 && textLiteral(s) == "ok" &&
      reference(s) == s && field(s) && called(s) && constructed(s).flag && wrapped(s).flag &&
      localValue(s) && conditional(s) && matched(s) && listed(s).contains(true) &&
      optional(s) == Some(true) && equal(s) && unequal(s) && conjoined(s) && disjoined(s) &&
      negated(s) && less(s) && lessEqual(s) && greater(s) && greaterEqual(s) &&
      added(s) == 1 && subtracted(s) == 0 && contained(s) && copied(s).flag &&
      minimum(s) == 2 && maximum(s) == 0 && concatenated(s).contains(false) &&
      prepended(s).contains(true) && converted(s).contains(true) && contracted(s) &&
      interpolated(s) == "value=true"
  object rules extends Bindings(press ~> stepRecord)
val predicate = Checked.property expect(check(after.state.flag))

object EmptyDraft extends Machine[State, Boolean, Boolean]:
  val init = State(false, UpTo(0))
  def end(s: State) = true
  object effects:
    val keep = effect {}
  object rules extends Bindings(press ~> effects.keep)
