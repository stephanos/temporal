package fixture.rulesshapes

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)

object hand extends Actor:
  val zero = action(this)
  val one = action(this).input[Boolean]("s")
  val two = action(this).inputs[Two]
  val three = action(this).inputs[Three]
  val four =
    action(this).inputs[Four]
  val five = action(this)
    .inputs[Five]




  val six = action(this)
    .inputs[Six]






object Arities extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def zero(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
    def one(s: Boolean, a: Boolean) = stay[Boolean, Boolean, Nothing](s && a)
    def two(s: Boolean, a: Boolean, b: Boolean) = stay[Boolean, Boolean, Nothing](s && a && b)
    def three(s: Boolean, a: Boolean, b: Boolean, c: Boolean) =
      stay[Boolean, Boolean, Nothing](s && a && b && c)
    def four(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean) =
      stay[Boolean, Boolean, Nothing](s && a && b && c && d)
    def five(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean) =
      stay[Boolean, Boolean, Nothing](s && a && b && c && d && e)
    def six(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean) =
      stay[Boolean, Boolean, Nothing](s && a && b && c && d && e && f)
  object rules extends Rules:
    on(hand.zero)(where(s => !s).where(s => s == false) ~> effects.zero)
    on(hand.one)(always ~> effects.one)
    on(hand.two)(always ~> ((s: Boolean, in: Two) => effects.two(s, in.s, in.state)))
    on(hand.three)(always ~> ((s: Boolean, in: Three) => effects.three(s, in.a, in.b, in.c)))
    on(hand.four)(always ~> ((s: Boolean, in: Four) => effects.four(s, in.a, in.b, in.c, in.d)))
    on(hand.five)(always ~> ((s: Boolean, in: Five) => effects.five(s, in.a, in.b, in.c, in.d, in.e)))
    on(hand.six)(always ~> ((s: Boolean, in: Six) => effects.six(s, in.a, in.b, in.c, in.d, in.e, in.f)))

enum Phase:
  case open
  case done extends Phase, Closed
final case class Snapshot(phase: Phase)
object PhaseRules extends Machine[Snapshot, Boolean, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = Snapshot(Phase.open)
  object states:
    def closed(p: Phase) = p == Phase.done
  object effects:
    def keep(s: Snapshot) = stay[Snapshot, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.zero):
      when(Phase.open) ~> effects.keep
      when(states.closed) ~> effects.keep
    on(hand.one)(when[Closed] ~> effects.keep)

// Native actions declare one complete record for several inputs (fn-165). Each record keeps the
// retired input chain's names, order and types, one IR input a field; the effects keep their flat
// helper parameters and the rules read the record's fields into them. The records sit last so
// every frozen position above keeps its line.
final case class Two(s: Boolean, state: Boolean)
final case class Three(a: Boolean, b: Boolean, c: Boolean)
final case class Four(a: Boolean, b: Boolean, c: Boolean, d: Boolean)
final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean)
final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)
