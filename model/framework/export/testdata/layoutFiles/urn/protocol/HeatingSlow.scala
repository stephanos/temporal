package temporal.features.urn
package protocol

import framework.dsl.*

object HeatingSlow extends Derived(UrnProtocol.restrict(user.flip))
