package umpire.caseproducer

import temporal.server.api.testpilot.v1.Correlated.*
import temporal.server.api.testpilot.v1.ExpressionOuterClass.{CorrelatedStepField, Expression}
import temporal.server.api.testpilot.v1.ValueOuterClass.ModelValue
import umpire.*
import umpire.Canonical.*

import scala.jdk.CollectionConverters.*

/**
 * A clause pattern: the trace field, the value it references, and the value it equals. The field is
 * selected-action, resulting-state, outcome or observation.
 */
final private[caseproducer] case class Pattern(field: String, reference: String, value: String):
  def json: String =
    s"""{"field":${quote(field)},"reference":${quote(
        reference
      )},"constraint":{"kind":"equals","value":${quote(value)}}}"""

  /** Whether one taken step already carries the pattern's value. */
  def holds(s: Taken): Boolean =
    def carries(a: Atom) = a.id == reference && a.value == value
    field match
      case "selected-action" => carries(s.action)
      case "resulting-state" => carries(s.state)
      case "outcome"         => carries(s.outcome)
      case "observation"     => s.facts.exists(carries)
      case _                 => false

  /** The correlated step field the pattern's trace field reads. */
  def stepField: CorrelatedStepField = field match
    case "selected-action" => CorrelatedStepField.CORRELATED_STEP_FIELD_ACTION
    case "outcome"         => CorrelatedStepField.CORRELATED_STEP_FIELD_OUTCOME
    case "resulting-state" => CorrelatedStepField.CORRELATED_STEP_FIELD_STATE
    case _                 => CorrelatedStepField.CORRELATED_STEP_FIELD_FACT

  /** The step condition the pattern lowers to: its step reference compared equal with the text. */
  def condition: Expression =
    Build.equal(Build.correlatedStep(stepField, reference), Build.literal(Build.text(value)))

/**
 * One lowered requirement as an operation-correlated clause: from the operation's first selected
 * action, the required value is due within as many transitions as the Scenario places between them.
 */
final private[caseproducer] case class Clause(id: String, response: Pattern, bound: Int)

/**
 * The checked projection: its rules sorted by kind, its canonical fingerprint, and the rows an
 * operation can take under the actions its rules confirm.
 */
final private[caseproducer] case class ProjectionPlan(
    rules: Vector[ResolvedRule],
    sources: Vector[String],
    fingerprint: String,
    transitions: Vector[(Atom, Taken)]
)

extension (p: Production)(using Fails)
  /**
   * Places every lowered clause by the Scenario: its trigger is the operation's first action, its
   * bound the position of the clause's own action in the pinned schedule. A clause whose value the
   * trace already carries before its action would answer without the action being observed, so it
   * rejects. Clauses come in the checked Property's order, which is by clause id.
   */
  private[caseproducer] def scopedClauses(groups: Vector[Group]): Vector[Clause] =
    val t = p.t
    val propertyID = p.q.property.propertyID(t)
    val out = groups.flatMap { g =>
      val bound = p.schedule.indexOf(t.actionAtom(g.trigger).id)
      if bound < 0 then reject(propertyID, "property.clause-occurrence")
      g.requirements.map { r =>
        val id = s"$propertyID.${r.label}"
        val response = r.kind match
          case RequirementKind.fact    => Pattern("observation", t.factAtom(r.value).id, r.value)
          case RequirementKind.outcome => Pattern("outcome", t.outcomeAtom(r.value).id, r.value)
          case RequirementKind.state => Pattern("resulting-state", t.stateAtom(r.value).id, r.value)
        if p.steps.take(bound).exists(response.holds) then
          reject(id, "property.clause-early-response")
        Clause(id, response, bound)
      }
    }
    if out.isEmpty then reject(propertyID, "property.clauses.absent")
    out.sortBy(_.id)

  private[caseproducer] def trigger: Pattern =
    Pattern("selected-action", p.opening.id, p.opening.value)

  /**
   * The semantic string of the Property the Case carries: no same-step clauses, and one correlated
   * rule per placed clause (`correlatedRuleJson`).
   */
  private[caseproducer] def correlatedPropertySemantic(
      propertyID: String,
      clauses: Vector[Clause]
  ): String =
    val rules = clauses.map(c =>
      s"""{"id":${quote(
          c.id
        )},"kind":"correlated-eventually-within/v1","trigger":${trigger.json},"response":${c.response.json},""" +
        s""""scope":[${quote(p.r.scopeField)}],"key":${quote(
            p.r.operationKey
          )},"clock":"operation-transitions",""" +
        s""""bound":${c.bound},"ending":"partial"}"""
    )
    p.t.propertyHeader(
      propertyID
    ) + s""","logicalTimeSource":null,"clauses":[],"correlatedRules":${array(rules)}}"""

  /** `Case.Projection.check` over the declaration the evidence rules make. */
  private[caseproducer] def projection(rules: Vector[ResolvedRule]): ProjectionPlan =
    val sources = rules.map(_.rule.source.sourceID).distinct.sorted
    val sorted = rules.sortBy(_.rule.source.kindID)
    ProjectionPlan(
      sorted,
      sources,
      fingerprint(projectionCanonical(sorted, sources)),
      projectedRows(sorted)
    )

  /** The array the projection's fingerprint hashes. */
  private def projectionCanonical(sorted: Vector[ResolvedRule], sources: Vector[String]): String =
    val ruleJson = sorted.map { r =>
      val confirmed = r.steps.map(s =>
        s"[${quote(s.action.value)},${quote(s.state.value)},${quote(s.outcome.value)},${array(s.facts.map(f => quote(f.value)))}]"
      )
      s"""[${quote(r.rule.source.kindID)},[],["confirmed",null,${array(confirmed)}]]"""
    }
    val l = p.r.projectionLimits
    // The limits are numbers written into the array as they are, not strings.
    val limits = Seq(l.events, l.buffered, l.keys, l.support, l.work, l.eventSize).map(_.toString)
    array(
      Seq(
        quote("checked-projection/v2"),
        quote(p.r.projectionID),
        quote(p.t.targetFingerprint),
        quote(p.t.setupKey),
        quote(p.initial.value),
        array(Seq(quote(p.r.scopeField))),
        quote(p.r.operationKey),
        array(quoted(sources)),
        array(ruleJson),
        array(limits)
      )
    )

  /**
   * Every row an operation can take: from the start, under the actions the rules confirm, and the
   * states those reach.
   */
  private def projectedRows(sorted: Vector[ResolvedRule]): Vector[(Atom, Taken)] =
    val relevant = sorted.flatMap(_.steps.map(_.action.value)).distinct
    for prior <- reachableUnder(relevant); action <- relevant; res <- p.resultsOf(prior, action)
    yield (p.t.stateAtom(prior), res)

  /**
   * The states an operation reaches from the start by these actions, breadth-first, each frontier in
   * the order the states were first reached (`reachableStates`).
   */
  private def reachableUnder(actions: Vector[String]): Vector[String] =
    Iterator
      .iterate((Vector(p.initial.value), Vector(p.initial.value))) { (visited, frontier) =>
        val next = frontier
          .flatMap(s => actions.flatMap(a => p.resultsOf(s, a).map(_.state.value)))
          .distinct
          .filterNot(visited.contains)
        (visited ++ next, next)
      }
      .dropWhile(_._2.nonEmpty)
      .next()
      ._1

  /** `Umpire.Case.Correlated.lower`'s wire form. */
  private[caseproducer] def correlatedContract(
      plan: ProjectionPlan,
      clauses: Vector[Clause]
  ): CorrelatedContract =
    CorrelatedContract
      .newBuilder()
      .setProjectionId(p.r.projectionID)
      .setProjectionFingerprint(plan.fingerprint)
      .setEvidenceObservationId(p.r.correlatedObservation)
      .addScopeFields(p.r.scopeField)
      .setOperationField(p.r.operationKey)
      .addAllSources(plan.sources.asJava)
      .setInitialState(modelValue(p.initial))
      .addAllInitialStateFields(fields(p.initial.value).asJava)
      .addAllTransitions(
        plan.transitions
          .map((prior, res) =>
            output(res).toBuilder
              .setPriorState(modelValue(prior))
              .addAllPriorFields(fields(prior.value).asJava)
              .build()
          )
          .asJava
      )
      .addAllProjectionRules(
        plan.rules
          .map(r =>
            CorrelatedProjectionRule
              .newBuilder()
              .setKind(r.rule.source.kindID)
              .setMeaning(CorrelatedEvidenceMeaning.CORRELATED_EVIDENCE_MEANING_CONFIRMED)
              .addAllOutputs(r.steps.map(output).asJava)
              .build()
          )
          .asJava
      )
      .addAllRules(
        clauses
          .map(c =>
            CorrelatedRule
              .newBuilder()
              .setRuleId(c.id)
              .setClock(CorrelatedClock.CORRELATED_CLOCK_OPERATION_TRANSITIONS)
              .setBound(c.bound.toLong)
              .setEnding(TraceEnding.TRACE_ENDING_PARTIAL)
              .setTrigger(trigger.condition)
              .setResponse(c.response.condition)
              .build()
          )
          .asJava
      )
      .build()

  private def output(s: Taken): CorrelatedTransition = CorrelatedTransition
    .newBuilder()
    .setAction(modelValue(s.action))
    .setState(modelValue(s.state))
    .setOutcome(modelValue(s.outcome))
    .addAllFacts(s.facts.map(modelValue).asJava)
    .addAllStateFields(fields(s.state.value).asJava)
    .build()

  private def fields(state: String): Vector[ModelValue] = p.t.fieldValues(state).map(modelValue)

private def modelValue(a: Atom): ModelValue =
  ModelValue.newBuilder().setDefinitionId(a.id).setValue(a.value).build()
