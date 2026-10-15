package fixture.finiteschemaroles

import framework.dsl.*

trait Settled extends Succeeded
enum Phase:
  case plain
  case waiting extends Phase, Waiting
  case done extends Phase, Settled
  case failed extends Phase, Failed
enum Plain:
  case first, second
enum Outcome:
  case accepted
final case class Inner(phase: Phase, flag: Boolean)
final case class Snapshot(inner: Inner)
type Alias = Snapshot
type Done = Phase.done.type

object Default extends Machine[Snapshot, Outcome, Nothing], Phased[Snapshot, Phase](_.inner.phase):
  val init = Snapshot(Inner(Phase.waiting, false))
  object rules extends Bindings()
object Explicit extends Machine[Alias, Outcome, Nothing], Phased[Alias, Phase](_.inner.phase):
  val init = Snapshot(Inner(Phase.waiting, false))
  override def end(s: State) = s.inner.flag
  object rules extends Bindings()
object Replaced extends Derived(Default.unmonitored)
object PlainMachine extends Machine[Plain, Outcome, Nothing]:
  val init = Plain.first
  def end(s: State) = true
  object rules extends Bindings()

enum OpenPhase:
  case waiting extends OpenPhase, Waiting
final case class OpenSnapshot(phase: OpenPhase)
object OpenOverride
    extends Machine[OpenSnapshot, Outcome, Nothing],
      Phased[OpenSnapshot, OpenPhase](_.phase):
  val init = OpenSnapshot(OpenPhase.waiting)
  override def end(s: State) = true
  object rules extends Bindings()
abstract class OpenDefaultSupport
    extends Machine[OpenSnapshot, Outcome, Nothing],
      Phased[OpenSnapshot, OpenPhase](_.phase):
  val init = OpenSnapshot(OpenPhase.waiting)
  object rules extends Bindings()

object OpenDefault extends OpenDefaultSupport

enum FieldPhase:
  case waiting extends FieldPhase, Waiting
  case done(flag: Boolean) extends FieldPhase, Closed
final case class FieldSnapshot(phase: FieldPhase)
object FieldSchema extends Machine[FieldSnapshot, Outcome, Nothing]:
  val init = FieldSnapshot(FieldPhase.waiting)
  def end(s: State) = true
  object rules extends Bindings()
abstract class FieldDefaultSupport
    extends Machine[FieldSnapshot, Outcome, Nothing],
      Phased[FieldSnapshot, FieldPhase](_.phase):
  val init = FieldSnapshot(FieldPhase.waiting)
  object rules extends Bindings()
object FieldDefault extends FieldDefaultSupport
abstract class FieldConsumerSupport extends Machine[FieldSnapshot, Outcome, Nothing]:
  val init = FieldSnapshot(FieldPhase.waiting)
  def end(s: State) = s.phase.in[Closed]
  object rules extends Bindings()

object FieldConsumer extends FieldConsumerSupport
