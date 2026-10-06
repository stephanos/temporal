// The Temporal kit's sugar: definitions whose meaning a core realization declaration already
// expresses, kept for readability. Each names the core form it stands for, and the IR
// generator (model/irgen/Syntax.scala) lowers it to the IR that core form lifts to. No core file
// of the kit uses them.
package temporal.realize

import umpire.Slot
import umpire.realize.{Field, RequestScope, TypedOperand}

// A field of the request of the scope it is written in, the named slot `:=` gives an operand.
// Core form: the typed field `Field[Req, V](_.name)` that `Assignment.typed` takes.
final class RequestField[Req, V] private[realize] (val target: Field[Req, V])
    extends Slot[TypedOperand[V]]

// `field(_.namespace) := operand`, written inside `rpc(…) { … }` or `readUntil(…) { … }`: the
// field of the call's request the selector names receives the operand. The request type is the
// scope's, so the selector is typed against it and no line repeats it; `:=` is the named slot's one
// operator, as for an action's input. Core form:
// `Assignment.typed(Field[Req, V](_.namespace), operand)`.
def field[Req, V](using RequestScope[Req])(select: Req => V) =
  RequestField(Field(select))
