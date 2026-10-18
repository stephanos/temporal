package fixture.bounds

import framework.dsl.*
import temporal.Bounds
import temporal.Bounds.{five, fiveWide, seven, three, twelve}

enum State:
  case waiting, done
enum Outcome:
  case ok
given Ok[Outcome] = Ok(Outcome.ok)
object client extends Actor:
  val finish = action(this)

final case class Steady[S](waiting: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]
object Steady extends CapabilityKind:
  def kept[S](m: Declares[S])(waiting: S => Boolean): Property[S] =
    m.property.across.expect(check(!waiting(before) || waiting(after.state)))
  def absent[S](m: Declares[S])(waiting: S => Boolean): Property[S] =
    m.property expect(check(!waiting(after.state)))

object Job extends Machine[State, Outcome, Nothing]:
  val init = State.waiting
  def end(s: State) = s == State.done
  object states:
    def waiting(s: State) = s == State.waiting
  object effects:
    def finish(s: State) = enter[State, Outcome, Nothing](State.done)
  object rules extends Rules:
    on(client.finish) {
      always ~> effects.finish
    }
  object properties:
    val done = property expect(check(after.state == State.done))
  object capabilities extends Capabilities:
    val steady: Capability = Steady(waiting = states.waiting)
  object queries:
    capabilities.bound(Bounds.six, Steady.kept[State] -> Bounds.two)
    val any = scenario.free
    val q2 = query verify properties.done in any limits Bounds.two
    val q3 = query verify properties.done in any limits three
    val q4 = query verify properties.done in any limits Bounds.four
    val q5 = query verify properties.done in any limits five
    val q6 = query verify properties.done in any limits Bounds.six
    val q7 = query verify properties.done in any limits seven
    val q8 = query verify properties.done in any limits Bounds.eight
    val q12 = query verify properties.done in any limits twelve
    val q4Wide = query verify properties.done in any limits Bounds.fourWide
    val q5Wide = query verify properties.done in any limits fiveWide
    val q12Wide = query verify properties.done in any limits Bounds.twelveWide
