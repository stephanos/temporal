// Cancel: a cancel is a request, recorded on the entity while its work is in flight.
package temporal.capabilities

import umpire.*

// A cancel request is recorded in its own step: that step records `requested`. A functional law,
// asked by a find from a live state the capability's `reach` gets to, for the reason
// `terminateSettles` gives.
object cancelIsRequested
    extends Law(
      cites = Seq(
        "chasm/lib/activity/operator_commands.go",
        "chasm/lib/activity/statemachine.go",
        "chasm/lib/nexusoperation/operation.go"
      ),
      promises = "a cancel request of an entity whose work is in flight records the request and " +
        "leaves the entity live",
      doesNotPromise = "that the entity is then canceled, which only its work's answer settles; what a cancel with " +
        "no work in flight does (the activity settles canceled at once); or what a second request " +
        "answers (the Nexus operation answers ErrCancellationAlreadyRequested)"
    ):
  def apply[S](m: Declares[S])(requestCancel: ClassRef, requested: m.Fact): Property[S] =
    m.property when requestCancel holds (_.records(requested))
