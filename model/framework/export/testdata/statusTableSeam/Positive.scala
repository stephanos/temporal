package temporal.features.statusfixture.protocol

import framework.dsl.*
import temporal.capabilities.statusfixture.Observable

object Positive extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
  object capabilities extends Capabilities:
    val observable: Capability =
      Observable(statusTable = temporal.features.statusfixture.realization.statuses)
