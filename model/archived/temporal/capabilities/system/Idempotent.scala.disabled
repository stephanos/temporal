// Idempotent (fn-163): repeating a request an entity took is answered as the first was, a promise
// about an action rather than an entity. A system capability, named for its one use (the standalone
// Nexus operation's controls): a closed operation keeps its state and answers a control it is no
// longer open to as rejected, except one that repeats a request it took, answered OK. Its law lowers
// to the IR's Property; a machine reads it as its reading of Closable's closedIsRejectedUniformly.
package temporal.capabilities.system

import framework.dsl.*

object Idempotent:
  // Why a repeated request is answered OK even after close (operation.go RequestCancel, Terminate).
  val repeatedRequestsAnswer =
    "a repeated request id is answered OK after close: operation.go RequestCancel and Terminate"

  // A step from a closed state keeps it, and answers `rejected`, or `accepted` where it repeats a
  // request the entity took (`repeats` of the state before).
  def closedRejectsOrRepeats[S](m: Declares[S])(
      closed: S => Boolean,
      repeats: S => Boolean,
      rejected: m.Outcome,
      accepted: m.Outcome
  )(using CanEqual[S, S], CanEqual[m.Outcome, m.Outcome]): Property[S] =
    m.property holdsAcross ((before, after) =>
      !closed(before) ||
        (after.state == before && (after.outcome == rejected ||
          (after.outcome == accepted && repeats(before))))
    )
