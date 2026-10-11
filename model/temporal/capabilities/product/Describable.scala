// The realization's table from each recorded fact to the status a description reads.
package temporal.capabilities.product

import framework.*
import framework.realize.StatusTable

final case class Describable[S, V](statusTable: StatusTable[V])
    extends CapabilityOf[S, Nothing, Nothing]

object Describable extends CapabilityKind
