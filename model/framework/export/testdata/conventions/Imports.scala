package temporal.capabilities.conventionfixtures

import temporal.features.activity.mini.product.ActivityProduct

val forbiddenFeature = ActivityProduct.name

package level:
  import temporal.features.activity.mini.product.{Fact, Phase, State}
  import temporal.features.activity.mini.protocol.*
  val explicit = List(classOf[Phase], classOf[State], classOf[Fact])
  val wildcard = ActivityProtocol.name
