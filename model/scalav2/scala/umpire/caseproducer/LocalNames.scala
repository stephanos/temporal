package umpire.caseproducer

import temporal.server.api.testpilot.v1.CaseOuterClass.{Case, LocalName}
import temporal.server.api.testpilot.v1.Correlated.*
import temporal.server.api.testpilot.v1.ExpressionOuterClass.Expression
import temporal.server.api.testpilot.v1.InstructionOuterClass.{Instruction, InstructionNode}
import temporal.server.api.testpilot.v1.ValueOuterClass.ModelValue
import umpire.Fails

import scala.collection.mutable
import scala.jdk.CollectionConverters.*

/**
 * `Umpire.Case.LocalNames.localize`: trades every Definition ID the Program and Contract name for its
 * shortest dotted suffix no other ID of the Case shares, records the renaming in provenance, and
 * rejects a renaming that would merge two names. Model value spellings stay as declared, because
 * no value of a Scala Model is a Definition ID or a structural key.
 */
private[caseproducer] object LocalNames:
  def localize(c: Case.Builder)(using Fails): Unit =
    val ids = mutable.LinkedHashSet.empty[String]
    visitCase(
      c,
      id =>
        if id.nonEmpty then ids += id; id
    )
    val all = ids.toVector
    val names = all.map(id => id -> localName(all, id)).toMap
    all.groupBy(names).foreach { (local, sharing) =>
      if sharing.size > 1 then
        reject(sharing(1), s"local-name $local names ${sharing(0)} and ${sharing(1)}")
    }
    visitCase(c, id => if id.isEmpty then id else names(id))
    for id <- all if names(id) != id do
      c.getProvenanceBuilder.addLocalNames(
        LocalName.newBuilder().setLocalName(names(id)).setDefinitionId(id)
      )

  /**
   * The shortest dotted suffix of `id` no other member of `ids` shares at that length, or `id`
   * itself when every suffix is shared.
   */
  def localName(ids: Vector[String], id: String): String =
    val segments = id.split('.').toVector
    def suffix(parts: Vector[String], count: Int) = parts.takeRight(count).mkString(".")
    (1 to segments.size).iterator
      .map(n => n -> suffix(segments, n))
      .find((n, s) => ids.forall(o => o == id || suffix(o.split('.').toVector, n) != s))
      .fold(id)(_._2)

  /**
   * Every name position of the Program and the correlated Contract, in the order the Lean traversal
   * visits them: the Program's instructions, then its evidence declarations, then the correlated
   * Contract field by field. A model value is visited by its definition's name.
   */
  private def visitCase(c: Case.Builder, name: String => String): Unit =
    val program = c.getProgramBuilder
    for e <- program.getEntrypointsBuilderList.asScala; n <- e.getInstructionsBuilderList.asScala do
      visitInstruction(n, name)
    if program.hasCleanup then
      for n <- program.getCleanupBuilder.getInstructionsBuilderList.asScala do
        visitInstruction(n, name)
    for d <- program.getEvidenceBuilderList.asScala do
      d.setEvidenceId(name(d.getEvidenceId)).setEvidenceSource(name(d.getEvidenceSource))
      for s <- d.getScopeBuilderList.asScala do s.setFieldId(name(s.getFieldId))
      for f <- d.getFieldsBuilderList.asScala do f.setFieldId(name(f.getFieldId))
    val contract = c.getContractBuilder
    for r <- contract.getRulesBuilderList.asScala do r.setRuleId(name(r.getRuleId))
    if contract.hasCorrelated then visitContract(contract.getCorrelatedBuilder, name)

  /** The correlated Contract's names, field by field, in declaration order. */
  private def visitContract(cc: CorrelatedContract.Builder, name: String => String): Unit =
    cc.setProjectionId(name(cc.getProjectionId))
    for i <- 0 until cc.getScopeFieldsCount do cc.setScopeFields(i, name(cc.getScopeFields(i)))
    cc.setOperationField(name(cc.getOperationField))
    for i <- 0 until cc.getSourcesCount do cc.setSources(i, name(cc.getSources(i)))
    if cc.hasInitialState then visitValue(cc.getInitialStateBuilder, name)
    cc.getInitialStateFieldsBuilderList.asScala.foreach(visitValue(_, name))
    cc.getTransitionsBuilderList.asScala.foreach(visitTransition(_, name))
    for r <- cc.getProjectionRulesBuilderList.asScala do
      r.setKind(name(r.getKind))
      if r.hasSubmission then visitValue(r.getSubmissionBuilder, name)
      r.getOutputsBuilderList.asScala.foreach(visitTransition(_, name))
      for f <- r.getFieldsBuilderList.asScala do f.setFieldId(name(f.getFieldId))
    for r <- cc.getRulesBuilderList.asScala do
      r.setRuleId(name(r.getRuleId))
      if r.hasTrigger then visitExpression(r.getTriggerBuilder, name)
      if r.hasResponse then visitExpression(r.getResponseBuilder, name)
      if r.hasCorrelation then visitExpression(r.getCorrelationBuilder, name)

  private def visitValue(v: ModelValue.Builder, name: String => String): Unit =
    v.setDefinitionId(name(v.getDefinitionId))

  private def visitTransition(t: CorrelatedTransition.Builder, name: String => String): Unit =
    if t.hasPriorState then visitValue(t.getPriorStateBuilder, name)
    if t.hasAction then visitValue(t.getActionBuilder, name)
    if t.hasState then visitValue(t.getStateBuilder, name)
    if t.hasOutcome then visitValue(t.getOutcomeBuilder, name)
    t.getFactsBuilderList.asScala.foreach(visitValue(_, name))
    t.getPriorFieldsBuilderList.asScala.foreach(visitValue(_, name))
    t.getStateFieldsBuilderList.asScala.foreach(visitValue(_, name))

  private def visitInstruction(n: InstructionNode.Builder, name: String => String): Unit =
    if n.hasInstruction then
      val in = n.getInstructionBuilder
      in.getInstructionCase match
        case Instruction.InstructionCase.INVOKE_RPC =>
          for
            read <- in.getInvokeRpcBuilder.getResponseReadsBuilderList.asScala
            target <- read.getTargetsBuilderList.asScala if target.hasCorrelatedEvidence
            rule <- target.getCorrelatedEvidenceBuilder.getRulesBuilderList.asScala
          do
            rule
              .setEvidenceSource(name(rule.getEvidenceSource))
              .setKind(name(rule.getKind))
              .setEvidenceId(name(rule.getEvidenceId))
        case Instruction.InstructionCase.READ_EVIDENCE =>
          val r = in.getReadEvidenceBuilder
          r.setEvidenceId(name(r.getEvidenceId))
        case _ => ()

  private def visitExpression(e: Expression.Builder, name: String => String): Unit =
    e.getExpressionCase match
      case Expression.ExpressionCase.REFERENCE =>
        val r = e.getReferenceBuilder
        if r.hasCorrelatedStep then
          val s = r.getCorrelatedStepBuilder
          s.setDefinitionId(name(s.getDefinitionId))
      case Expression.ExpressionCase.COMPARE =>
        val c = e.getCompareBuilder
        if c.hasLeft then visitExpression(c.getLeftBuilder, name)
        if c.hasRight then visitExpression(c.getRightBuilder, name)
      case Expression.ExpressionCase.PRESENT =>
        visitExpression(e.getPresentBuilder.getOperandBuilder, name)
      case Expression.ExpressionCase.PATH =>
        visitExpression(e.getPathBuilder.getOperandBuilder, name)
      case Expression.ExpressionCase.ALL =>
        e.getAllBuilder.getOperandsBuilderList.asScala.foreach(visitExpression(_, name))
      case Expression.ExpressionCase.ANY =>
        e.getAnyBuilder.getOperandsBuilderList.asScala.foreach(visitExpression(_, name))
      case _ => ()
