package framework

import scala.annotation.unused

// A machine object's refinement, the section `object refinement extends Refinement(OrderProduct)`
// of the machine that refines `OrderProduct`: `toProduct`, the map from this machine's states to the
// refined machine's, which a map into another machine's states does not implement; where declared,
// `visible`, the facts the refined machine sees, `visibleOutcomes`, the outcomes it sees, and
// `unobservable`, the timers whose step records nothing a Run can read. A machine refines at most
// one machine. The IR generator reads it from the source.
abstract class Refinement[S, P](using @unused owner: Owner[S, ?, ?])(val of: Machine[P, ?, ?]):
  // The state of the refined machine a state of this one reads as.
  def toProduct(s: S): P

object Refinement:
  // The states of the refining machine whose `Closed` role differs from that of their image under
  // `refinement`, each with its image, over every value of `S`: the refinement keeps closedness
  // where the result is empty. Roles are read through each machine's phase projection, `phase` and
  // `productPhase` (model/framework/Roles.scala). The IR holds no roles, so the model tests call it for
  // each refinement whose machines' phases both carry roles, naming the refinement.
  def unclosed[S, P](refinement: Refinement[S, P])(phase: S => Any, productPhase: P => Any)(using
      states: Finite[S]
  ): IndexedSeq[(S, P)] =
    def closed(p: Any) = p match
      case _: Closed => true
      case _         => false
    states.values
      .map(s => s -> refinement.toProduct(s))
      .filter((s, image) => closed(phase(s)) != closed(productPhase(image)))

  // The machine `machine`'s `object refinement` refines, where it declares one, for the gate's
  // construction of what an IR file lifts (IrFile.construct). The section is a member object a
  // machine need not declare, so the base class names no member for it; it is found as the
  // object's nested module, `<Machine>$refinement$`, which initializes it. The machine's own
  // wiring, its `rules`, reads no section this way.
  private[framework] def declaredBy(machine: Machine[?, ?, ?]): Option[Model] =
    scala.util
      .Try(java.lang.Class.forName(machine.getClass.getName + "refinement$"))
      .flatMap(c =>
        scala.util.Try(c.getField("MODULE$").get(null))
      ) // scalafix:ok DisableSyntax.null
      .toOption
      .collect { case r: Refinement[?, ?] => r.of }
