package framework.rulespartialnative

import framework.dsl.*
import fixture.rulespartial.{Client, LocalForwarder, StoredForwarder}
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val values = Vector(Client, LocalForwarder, StoredForwarder).map { machine =>
      machine.headerIdentity.get._2 -> RuleBindings.collect(machine)
    }
    val model = ir.Model(
      machines =
        values.sortBy(_._1).map((name, value) => ir.Machine(name = name, steps = value.steps)),
      functions = values.flatMap(_._2.functions).distinct.sortBy(_.name),
      types = values.flatMap(_._2.types).distinct.sortBy(_.name)
    )
    println(
      "rules:" + java.util.Base64.getEncoder.encodeToString(model.toByteArray)
    )
    println(s"partial-controls:args=${args.length}")
