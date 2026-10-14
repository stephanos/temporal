package framework.residual

import java.util.Base64
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir
import fixture.residualnames.Names.*
import fixture.residualnames.{inferredFairness, Pair, Paired, Subject}

object Native:
  object Other extends Composition[Pair](_.left -> Subject, _.right -> Subject):
    def end(s: Pair): Boolean = s.left && s.right
    object syncs extends Syncs:
      sync(_.left -> emptyAction, _.right -> namedAction)
  private def emit(label: String, model: ir.Model): Unit =
    println(label + ":" + Base64.getEncoder.encodeToString(model.toByteArray))

  def main(args: Array[String]): Unit =
    emit(
      "names",
      Signature.collect(
        Vector(inferredAction, namedAction, emptyAction),
        Nil,
        Vector(inferredAssumption, namedAssumption, emptyAssumption, inferredFairness)
      )
    )
    emit(
      "limits",
      ir.Model(queries =
        Vector(inferredLimits, namedLimits, emptyLimits).map(l =>
          ir.Query(name = l.sourceName, limits = Some(l.toIR))
        )
      )
    )
    emit(
      "declarations",
      Signature.collect(
        Nil,
        Nil,
        Vector(fixture.declarations.storeOpaque, fixture.declarations.flushRuns),
        Vector(fixture.declarations.crashUnmodeled),
        Vector(fixture.declarations.two)
      )
    )
    emit(
      "declarationLimits",
      ir.Model(queries = Seq(ir.Query(limits = Some(fixture.declarations.two.toIR))))
    )
    emit(
      "composed",
      Signature.collect(
        Nil,
        Nil,
        Vector(fixture.composedprogress.flowing, fixture.composedprogress.graded)
      )
    )
    val copyEntity = inferredEntity.copy(key = "copy")
    val emptyCopyEntity = inferredEntity.copy(name = "")
    val copyObservation = inferredObservation.copy(on = copyEntity)
    val emptyCopyObservation = inferredObservation.copy(name = "")
    val copyLimits = inferredLimits.copy(steps = 1)
    val emptyCopyLimits = inferredLimits.copy(name = "")
    assert(copyEntity.sourceName == "inferredEntity")
    assert(emptyCopyEntity.sourceName == "")
    assert(copyObservation.sourceName == "inferredObservation")
    assert(emptyCopyObservation.sourceName == "")
    assert(copyLimits.toIR.name == "inferredLimits")
    assert(emptyCopyLimits.toIR.name == "")
    assert(inferredEntity.productElement(0).equals(""))
    assert(inferredObservation.productElement(0).equals(""))
    assert(inferredLimits.productElement(0).equals(""))
    assert(copyEntity.key == "copy")
    assert(copyObservation.on eq copyEntity)
    assert(namedEntity.refer("parent") eq inferredEntity)
    assert(emptyEntity.sourceName == "")
    val modified = emptyAction
      .input[Boolean]("flag")
      .results("reply")
      .on(emptyEntity)
      .creates(inferredEntity)
      .example(false, "no")
    assert(modified.toIR.name == "")
    assert(modified.toIR.on == "")
    assert(modified.toIR.creates == "inferredEntity")
    val emptySegments = framework.dsl.assume
      .fair()
      .fair(Paired.synced(_.left -> emptyAction))
      .fair()
      .retainName("framework.residual.emptySegments", "emptySegments")
    assert(emptySegments.toIR.fair.isEmpty)
    assert(
      emptySegments.toIR.fairRefs.head.selection.equals(ir.ComposedActionRef.Selection.Sync(""))
    )
    val mixed = emptySegments.fair(namedAction)
    val rejected = try
      mixed.toIR
      false
    catch case e: IllegalArgumentException => e.getMessage.contains("both actions and composed")
    assert(rejected)
    val owners = framework.dsl
      .assume("owners")
      .fair(Paired.synced(_.left -> emptyAction), Other.synced(_.left -> emptyAction))
      .retainName("framework.residual.owners", "owners")
    val multiple = try
      owners.toIR
      false
    catch
      case e: IllegalArgumentException =>
        e.getMessage.contains("assumption framework.residual.owners") &&
        e.getMessage.contains("paired and other")
    assert(multiple)
    println("controls:passed")
