package temporal.capabilities.discoveryfixtures

import temporal.features.nexus.shared.product.{NexusProduct, Phase as ProductPhase}

object Discovery:
  val discoveredFeature = NexusProduct.name
  val aliasedProductPhase = classOf[ProductPhase]
