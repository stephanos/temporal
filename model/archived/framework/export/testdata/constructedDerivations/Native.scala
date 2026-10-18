package framework.derivationsnative

import java.util.Base64
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val module = java.lang.Class
      .forName(args(0) + "$")
      .getField("MODULE$")
      .get(null)
      .asInstanceOf[Machine[?, ?, ?]]
    val source = module.ruleSource
    val original = RuleBindings.collect(source)
    val value = RuleBindings.collect(module)
    if args(0).startsWith("fixture.derivationfacets.") then
      val simple = args(0).stripPrefix("fixture.derivationfacets.")
      val peer = (if simple.startsWith("A") then "B" else "A") + simple.drop(1)
      val other = java.lang.Class
        .forName("fixture.derivationfacets." + peer + "$")
        .getField("MODULE$")
        .get(null)
        .asInstanceOf[Machine[?, ?, ?]]
      val otherValue = RuleBindings.collect(other)
      assert(RuleBindings.collect(module).equals(value))
      assert(RuleBindings.collect(other).equals(otherValue))
    assert(RuleBindings.collect(source).equals(original))
    assert(RuleBindings.collect(module).equals(value))
    println("channels:" + value.channels.mkString(","))
    println("holes:" + value.holes.mkString(","))
    val model = ir.Model(
      machines = Seq(ir.Machine(name = module.headerIdentity.get._2, steps = value.steps)),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
    println("derivations:" + Base64.getEncoder.encodeToString(model.toByteArray))
