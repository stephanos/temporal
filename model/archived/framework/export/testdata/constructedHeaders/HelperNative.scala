package framework.headerclient

import java.util.Base64
import framework.dsl.*
import fixture.headerclient.{DerivedInherited, Inherited}
import fixture.headerlibrary.{Counts, LibraryMachine}
import io.temporal.server.api.umpire.v1 as ir

object Client:
  def main(args: Array[String]): Unit =
    val declared = MachineHeaders.collect(Inherited)
    assert(declared.header.family == "fixture.headerclient")
    assert(declared.header.name == "inherited")
    assert(declared.header.stateType == "fixture.constructedheader.State")
    assert(declared.header.outcomeType == "fixture.constructedheader.Outcome")
    val derived = MachineHeaders.collect(DerivedInherited)
    assert(derived.header.name == "derivedInherited")
    assert(Inherited.fs eq DerivedInherited.fs)
    val library = LibraryMachine
    assert(Counts.init == 1 && Counts.name == 0)
    val header = MachineHeaders.collect(library)
    assert(header.header.family == "fixture.headerlibrary")
    assert(header.header.name == "libraryMachine")
    assert(Counts.init == 1 && Counts.name == 0)
    assert(MachineHeaders.collect(library) == header)
    val entries = Vector(declared, derived)
    val types = entries.flatMap(_.types).groupBy(_.name).toVector.sortBy(_._1).map { (_, copies) =>
      assert(copies.distinct.size == 1)
      copies.head
    }
    val projected = entries
      .map { entry =>
        val h = entry.header
        ir.Machine(
          family = h.family,
          name = h.name,
          stateType = h.stateType,
          outcomeType = h.outcomeType,
          factType = h.factType
        )
      }
      .sortBy(_.name)
    println(
      "helper:" + Base64.getEncoder.encodeToString(
        ir.Model(machines = projected, types = types).toByteArray
      )
    )
    println("helper-controls:passed,init=1,name=0")
