package temporal.features.urn
package protocol

import framework.dsl.*

object BoilingOver extends Derived(UrnProtocol.restrict(user.flip))
