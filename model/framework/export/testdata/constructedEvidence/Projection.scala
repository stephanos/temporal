package framework.evidencecontext

import java.util.Base64
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir

object Projection:
  private def decode(value: String): String =
    new String(Base64.getDecoder.decode(value), java.nio.charset.StandardCharsets.UTF_8)
  private def context(value: String): MachineEvidence.SourceIndex =
    MachineEvidence.SourceIndex(
      decode(value).linesIterator
        .filter(_.nonEmpty)
        .map { line =>
          val fields = line.split("\t", -1)
          require(fields.length == 2)
          MachineEvidence.IndexedSource(
            decode(fields(0)),
            Option(fields(1)).filter(_ != "-").map(decode)
          )
        }
        .toVector
    )
  private def model(value: MachineEvidence.Contribution): ir.Model =
    ir.Model(
      machines = Seq(ir.Machine(evidence = value.evidence.getOrElse(""))),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
  private def emit(name: String, value: MachineEvidence.Contribution): Unit =
    println(name + ":" + Base64.getEncoder.encodeToString(model(value).toByteArray))
  def main(args: Array[String]): Unit =
    val machine = java.lang.Class
      .forName(args(0) + "$")
      .getField("MODULE$")
      .get(null)
      .asInstanceOf[Machine[?, ?, ?]]
    val a = context(args(1))
    val b = context(args(2))
    val canonical = MachineEvidence.collect(machine)
    val first = MachineEvidence.project(canonical, a)
    val second = MachineEvidence.project(canonical, b)
    assert(MachineEvidence.project(canonical, a).equals(first))
    assert(MachineEvidence.project(canonical, b).equals(second))
    assert(MachineEvidence.collect(machine).equals(canonical))
    val source = machine.evidenceSource
    val original = MachineEvidence.collect(source)
    val built = source.restrict()
    assert(MachineEvidence.collect(built).equals(original))
    assert(
      MachineEvidence
        .project(MachineEvidence.collect(built), a)
        .equals(
          MachineEvidence.project(original, a)
        )
    )
    assert(MachineEvidence.collect(source).equals(original))
    assert(MachineEvidence.collect(machine).equals(canonical))
    emit("canonical", canonical)
    emit("first", first)
    emit("second", second)
