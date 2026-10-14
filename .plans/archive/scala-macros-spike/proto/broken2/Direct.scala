package samples

import framework.dsl.*
import framework.dsl.Finite.given

// The same refusals, reached by calling the reifier directly rather than through the inline `on`.
object Direct:
  final case class Counter(n: UpTo[3], on: Boolean)
  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)

  def press(state: Counter): List[Step[Counter, Outcome, Nothing]] =
    var next: Int = state.n
    next = next + 1
    enter(state.copy(n = UpTo(next)))

  def spin(state: Counter): List[Step[Counter, Outcome, Nothing]] =
    while state.on do ()
    enter(state)

  val pressIR = framework.reify.Reify.function(press)
  val spinIR = framework.reify.Reify.function((s: Counter) =>
    while s.on do ()
    enter(s)
  )
