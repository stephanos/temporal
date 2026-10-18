package framework.dsl

import scala.annotation.unused

// A machine object's refinement, the section `object refinement extends Refinement(OrderProduct)`
// of the machine that refines `OrderProduct`: `toProduct`, the map from this machine's states to the
// refined machine's, which a map into another machine's states does not implement; where declared,
// `visible`, the facts the refined machine sees, `visibleOutcomes`, the outcomes it sees, and
// `unobservable`, the timers whose step records nothing a Run can read. A machine refines at most
// one machine. The IR generator reads it from the source.
abstract class Refinement[S, P](using @unused owner: Owner[S, ?, ?])(
    val of: RecordMachine[P, ?, ?]
):
  // The state of the refined machine a state of this one reads as. Of a Machine that declares its
  // state as vars it reads them by name, `def toProduct(state: State) = Consensus.init.copy(chosen =
  // chosen(current))`: it is evaluated with the state restored into them (`image`).
  def toProduct(s: S): P

  // The image of `s` in the refined machine: `toProduct` evaluated with `s` restored into the vars
  // of a Machine that declares them.
  final def image(s: S): P = s match
    case state: State => state.owner.at(state)(NativeRuntime.installed.invoke(toProduct, s))
    case _            => NativeRuntime.installed.invoke(toProduct, s)

object Refinement:
  // The machine `machine`'s `object refinement` refines, where it declares one, for the gate's
  // construction of what an IR file lifts (IrFile.construct). The section is a member object a
  // machine need not declare, so the base class names no member for it; it is found as the
  // object's nested module, `<Machine>$refinement$`, which initializes it. The machine's own
  // wiring, its `rules`, reads no section this way.
  private[framework] def declaredBy(machine: RecordMachine[?, ?, ?]): Option[Refinement[?, ?]] =
    val declaration =
      try Some(java.lang.Class.forName(machine.getClass.getName + "refinement$"))
      catch case _: ClassNotFoundException => None
    declaration
      .map(c => c.getField("MODULE$").get(null)) // scalafix:ok DisableSyntax.null
      .map:
        case refinement: Refinement[?, ?] => refinement
        case other                        =>
          throw IllegalStateException(s"${machine.name}.refinement is not a Refinement: $other")
