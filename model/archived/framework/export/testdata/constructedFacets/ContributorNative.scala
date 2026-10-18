package framework.facetscontributorsnative

import java.util.Base64
import framework.dsl.*
import fixture.facetscontributors.{first, Forward, Reverse}
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def run(machine: Machine[?, ?, ?], args: Array[String]): Unit =
    val value = RuleBindings.collect(machine)
    val roots = value.steps.map(_.function)
    assert(
      roots == machine.table.map((decl, _) => machine.headerIdentity.get._2 + ".rules." + decl.name)
    )
    assert(RuleBindings.collect(machine).equals(value))
    assert(RuleBindings.collect(machine.restrict(first.go)).steps.size == 1)
    val model = ir.Model(
      machines = Seq(ir.Machine(name = machine.headerIdentity.get._2, steps = value.steps)),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
    println("facets:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"facets-controls:args=${args.length}")

object NativeForward:
  def main(args: Array[String]): Unit = Native.run(Forward, args)

object NativeReverse:
  def main(args: Array[String]): Unit = Native.run(Reverse, args)
