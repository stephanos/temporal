package temporal.capabilities.conventionfixtures

import temporal.features.nexus.product.NexusProduct

val forbiddenFeature = NexusProduct.name

package level:
  import temporal.features.nexus.workflow.system.{Fact, Phase, State}
  import temporal.features.nexus.standalone.system.*
  val explicit = List(classOf[Phase], classOf[State], classOf[Fact])
  val wildcard = NexusSystem.name
