package framework.dsl

import propertygroups.*
import propertygroups.features.specimen.workflow.*
import propertygroups.foundations.lonely.Lonely
import propertygroups.foundations.bundleonly.BundleOnly

object PropertyGroupsNative:
  def main(args: Array[String]): Unit =
    NativeChecks.check(
      List(
        Subject.properties.safety.closed,
        Subject.properties.safety.aliased,
        Subject.properties.safety.bundle.open,
        Subject.properties.safety.bundle.closed
      ),
      Subject.properties.safety.stable,
      List(
        Subject.properties.liveness.reaches,
        Subject.properties.liveness.factory,
        Lonely.properties.liveness.selectedProgress,
        BundleOnly.properties.liveness.bundle.progress
      ),
      Subject,
      Lonely,
      BundleOnly,
      Subject.queries.finished,
      Subject.monitors.watched
    )
    assert(Subject.properties.safety.watched eq Subject.monitors.watched)
    assert(Subject.properties.safety.selected eq Subject.properties.safety.bundle.closed)
    assert(Subject.properties.safety.repeated eq Subject.properties.safety.aliased)
    val predicate = Joined.properties.safety.member.decl.holds.get
      .asInstanceOf[Step[Pair, String, String] => Boolean]
    assert(
      Vector(State.open, State.closed).map(s => predicate(Step("done", Pair(s)))) ==
        Vector(false, true)
    )
    println("PROPERTY_GROUPS_NATIVE_OK predicateRows=30 monitorRows=8 properties=6 progress=4")
