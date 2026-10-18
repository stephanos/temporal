package fixture.rulesrefusals

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)

object UnusedNil extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def unused(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = Nil
  object rules extends Rules:
    disabled(hand.go)

object UsedDisabled extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def empty(@scala.annotation.unused s: Boolean): List[Step[Boolean, Boolean, Nothing]] = disabled
  object rules extends Rules:
    on(hand.go)(always ~> effects.empty)

object UnusedList extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def unused(@scala.annotation.unused s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List()
  object rules extends Rules:
    disabled(hand.go)

object MatchFirst extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def unused(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = s match
      case false => Nil
      case true  => stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    disabled(hand.go)

object MatchLast extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def unused(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = s match
      case false => stay[Boolean, Boolean, Nothing](s)
      case true  => Nil
  object rules extends Rules:
    disabled(hand.go)

object RequireGuard extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.go)(where { (s: Boolean) => require(s); s } ~> effects.keep)

object EnsureGuard extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.go)(where((s: Boolean) => s.ensuring((result: Boolean) => result == s)) ~> effects.keep)

object RequireEffect extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.go)(always ~> ((s: Boolean) =>
      require(s); effects.keep(s)
    ))

object EnsureEffect extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.go)(always ~> ((s: Boolean) => effects.keep(s).ensuring(_.nonEmpty)))

object Exhausted extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(collision.go)(always ~> effects.keep)
object collision extends Actor:
  val go = action(this).inputs[Exhausting]
// A native record (fn-165) with the retired chain's inputs, which exhaust the state names.
final case class Exhausting(s: Boolean, state: Boolean, s0: Boolean)

object HelperResult extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  def helper(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = Nil
  object effects:
    def unused(s: Boolean) = helper(s)
  object rules extends Rules:
    disabled(hand.go)
