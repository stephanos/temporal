/* Terminate: what a terminate does to a live entity, wherever the entity is. */
package temporal.laws

import umpire.*
import umpire.laws.Law

/**
 * A terminate settles the entity in its own step: that step records `settled`. A functional law,
 * asked by a find from a live state the capability's `reach` gets to rather than verified over every
 * row: a terminate of a closed entity is Closable's to answer, and saying "from a live state" of
 * one action would need a transition Property restricted by `when`, which is not supported.
 */
def terminateSettles[S](m: Declares[S])(terminate: ClassRef, settled: m.Fact): Property[S] =
  m.property when terminate holds (_.records(settled))

val terminateSettlesLaw = Law(
  "terminateSettles",
  terminateSettles,
  cites = Seq(
    "chasm/lib/activity/activity.go",
    "chasm/lib/activity/statemachine.go",
    "chasm/lib/nexusoperation/operation_statemachine.go"
  ),
  promises = "a terminate of a live entity settles it as terminated in one step and records that",
  doesNotPromise =
    "what a terminate of a closed entity answers (closedIsRejectedUniformly), what a second " +
      "terminate answers (the activity answers the same request id OK, the Nexus operation also " +
      "refuses another id by name), or that a reason and an identity are recorded: the schedule " +
      "records only that it closed"
)
