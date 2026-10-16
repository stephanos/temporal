package framework.finiteschemanative

import java.util.Base64
import framework.dsl.*
import framework.dsl.Finite.given
import fixture.finiteschemaroles.*
import io.temporal.server.api.umpire.v1 as ir

// A values-only witness. Under the reify plugin an anonymous `new Finite[T]:` does not compile:
// the plugin retains a schema on every concrete Finite construction, its parent call included.
final class Unread[T](message: String)(using Data.Structural[T]) extends Finite[T]:
  def values = throw AssertionError(message)

object Native:
  private def projected(value: MachineHeaders.Contribution): ir.Machine =
    val h = value.header
    ir.Machine(
      family = h.family,
      name = h.name,
      stateType = h.stateType,
      outcomeType = h.outcomeType,
      factType = h.factType
    )
  def main(args: Array[String]): Unit =
    val all = Vector(Default, Explicit, Replaced, PlainMachine, OpenOverride, FieldSchema)
      .map(MachineHeaders.collect(_))
    val schemas =
      Vector(Finite[Phase], Finite[Plain], Finite[Snapshot], Finite[OpenPhase], Finite[FieldPhase])
        .map(_.schema.get)
    val types = (all.flatMap(_.types) ++ schemas.flatMap(_.types))
      .groupBy(_.name)
      .toVector
      .sortBy(_._1)
      .map { (_, copies) =>
        assert(copies.distinct.size == 1)
        copies.head
      }
    val model = ir.Model(machines = all.map(projected).sortBy(_.name), types = types)
    val phases = Finite[Phase].values
    assert(phases == IndexedSeq(Phase.plain, Phase.waiting, Phase.done, Phase.failed))
    assert(phases(2) eq Phase.done)
    assert(Phase.done.in[Settled] && Phase.done.in[Succeeded] && Phase.done.in[Closed])
    assert(Phase.waiting.in[Waiting] && Phase.waiting.in[Live])
    assert(Phase.failed.in[Failed] && !Phase.failed.in[Live])
    assert(Finite[Done].values.head eq Phase.done)
    assert(Finite[Plain.first.type].values.head eq Plain.first)
    assert(Finite[Alias].schema.get == Finite[Snapshot].schema.get)
    assert(Default.fs eq Replaced.fs)
    val restricted = Finite.of[Phase](Phase.failed, Phase.done)
    assert(restricted.values == IndexedSeq(Phase.failed, Phase.done))
    assert(restricted.schema.get == Finite[Phase].schema.get)
    val empty = Finite.of[Phase]()
    assert(empty.values.isEmpty && empty.schema.get == restricted.schema.get)
    val throwing: Finite[Phase] = Unread[Phase]("schema read enumerated role catalog")
    assert(throwing.schema.get == restricted.schema.get)
    var reads = 0
    lazy val lazyWitness: Finite[Phase] =
      reads += 1
      Finite.of[Phase](Phase.done)
    assert(reads == 0)
    assert(lazyWitness.schema.get == restricted.schema.get && reads == 1)
    assert(lazyWitness.values.head eq Phase.done)
    assert(Default.end(Snapshot(Inner(Phase.done, false))))
    assert(Default.end(Snapshot(Inner(Phase.failed, false))))
    assert(!Default.end(Snapshot(Inner(Phase.waiting, false))))
    assert(Explicit.end(Snapshot(Inner(Phase.waiting, true))))
    assert(!Explicit.end(Snapshot(Inner(Phase.done, false))))
    assert(Replaced.end(Snapshot(Inner(Phase.done, false))))
    assert(OpenOverride.end(OpenSnapshot(OpenPhase.waiting)))
    val missing = try
      val accepted = OpenDefault.end(OpenSnapshot(OpenPhase.waiting))
      throw AssertionError(s"missing Closed set was accepted: $accepted")
    catch case error: IllegalArgumentException => error.getMessage
    assert(missing.contains("openDefault") && missing.contains("has no Closed case"), missing)
    assert(
      Finite[FieldPhase].values == IndexedSeq(
        FieldPhase.waiting,
        FieldPhase.done(false),
        FieldPhase.done(true)
      )
    )
    assert(FieldDefault.end(FieldSnapshot(FieldPhase.done(false))))
    assert(FieldConsumer.end(FieldSnapshot(FieldPhase.done(true))))
    println("roles:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println("role-controls:catalogs,identity,aliases,lazy,default,explicit,derived,consumer-stages")
