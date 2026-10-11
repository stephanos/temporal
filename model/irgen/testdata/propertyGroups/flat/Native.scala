package framework

import propertygroups.*
import propertygroups.features.specimen.workflow.*
import propertygroups.foundations.lonely.Lonely
import propertygroups.foundations.bundleonly.BundleOnly

object PropertyGroupsNative:
  def main(args: Array[String]): Unit =
    NativeChecks.check(
      List(
        Subject.properties.closed,
        Subject.properties.aliased,
        Subject.properties.bundle.open,
        Subject.properties.bundle.closed
      ),
      Subject.properties.stable,
      List(
        Subject.properties.reaches,
        Subject.properties.factory,
        Lonely.properties.selectedProgress,
        BundleOnly.properties.bundle.progress
      ),
      Subject,
      Lonely,
      BundleOnly,
      Subject.queries.finished,
      Subject.monitors.watched
    )
    assert(Subject.properties.watched eq Subject.monitors.watched)
    assert(Subject.properties.selected eq Subject.properties.bundle.closed)
    assert(Subject.properties.repeated eq Subject.properties.aliased)
    val predicate = Joined.properties.member.decl.holds.get
      .asInstanceOf[Step[Pair, String, String] => Boolean]
    assert(
      Vector(State.open, State.closed).map(s => predicate(Step("done", Pair(s)))) ==
        Vector(false, true)
    )
    println("PROPERTY_GROUPS_NATIVE_OK predicateRows=30 monitorRows=8 properties=6 progress=4")
