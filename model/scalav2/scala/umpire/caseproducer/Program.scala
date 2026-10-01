package umpire.caseproducer

import temporal.server.api.testpilot.v1.Correlated.NamedValue
import temporal.server.api.testpilot.v1.InstructionOuterClass.InstructionNode
import temporal.server.api.testpilot.v1.ProgramOuterClass.*
import umpire.Fails

import scala.collection.mutable
import scala.jdk.CollectionConverters.*

/**
 * `Umpire.Case.Producer.assembleProgram` for a Case over one instance: the realization's
 * entrypoints in order, each item emitting its fixed node, its conditional node, or the path's
 * actions of its classes, and every bound action on the path placed exactly once.
 */
final private[caseproducer] class Assembler(p: Production, rules: Vector[EvidenceRule])(using
    Fails
):
  private val placement = Placement(p.identity, 1, 1)

  /** Each binding resolved against the Model's vocabulary, beside the action it states. */
  private val bindings = p.r.actions.map(b => b.copy(action = b.resolve(p.t)))
  private val stated = p.r.actions.map(_.action)
  private val placed = mutable.ArrayBuffer.empty[String]

  def program: Program =
    val entrypoints = p.r.plan.entrypoints.map(e => e.activate(placement, emit(e.items)))
    for b <- bindings if onPath(b.action) && !placed.contains(b.action) do
      reject(b.action, "realization.action-unplaced")
    val builder = Program
      .newBuilder()
      .setProgramId(p.identity.programID)
      .addAllRoles(p.r.plan.roles.asJava)
      .addAllSlots((p.r.plan.slots ++ p.r.plan.instanceSlots(placement)).asJava)
      .addAllObservations(p.r.plan.observations.asJava)
      .addAllEntrypoints(entrypoints.asJava)
      .addAllEvidence(evidenceDeclarations.asJava)
    p.r.plan.cleanup.foreach(builder.setCleanup)
    builder.build()

  private def onPath(action: String): Boolean = p.schedule.contains(action)

  /** Whether the path performs a class one of the keys names. */
  private def performing(keys: Vector[String]): Boolean =
    bindings.exists(b => keys.contains(b.key) && onPath(b.action))

  /**
   * An item names classes by the ids the realization states; the ones its bindings resolved carry
   * the Model's ids instead.
   */
  private def resolvedClasses(classes: Vector[String]): Vector[String] =
    classes.map(c =>
      stated.indexOf(c) match
        case -1 => c
        case j  => bindings(j).action
    )

  private def emit(items: Vector[Item]): Vector[InstructionNode] = items.flatMap {
    case Item.Fixed(node)            => Vector(node(placement, rules))
    case Item.WhenOnPath(keys, node) =>
      if performing(keys) then Vector(node(placement, rules)) else Vector.empty
    case Item.PerInstance(inner) => emit(inner)
    case Item.Actions(classes)   =>
      val resolved = resolvedClasses(classes)
      for c <- resolved if placed.contains(c) do reject(c, "realization.action-placed-twice")
      placed ++= resolved
      actionNodes(resolved)
  }

  /**
   * Every occurrence on the path whose class the item names, in path order; a class performed again
   * appends its 1-based ordinal to the binding's instruction id.
   */
  private def actionNodes(classes: Vector[String]): Vector[InstructionNode] =
    val seen = mutable.Map.empty[String, Int].withDefaultValue(0)
    p.schedule.flatMap { action =>
      val ordinal = seen(action)
      seen(action) += 1
      if !classes.contains(action) then None
      else
        val b =
          bindings.find(_.action == action).getOrElse(reject(action, "realization.action-unbound"))
        val id =
          b.instructionID + placement.suffix + (if ordinal > 0 then s"-${ordinal + 1}" else "")
        Some(b.node(placement, id))
    }

  /**
   * Each admitted kind the rules read, declared once in the order the rules first name them, scoped
   * to this Case's Run.
   */
  private def evidenceDeclarations: Vector[EvidenceDeclaration] =
    rules.map(_.source).distinctBy(_.kindID).map { s =>
      val d = EvidenceDeclaration
        .newBuilder()
        .setEvidenceId(s.kindID)
        .setEvidenceSource(s.sourceID)
        .addScope(
          NamedValue
            .newBuilder()
            .setFieldId(p.r.scopeField)
            .setValue(Build.text(p.identity.runScope))
        )
        .setOperation(s.operationKeyPath)
      if s.readsHistory then
        d.setHistoryEvent(
          HistoryEventSource.newBuilder().setAttributesField(s.recorded.historyAttributes)
        )
      else d.setRead(ReadSource.newBuilder().setMethod(s.recorded.method).setPath(s.recorded.path))
      d.build()
    }
