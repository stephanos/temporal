package temporal.capabilities.discoveryfixtures

import temporal.features.activity.mini.product.{ActivityProduct, Phase as ProductPhase}

object Discovery:
  val discoveredFeature = ActivityProduct.name
  val aliasedProductPhase = classOf[ProductPhase]
