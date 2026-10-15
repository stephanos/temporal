package framework.ruleshelpernative

import java.util.Base64
import framework.dsl.*
import fixture.rulesclient.{Client, Contracts}
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val contributions = Vector(Client, Contracts).map { machine =>
      machine.headerIdentity.get._2 -> RuleBindings.collect(machine)
    }
    val functions = contributions
      .flatMap(_._2.functions)
      .groupBy(_.name)
      .toVector
      .sortBy(_._1)
      .map { (_, values) => assert(values.distinct.size == 1); values.head }
    val types = contributions
      .flatMap(_._2.types)
      .groupBy(_.name)
      .toVector
      .sortBy(_._1)
      .map { (_, values) => assert(values.map(_.clearPosition).distinct.size == 1); values.head }
    val model = ir.Model(
      machines = contributions.sortBy(_._1).map { (name, value) =>
        ir.Machine(name = name, steps = value.steps)
      },
      functions = functions,
      types = types
    )
    assert(functions.find(_.name.endsWith("states$.guarded")).get.requires.nonEmpty)
    assert(functions.find(_.name.endsWith("effects$.held")).get.requires.nonEmpty)
    assert(functions.filter(_.name.contains(".rules.")).forall(_.requires.isEmpty))
    assert(!functions.exists(_.name.endsWith("unused")))
    println("rules:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"helper-controls:args=${args.length}")
