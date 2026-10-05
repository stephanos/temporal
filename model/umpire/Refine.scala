package umpire

import scala.annotation.unused

/** A declared `refines:` line: the refined machine and the state map. */
final private[umpire] case class RefinementDecl[S](product: Model, map: S => Any)

/**
 * `refines: product` / `map: f`. The map's result type is the product's state type, so a map into
 * another machine's states does not compile.
 */
def refines[S, PS, PO, PF](using
    m: MachineScope[S, ?, ?]
)(product: Machine[PS, PO, PF])(f: S => PS): Unit =
  m.refinement = Some(RefinementDecl[S](product, f))

/**
 * A machine object's refinement, the section `object refinement extends Refinement(OrderProduct)`
 * of the machine that refines `OrderProduct`: `toProduct`, the map from this machine's states to the
 * refined machine's, which a map into another machine's states does not implement; where declared,
 * `visible`, the facts the refined machine sees, `visibleOutcomes`, the outcomes it sees, and
 * `unobservable`, the timers whose step records nothing a Run can read. A machine refines at most
 * one machine. The IR generator reads it from the source.
 */
abstract class Refinement[S, P](using @unused owner: Owner[S, ?, ?])(val of: Machine[P, ?, ?])
    extends Section:
  /** The state of the refined machine a state of this one reads as. */
  def toProduct(s: S): P
