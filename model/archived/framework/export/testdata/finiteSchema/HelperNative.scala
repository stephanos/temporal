package framework.finiteschemahelpernative

import java.util.Base64
import framework.dsl.{Finite, MachineHeaders}
import fixture.finiteschemaclient.*
import fixture.finiteschemahelper.Helper
import fixture.finiteschemaroles.{Phase, Snapshot}
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val all = Vector(Inherited, DerivedInherited).map(MachineHeaders.collect(_))
    val machines = all
      .map { value =>
        val h = value.header
        ir.Machine(
          family = h.family,
          name = h.name,
          stateType = h.stateType,
          outcomeType = h.outcomeType,
          factType = h.factType
        )
      }
      .sortBy(_.name)
    val types = all.flatMap(_.types).groupBy(_.name).toVector.sortBy(_._1).map { (_, copies) =>
      assert(copies.distinct.size == 1)
      copies.head
    }
    assert(Helper.forward(Helper.phase) eq Helper.phase)
    assert(Helper.forward(Helper.snapshot) eq Helper.snapshot)
    assert(Helper.phase.schema.get == Finite[Phase].schema.get)
    assert(Helper.snapshot.schema.get == Finite[Snapshot].schema.get)
    assert(Inherited.fs eq DerivedInherited.fs)
    println(
      "roles:" + Base64.getEncoder.encodeToString(
        ir.Model(machines = machines, types = types).toByteArray
      )
    )
