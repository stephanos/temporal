// Sugar whose types cross, which its signature refuses before anything is lifted: the lifter's
// tests build this and expect a type error at each marked line.
package fixture.crossed

import umpire.*

given Accepted[Outcome] = Accepted(Outcome.accepted)

// `implies` of a left side that is no Boolean.
val notBoolean: Boolean = Signal.up implies true

// `records` of a fact of another type than the step's.
def foreignFact(after: Step[Here, Outcome, Note]): Boolean = after.records(Signal.up)

// `accept` of a fact of another type than the step function's.
def acceptForeign(h: Here): List[Step[Here, Outcome, Note]] = accept(h, Signal.up)

// `stay` in a state of another type than the step function's.
def stayElsewhere(h: Here): List[Step[Here, Outcome, Note]] = stay(There(h.on))

// `sticky` of a promise about the steps of another state type than the monitor's.
def thereOn(after: Step[There, Outcome, Note]): Boolean = after.state.on
val stickyElsewhere: Monitor[Here, Outcome, Note, Boolean] = sticky(thereOn)
