package framework.headernative

import java.util.Base64
import framework.dsl.*
import fixture.constructedheader.*
import io.temporal.server.api.umpire.v1 as ir

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
    val ordinary = Vector(Named, Aliased, BooleanMachine, Replaced, OpaqueOutcome, InboxMachine)
      .map(MachineHeaders.collect(_))
    val failure = try
      val unexpected = MachineHeaders.collect(NestedMachine)
      throw AssertionError(s"nested ownership was guessed: $unexpected")
    catch case error: IllegalArgumentException => error.getMessage
    assert(failure.contains("nestedMachine") && failure.contains("Held.box"), failure)
    val nested = MachineHeaders.collect(
      NestedMachine,
      Seq(fixture.constructedheader.held -> "fixture.constructedheader.Held")
    )
    val all = ordinary :+ nested
    val types = all.flatMap(_.types).groupBy(_.name).toVector.sortBy(_._1).map { (_, copies) =>
      assert(copies.distinct.size == 1)
      copies.head
    }
    val model = ir.Model(machines = all.map(projected).sortBy(_.name), types = types)
    println("headers:" + Base64.getEncoder.encodeToString(model.toByteArray))
    val scalar = MachineHeaders.collect(Scalar)
    assert(
      scalar.header.stateType.isEmpty && scalar.header.outcomeType.isEmpty && scalar.header.factType.isEmpty
    )
    val opaque = MachineHeaders.collect(Opaque)
    assert(opaque.header.stateType.isEmpty && opaque.header.outcomeType.isEmpty)
    assert(Opaque.fs eq Opaque.fo)
    assert(Opaque.fs eq Domains.shared)
    assert(
      Opaque.fs.schemaFor("fixture.constructedheader.Domains.Left").get.root.getIntRange.high == 2
    )
    assert(
      Opaque.fo.schemaFor("fixture.constructedheader.Domains.Right").get.root.getIntRange.high == 2
    )
    assert(Named.fs eq Replaced.fs)
    assert(MachineHeaders.collect(Named) == ordinary.head)
    println("native-controls:passed")
