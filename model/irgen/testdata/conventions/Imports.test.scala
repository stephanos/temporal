package temporal.capabilities.discoveryfixtures

import temporal.features.nexus.product.{NexusProduct, Phase as ProductPhase}

object Discovery:
  val discoveredFeature = NexusProduct.name
  val aliasedProductPhase = classOf[ProductPhase]
