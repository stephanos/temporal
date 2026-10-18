package framework.evidencenative

import java.util.Base64
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  private def collect(machine: Machine[?, ?, ?]): Option[AnyRef] =
    val collector = try Some(java.lang.Class.forName("framework.dsl.MachineEvidence$"))
    catch case _: ClassNotFoundException => None
    collector.flatMap { cls =>
      val method = try Some(cls.getMethod("collect", classOf[Machine[?, ?, ?]]))
      catch case _: NoSuchMethodException => None
      method.map { selected =>
        try selected.invoke(cls.getField("MODULE$").get(null), machine)
        catch case error: java.lang.reflect.InvocationTargetException => throw error.getCause
      }
    }
  def main(args: Array[String]): Unit =
    val machine = java.lang.Class
      .forName(args(0) + "$")
      .getField("MODULE$")
      .get(null)
      .asInstanceOf[Machine[?, ?, ?]]
    val expected = ir.Model.parseFrom(Base64.getDecoder.decode(args(1)))
    val result = collect(machine)
    assert(
      result.nonEmpty,
      "complete evidence contribution is absent for independently admitted root " +
        expected.machines.head.evidence + "; reached functions=" + expected.functions
          .map(_.name)
          .mkString(",") +
        "; reached types=" + expected.types.map(_.name).mkString(",")
    )
    val contribution = result.get
    def field[A](name: String): A =
      contribution.getClass.getMethod(name).invoke(contribution).asInstanceOf[A]
    assert(collect(machine).contains(contribution))
    val source =
      machine.getClass.getMethod("evidenceSource").invoke(machine).asInstanceOf[Machine[?, ?, ?]]
    val original = collect(source)
    val built = source.restrict()
    assert(collect(built).equals(original))
    assert(collect(source).equals(original))
    assert(collect(machine).contains(contribution))
    val peerName = if args(0).endsWith(".Default") then "Ordinary" else "Default"
    val peer = java.lang.Class
      .forName("fixture.constructedevidence." + peerName + "$")
      .getField("MODULE$")
      .get(null)
      .asInstanceOf[Machine[?, ?, ?]]
    val peerValue = collect(peer)
    assert(collect(machine).contains(contribution))
    assert(collect(peer).equals(peerValue))
    val evidence = field[Option[String]]("evidence")
    val model = ir.Model(
      machines = Seq(ir.Machine(evidence = evidence.getOrElse(""))),
      functions = field[Vector[ir.Function]]("functions").sortBy(_.name),
      types = field[Vector[ir.Type]]("types").sortBy(_.name)
    )
    println("channels:" + field[Vector[String]]("channels").mkString(","))
    println("holes:" + field[Vector[String]]("holes").mkString(","))
    println("evidence:" + Base64.getEncoder.encodeToString(model.toByteArray))
