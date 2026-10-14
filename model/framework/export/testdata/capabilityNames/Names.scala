package fixture.capabilitynames

import framework.dsl.*
import framework.dsl.Finite.given
import temporal.Bounds.three
import temporal.capabilities.system.{Expirable, Retryable}

enum Phase:
  case waiting extends Phase, Waiting
  case failed extends Phase, Failed
  case expired extends Phase, TimedOut
enum Fact:
  case timeout
enum Answer:
  case ok
given Ok[Answer] = Ok(Answer.ok)
final case class State(phase: Phase, attempts: UpTo[1], armed: Boolean)

object worker extends Actor:
  val fail = action(this)
val deadline = timer

object Record extends Machine[State, Answer, Fact], Phased[State, Phase](_.phase):
  val init = State(Phase.waiting, UpTo[1](0), true)
  object states:
    def attemptCount(s: State): Int = s.attempts
    def maximumAttempts(s: State): Option[UpTo[1]] = Some(s.attempts)
    def remaining(s: State): Boolean = s.attempts < 1
    def armed(s: State): Boolean = s.armed
  object effects:
    def fail(s: State): List[Step[State, Answer, Fact]] = enter(s.copy(phase = Phase.failed))
    def expire(s: State): List[Step[State, Answer, Fact]] =
      enter(s.copy(phase = Phase.expired), Fact.timeout)
  object rules extends Bindings(worker.fail ~> effects.fail, deadline ~> effects.expire)
  object capabilities extends Capabilities:
    val retryable: Capability = Retryable[State, Phase, 1](
      worker.fail,
      true,
      states.attemptCount,
      states.maximumAttempts,
      states.remaining
    )
    val scheduleDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
      deadline,
      states.armed,
      Fact.timeout,
      Seq(Fact.timeout)
    )
  object queries:
    capabilities.bound(three)
