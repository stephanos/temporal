package temporal.capabilities.statusfixture

import framework.dsl.{CapabilityKind, CapabilityOf}
import framework.realize.StatusTable

final case class Observable(statusTable: StatusTable[String])
    extends CapabilityOf[Boolean, Boolean, Nothing]
object Observable extends CapabilityKind
final case class Extra(statusTable: StatusTable[String], count: Int)
    extends CapabilityOf[Boolean, Boolean, Nothing]
object Extra extends CapabilityKind
final case class Holder(statusTable: StatusTable[String])
