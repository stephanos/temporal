package framework.facetsnative

import java.util.Base64
import framework.dsl.*
import fixture.constructedfacets.{A, B, MultiA, MultiB}
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def collect(machine: Machine[?, ?, ?]): ir.Model =
    val value = RuleBindings.collect(machine)
    ir.Model(
      machines = Seq(ir.Machine(name = machine.headerIdentity.get._2, steps = value.steps)),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
  def run(first: Machine[?, ?, ?], second: Machine[?, ?, ?], args: Array[String]): Unit =
    val before = collect(first)
    val other = collect(second)
    assert(collect(first).equals(before))
    assert(collect(second).equals(other))
    println("facets:" + Base64.getEncoder.encodeToString(before.toByteArray))
    println(s"facets-controls:args=${args.length}")

object NativeA:
  def main(args: Array[String]): Unit = Native.run(A, B, args)

object NativeB:
  def main(args: Array[String]): Unit = Native.run(B, A, args)

object NativeMA:
  def main(args: Array[String]): Unit = Native.run(MultiA, MultiB, args)

object NativeMB:
  def main(args: Array[String]): Unit = Native.run(MultiB, MultiA, args)
