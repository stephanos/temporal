/* The Temporal catalog: every law of this folder, keyed by the capability kinds that bring it, the
 * one catalog a Temporal Model's capability declarations read.
 */
package temporal.capabilities

import umpire.Catalog

given catalog: Catalog =
  Catalog.single(Closable)(terminalStatesAreFinal, closedIsRejectedUniformly) ++
    Catalog.single(Terminable)(terminateSettles) ++
    Catalog.single(Cancelable)(cancelIsRequested) ++
    Catalog.pair(Pausable, Pollable)(pausedIsNotDispatched)
