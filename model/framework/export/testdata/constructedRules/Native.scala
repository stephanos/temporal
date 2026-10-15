package framework.rulesnative

import java.util.Base64
import framework.dsl.*
import fixture.qualifiedrules as targets
import fixture.rulesshapes as shapes
import fixture.rulesdomains as domains
import fixture.rulessections as sections
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val machines = Vector[Machine[?, ?, ?]](
      targets.WholeOutside,
      targets.WholeInFrom,
      targets.ClassOutside,
      targets.ClassInFrom,
      targets.TwoOutside,
      targets.TwoInFrom,
      targets.ThreeOutside,
      targets.ThreeInFrom,
      targets.FourOutside,
      targets.FourInFrom,
      shapes.Arities,
      shapes.PhaseRules,
      domains.Domains,
      sections.Sections,
      sections.Rejecting
    )
    val contributions = machines.map { machine =>
      val value = RuleBindings.collect(machine)
      assert(RuleBindings.collect(machine).equals(value))
      machine.headerIdentity.get._2 -> value
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
    assert(
      contributions.flatMap(_._2.channels).isEmpty && contributions.flatMap(_._2.holes).isEmpty
    )
    val retained = RuleBindings.collect(domains.Domains.restrict(domains.hand.bool).unmonitored)
    assert(retained.steps.map(_.function) == Vector("domains.rules.bool"))
    assert(retained.functions.exists(_.name == "domains.rules.bool"))
    val disabled = RuleBindings.collect(sections.Sections.restrict(sections.hand.stopped))
    assert(disabled.steps.map(_.function) == Vector("sections.rules.stopped"))
    val fullSections = contributions.find(_._1 == "sections").get._2
    assert(disabled.types.nonEmpty)
    disabled.types.foreach { t =>
      assert(
        fullSections.types.find(_.name == t.name).exists(_.position.equals(t.position)),
        s"disabled-only type ${t.name} lost its definition position: ${t.position}"
      )
    }
    val replaced = domains.Domains
      .rebind(
        StepBinding[Boolean, Boolean, Nothing](domains.hand.bool.decl, domains.Domains.effects.keep)
      )
      .restrict()
    val refused = scala.util.Try(RuleBindings.collect(replaced)).failed.get
    assert(refused.getMessage.contains("rebind"))
    assert(functions.forall(_.position.exists(_.line > 0)))
    assert(
      shapes.Arities.rules.table
        .flatMap {
          case (_, Bound.Ruled(rows)) => rows
          case _                      => Vector.empty
        }
        .forall(_.productArity == 7)
    )
    println("rules:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"rules-controls:roots=${machines.size};args=${args.length}")
