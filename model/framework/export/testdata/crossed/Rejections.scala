// `rejects` where it does not compile: in a machine whose outcome is its own rather than the shared
// one, and explained twice. The lifter's tests build this and expect a type error at each marked
// line.
package fixture.crossed

import framework.*
import framework.outcomes.Rejection

object knocker extends Actor:
  val knock = action(this)

// `rejects` in a machine whose outcome is its own.
object OwnOutcome extends Machine[Here, Outcome, Nothing]:
  val init = Here(false)
  def end(s: Here) = true
  object rules extends Rules:
    on(knocker.knock)(always ~> rejects(Rejection.notFound))

// `because` of a rejection that is explained already.
object ExplainedTwice extends Machine[Here, outcomes.Outcome, Nothing]:
  val init = Here(false)
  def end(s: Here) = true
  object rules extends Rules:
    on(knocker.knock)(always ~> rejects(Rejection.notFound).because("gone").because("really"))
