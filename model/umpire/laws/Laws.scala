/* The entity-neutral laws: what any entity with a terminal status set is held to, whatever it is.
 *
 * Each law is an object named after the law: its `apply` takes the model and the capability's fields
 * and returns a Property of that model, and its `Law` arguments say what it promises, what it leaves
 * to other laws and which server code it rests on. The Property takes its name from the val that
 * declares the call (`val terminalStays = terminalStatesAreFinal(m)(…)`), or under a capability
 * declaration `<machine>.<law>`. The framework's files name no sugar, so each body is the core form
 * of the claim pattern its scaladoc names (model/umpire/Syntax.scala).
 */
package umpire.laws

import umpire.{Declares, Property}

/**
 * No step leaves the terminal set: from a state whose `status` is `terminal`, every step keeps the
 * `status`. The core form of `once(s => terminal(status(s))).keeps(status)`.
 */
object terminalStatesAreFinal
    extends Law(
      cites = Seq(
        "chasm/lib/activity/statemachine.go",
        "chasm/lib/nexusoperation/operation_statemachine.go"
      ),
      promises =
        "no step moves an entity out of a terminal status, or from one terminal status to another",
      doesNotPromise =
        "that a closed entity answers a mutation by rejecting it (closedIsRejectedUniformly), that " +
          "its status is reported, or that the close is recorded once"
    ):
  def apply[S, P](m: Declares[S])(status: S => P, terminal: P => Boolean): Property[S] =
    m.property holdsAcross ((before, after) =>
      !terminal(status(before)) || status(after.state) == status(before)
    )

/**
 * A closed entity is rejected alike by every step: from a state whose `status` is `terminal`, a step
 * keeps the state and answers `rejected`. Phrased over every step, because a transition Property
 * restricted by `when` is not supported (model/SEMANTICS.md, Claims).
 */
object closedIsRejectedUniformly
    extends Law(
      cites = Seq(
        "chasm/lib/activity/activity.go",
        "chasm/lib/nexusoperation/operation.go",
        "chasm/lib/scheduler/scheduler.go"
      ),
      promises =
        "every step from a terminal status keeps the state and answers the entity's rejecting outcome",
      doesNotPromise =
        "which outcome rejects (a parameter: the activity answers NotFound, the Nexus operation and " +
          "the schedule FailedPrecondition), or that a repeated request with the same request id " +
          "is rejected: the server answers it as the first"
    ):
  def apply[S, P](m: Declares[S])(
      status: S => P,
      terminal: P => Boolean,
      rejected: m.Outcome
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !terminal(status(before)) || (after.state == before && after.outcome == rejected)
    )
