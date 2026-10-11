package temporal.features.statusfixture.protocol

import framework.*
import temporal.capabilities.statusfixture.{Extra, Observable}

object Negative extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
  val wrongSection = Observable(
    temporal.features.statusfixture.realization.statuses
  )
  object capabilities extends Capabilities:
    val extra: Capability =
      Extra(temporal.features.statusfixture.realization.statuses, 1)
    val hidden: Capability = Observable {
      val table = temporal.features.statusfixture.realization.statuses; table
    }
    val call: Capability = Observable(
      identity(temporal.features.statusfixture.realization.statuses)
    )
    val qualifier: Capability = Observable(
      temporal.features.statusfixture.realization.makeHolder().statusTable
    )
    val unrelated: Capability =
      val _ =
        io.temporal.api.workflowservice.v1.StartActivityExecutionRequest.defaultInstance;
      Observable(temporal.features.statusfixture.realization.statuses)
  object impostor:
    object capabilities:
      val fake = Observable(
        temporal.features.statusfixture.realization.statuses
      )
