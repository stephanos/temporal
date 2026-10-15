package framework.facetsordinarynative

import java.util.Base64
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def run(machine: Machine[?, ?, ?], args: Array[String]): Unit =
    val value = RuleBindings.collect(machine)
    assert(RuleBindings.collect(machine).equals(value))
    assert(value.channels.isEmpty && value.holes.isEmpty)
    val model = ir.Model(
      machines = Seq(ir.Machine(name = machine.headerIdentity.get._2, steps = value.steps)),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
    println("facets:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"facets-controls:args=${args.length}")

object NativeWholeOutside:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.WholeOutside, args)

object NativeWholeInFrom:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.WholeInFrom, args)

object NativeClassOutside:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.ClassOutside, args)

object NativeClassInFrom:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.ClassInFrom, args)

object NativeTwoOutside:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.TwoOutside, args)

object NativeTwoInFrom:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.TwoInFrom, args)

object NativeThreeOutside:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.ThreeOutside, args)

object NativeThreeInFrom:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.ThreeInFrom, args)

object NativeFourOutside:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.FourOutside, args)

object NativeFourInFrom:
  def main(args: Array[String]): Unit = Native.run(fixture.facetstargets.FourInFrom, args)

object NativeArities:
  def main(args: Array[String]): Unit = Native.run(fixture.facetsshapes.Arities, args)

object NativePhaseRules:
  def main(args: Array[String]): Unit = Native.run(fixture.facetsshapes.PhaseRules, args)

object NativeDomains:
  def main(args: Array[String]): Unit = Native.run(fixture.facetsdomains.Domains, args)

object NativeSections:
  def main(args: Array[String]): Unit = Native.run(fixture.facetssections.Sections, args)

object NativeRejecting:
  def main(args: Array[String]): Unit = Native.run(fixture.facetssections.Rejecting, args)
