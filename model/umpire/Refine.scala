package umpire

import scala.annotation.unused

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

object Refinement:
  /**
   * The machine `machine`'s `object refinement` refines, where it declares one, for the gate's
   * construction of what an IR file lifts (IrFile.construct). The section is a member object a
   * machine need not declare, so the base class names no member for it; it is found as the
   * object's nested module, `<Machine>$refinement$`, which initializes it. The machine's own
   * wiring, its `rules`, reads no section this way.
   */
  private[umpire] def declaredBy(machine: Machine[?, ?, ?]): Option[Model] =
    scala.util
      .Try(java.lang.Class.forName(machine.getClass.getName + "refinement$"))
      .flatMap(c =>
        scala.util.Try(c.getField("MODULE$").get(null))
      ) // scalafix:ok DisableSyntax.null
      .toOption
      .collect { case r: Refinement[?, ?] => r.of }
