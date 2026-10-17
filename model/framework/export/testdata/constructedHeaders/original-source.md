# Independent header oracle

Captured before production edits at `7a769a651e29566b33584eb36536dc56117a2976`.
Whole raw output and compiled original source/jar/classpath remain under
`.flow/tmp/constructed-headers.11348373142731438684`; `original.json` copies that output.
Named, alias, Boolean, Derived, direct Inbox, nested record and named-state opaque-outcome roots
were admitted. UpTo and independent opaque states were refused as neither enum nor case class;
the shared-witness opaque outcome was refused for lacking a readable range declaration.
These three remain qualified native controls. Earlier failed discovery sources/logs remain under
`.flow/tmp/constructed-headers.13346461991610003494` and
`.flow/tmp/constructed-headers.14719426133380081055`.

```scala
package fixture.constructedheader

import framework.dsl.*

enum Mode:
  case ready, stopped
final case class Inner(mode: Mode, flag: Boolean)
final case class State(inner: Inner)
enum Outcome:
  case accepted, refused
enum Fact:
  case changed
type Alias = State

object Named extends Machine[State, Outcome, Fact]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
object Aliased extends Machine[Alias, Outcome, Nothing]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
object Scalar extends Machine[UpTo[2], Boolean, Nothing]:
  val init = UpTo[2](0)
  def end(s: State) = true
  object rules extends Bindings()
object BooleanMachine extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: State) = true
  object rules extends Bindings()
object Replaced extends Derived(Named.unmonitored)

object Domains:
  opaque type Left = Int
  opaque type Right = Int
  given left: Finite[Left] = Finite.upTo(2)
  given right: Finite[Right] = left
  val zero: Left = 0
object Opaque extends Machine[Domains.Left, Domains.Right, Nothing]:
  val init = Domains.zero
  def end(s: State) = true
  object rules extends Bindings()

object Independent:
  opaque type Left = Int
  opaque type Right = Int
  given left: Finite[Left] = Finite.upTo(2)
  given right: Finite[Right] = Finite.upTo(2)
  val zero: Left = 0
object OpaqueSeparate extends Machine[Independent.Left, Independent.Right, Nothing]:
  val init = Independent.zero
  def end(s: State) = true
  object rules extends Bindings()
object OpaqueOutcome extends Machine[State, Independent.Right, Nothing]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()

enum Message:
  case first, second
val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Held(box: Inbox[Message], flag: Boolean)
given held: Finite[Held] =
  given Finite[Inbox[Message]] = wire.contents
  Finite.derived
final case class Nested(held: Held)
object InboxMachine extends Machine[Held, Outcome, Nothing]:
  val init = Held(wire.empty, false)
  def end(s: State) = true
  object rules extends Bindings()
object NestedMachine extends Machine[Nested, Outcome, Nothing]:
  val init = Nested(Held(wire.empty, false))
  def end(s: State) = true
  object rules extends Bindings()
```

## Native re-anchoring (fn-174)

Native state refuses opaque state types at compile time, so `Opaque` and `OpaqueSeparate` moved
from `Headers.scala` to `Refused.scala`, where the suite asserts the compiler refusal at each
declaration instead of the old lift refusal. `original.json` never held either root and is
unchanged. `Scalar` keeps its direct `UpTo[2]` state, authored without explicit evidence
since UpTo's catalog moved to `Finite`'s companion and no longer competes with Data's structural
given for `Data[UpTo[N]]`. The reader now reads this state as a bounded counter and admits it, as it
admits the Boolean state, with no named header type; the suite asserts that admission where it
asserted the old refusal. The shared opaque witness checks moved to
`Throwing` (`Selection.scala`), whose outcome and facts are the two opaque types; Timing asserts
them. Native `Built` construction now defers reading its start (fn-123.10), so Timing reads each
built start once and keeps its `native-init=2` count.
