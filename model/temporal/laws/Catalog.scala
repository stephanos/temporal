/* The Temporal catalog: the entity-neutral laws and the Temporal ones, the one catalog a Temporal
 * Model's capability declarations read.
 */
package temporal.laws

import umpire.laws.{entityNeutral, Catalog}
import umpire.laws.Capability.*

given catalog: Catalog = entityNeutral ++
  Catalog.single(Terminable)(terminateSettlesLaw) ++
  Catalog.single(Cancelable)(cancelIsRequestedLaw) ++
  Catalog.pair(Pausable, Pollable)(pausedIsNotDispatchedLaw)
