package temporal.features.statusfixture.realization

val statuses = framework.realize.statusTable[String]()
def makeHolder() = temporal.capabilities.statusfixture.Holder(statuses)
