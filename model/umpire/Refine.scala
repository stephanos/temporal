package umpire

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
