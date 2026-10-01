/* Lowers one checked Scala Model Query into a Testpilot Case through a named realization: the
 * counterpart of model/lean/Umpire/Case/Producer.lean and model/go/caseproducer. One checked Model,
 * one selected witness and one realization become one Case: the Program is the realization's
 * scaffolding with the path's actions placed where the realization binds them, and the Contract is
 * the correlated capability the Property's clauses, placed by the Scenario, lower to.
 *
 * Orders, identities and spellings follow the Lean producer exactly, because the Case bytes are
 * compared with the checked-in fixtures Lean renders.
 */
package umpire.caseproducer

import temporal.server.api.testpilot.v1.CaseOuterClass.*
import temporal.server.api.testpilot.v1.ContractOuterClass.Contract
import temporal.server.api.testpilot.v1.InstructionOuterClass.InstructionNode
import temporal.server.api.testpilot.v1.ProgramOuterClass.{
  Cleanup,
  Entrypoint,
  Observation,
  Role,
  Slot
}
import umpire.*
import umpire.Canonical.*

import scala.jdk.CollectionConverters.*

/** Names one Case. Every other identity derives from the fixture name. */
final case class Identity(
    caseID: String,
    fixture: String,
    programID: String,
    contractID: String,
    runScope: String
)

object Identity:
  /** The identity a fixture name derives under a Case ID root. */
  def of(root: String, fixture: String): Identity =
    val id = s"$root.$fixture"
    Identity(id, fixture, s"$id.program", s"$id.contract", fixture)

  /**
   * The identity the Lean `case` command gives a set's Query: the Case ID `<root>.<set>.<query>`,
   * and the fixture `<set>-<query>` every other identity derives from.
   */
  def forQuery(root: String, set: String, query: String): Identity =
    val id = s"$root.$set.$query"
    Identity(id, s"$set-$query", s"$id.program", s"$id.contract", s"$set-$query")

/** Where a node lands: the Case, the instance that performs it, and how many there are. */
final case class Placement(identity: Identity, number: Int, count: Int):
  /** The suffix an instance's ids carry: none on a Case over one instance. */
  def suffix: String = if count <= 1 then "" else s"-$number"

/**
 * Where one evidence kind is read from: one arm of the history event's attributes, or the elements
 * of a repeated field a unary RPC returns.
 */
final case class Recorded(historyAttributes: String = "", method: String = "", path: String = "")

/** What a realization knows about one admitted evidence kind. */
final case class EvidenceSource(
    eventKind: String,
    recorded: Recorded,
    operationKeyPath: String,
    kindID: String,
    sourceID: String
):
  def readsHistory: Boolean = recorded.historyAttributes.nonEmpty

/** One resolved (action, source) pair: the action a recorded event confirms, and how to read it. */
final case class EvidenceRule(action: Atom, source: EvidenceSource):
  def readsHistory: Boolean = source.readsHistory

/** One taken step as model values: its action and result. */
final private[caseproducer] case class Taken(
    action: Atom,
    state: Atom,
    outcome: Atom,
    facts: Vector[Atom]
)

/** An evidence rule and the steps its evidence confirms: the silent steps before its own, then its own. */
final private[caseproducer] case class ResolvedRule(rule: EvidenceRule, steps: Vector[Taken])

type NodeOf = (Placement, Vector[EvidenceRule]) => InstructionNode

/** One item of an entrypoint's instruction sequence. */
enum Item:
  /** A node every Case carries once. */
  case Fixed(node: NodeOf)

  /** A node a Case carries once for each instance whose path performs a class the keys name. */
  case WhenOnPath(keys: Vector[String], node: NodeOf)

  /** Where the path's actions of these classes land, in path order. */
  case Actions(classes: Vector[String])

  /** These items once per instance. */
  case PerInstance(items: Vector[Item])

/** One entrypoint of a realization's Program: how it activates, and its items. */
final case class EntrypointPlan(
    activate: (Placement, Vector[InstructionNode]) => Entrypoint,
    items: Vector[Item],
    perInstance: Boolean = false
)

/** Everything a realization's Program carries besides its actions. */
final case class ProgramPlan(
    roles: Vector[Role],
    slots: Vector[Slot] = Vector.empty,
    instanceSlots: Placement => Vector[Slot] = _ => Vector.empty,
    observations: Vector[Observation],
    entrypoints: Vector[EntrypointPlan],
    cleanup: Option[Cleanup]
)

/**
 * What one action class is realized as. `key` is the class key a Scenario spells; `action` is the
 * Definition ID the realization states, read where the key names no class.
 */
final case class ActionBinding(
    action: String,
    key: String,
    instructionID: String,
    node: (Placement, String) => InstructionNode
):
  def resolve(t: Table): String =
    if key.nonEmpty && t.actions.contains(key) then t.actionAtom(key).id else action

/** The semantic window of the projection. */
final case class ProjectionLimits(
    events: Long,
    buffered: Long,
    keys: Long,
    support: Long,
    work: Long,
    eventSize: Long
)

/** The platform-owned binding of a Model to the runtime. */
final case class Realization(
    plan: ProgramPlan,
    actions: Vector[ActionBinding],
    producerID: String,
    producerVersion: String,
    projectionID: String,
    scopeField: String,
    operationKey: String,
    historyObservation: String,
    correlatedObservation: String,
    sources: Vector[EvidenceSource],
    projectionLimits: ProjectionLimits
)

/** Where a Model is declared, as Case provenance names it. */
final case class Source(path: String, provenance: String):
  def pb: SourceLocation = SourceLocation
    .newBuilder()
    .setPath(path)
    .setLine(1)
    .setColumn(1)
    .setProvenance(provenance)
    .build()

/**
 * A production failure: the construct that could not be realized and the definition it belongs to,
 * as `Umpire.Case.Compiler.Error` names them.
 */
private[caseproducer] def reject(definition: String, construct: String)(using Fails): Nothing =
  fail(definition, construct)

/** Lowers one checked find Query into a Case. */
def produce(q: Query, identity: Identity, r: Realization, source: Source): Checked[Case] = checked {
  Production(q, identity, r, source).produce
}

/** One Case being produced. */
final private[caseproducer] class Production(
    val q: Query,
    val identity: Identity,
    val r: Realization,
    val source: Source
)(using Fails):
  val t: Table = q.scenario.machine.table.get
  val answer: Answer = q.answer.get
  if answer.outcome != Verdict.found || answer.witness.isEmpty then
    reject(s"${t.family.root}.query.${q.name}", "witness.absent")
  val initial: Atom = answer.witness.get.initial
  val steps: Vector[Taken] =
    answer.witness.get.steps.map(s => Taken(s.action, s.state, s.outcome, s.facts))

  /** The pinned schedule's action ids, in trace order. */
  val schedule: Vector[String] = q.scenario.actions.map(t.actionAtom(_).id)
  if schedule.isEmpty then reject(q.scenario.scenarioID(t), "behavior.sequence.absent")
  val opening: Atom = t.actionAtom(q.scenario.actions.head)

  def produce: Case =
    val (witnessRules, silentGaps) = resolveEvidence(derivedEvidence)
    val evidenceRules = alternativeRules(witnessRules)
    val groups = Lower(q.property).get
    val clauses = this.scopedClauses(groups)
    val propertyID = q.property.propertyID(t)
    val propertyFingerprint = fingerprint(this.correlatedPropertySemantic(propertyID, clauses))
    val plan = this.projection(evidenceRules)
    val contract = this.correlatedContract(plan, clauses)
    val program = Assembler(this, evidenceRules.map(_.rule)).program
    val scenarioSemantic = q.scenario.scenarioSemantic(t)
    val propertySemantic = t.propertySemantic(propertyID, groups)
    val queryFingerprint = fingerprint(q.queryCanonical(t, fingerprint(propertySemantic)))
    def binding(id: String, fp: String, kind: DefinitionKind) =
      DefinitionBinding
        .newBuilder()
        .setDefinitionId(id)
        .setBehaviorFingerprint(fp)
        .setKind(kind)
        .build()
    val provenance = CaseProvenance
      .newBuilder()
      .setProducerId(r.producerID)
      .setProducerVersion(r.producerVersion)
      .addAllDefinitions(
        Seq(
          binding(t.ids.target, t.targetFingerprint, DefinitionKind.DEFINITION_KIND_TARGET),
          binding(
            q.scenario.scenarioID(t),
            fingerprint(scenarioSemantic),
            DefinitionKind.DEFINITION_KIND_SCENARIO
          ),
          binding(
            s"${t.family.root}.query.${q.name}",
            queryFingerprint,
            DefinitionKind.DEFINITION_KIND_QUERY
          ),
          binding(propertyID, propertyFingerprint, DefinitionKind.DEFINITION_KIND_PROPERTY)
        ).asJava
      )
      .addAllSources(Seq.fill(4)(source.pb).asJava)
      .addAllKnownGaps(silentGaps.asJava)
      .addAllCorrelatedRules(
        clauses
          .map(c =>
            CorrelatedRuleBinding
              .newBuilder()
              .setRuleId(c.id)
              .setPropertyId(propertyID)
              .setPropertyFingerprint(propertyFingerprint)
              .setProjectionId(r.projectionID)
              .setProjectionFingerprint(plan.fingerprint)
              .setSource(source.pb)
              .build()
          )
          .asJava
      )
      .addAllAbstractionClaims(
        t.claims
          .filter(c => schedule.contains(c.member))
          .map(c =>
            AbstractionClaim
              .newBuilder()
              .setAction(c.action)
              .setField(c.field)
              .setClassName(c.className)
              .setExample(c.example)
              .build()
          )
          .asJava
      )
    val built = Case
      .newBuilder()
      .setCaseId(identity.caseID)
      .setVersion(FormatVersion.newBuilder().setMajor(1))
      .setProvenance(provenance)
      .setProgram(program)
      .setContract(Contract.newBuilder().setContractId(identity.contractID).setCorrelated(contract))
    LocalNames.localize(built)
    built.build()

  /**
   * The evidence mappings a witness implies under the machine's own evidence lines: one per fact a
   * step records that some line covers, naming the step's action.
   */
  def derivedEvidence: Vector[(String, String)] =
    steps
      .flatMap(s => s.facts.flatMap(f => lineFor(f.value).map(kind => s.action.id -> kind)))
      .distinct

  private def lineFor(fact: String): Option[String] =
    t.evidence.collectFirst {
      case (name, kind) if fact == name || fact.startsWith(s"$name-") => kind
    }

  def sourceOf(kind: String): Option[EvidenceSource] = r.sources.find(_.eventKind == kind)

  /**
   * Resolves each mapping against the realization's admitted kinds and walks the witness: an
   * observed step's rule confirms the silent steps before it together with its own, and each silent
   * step becomes a Known Gap.
   */
  def resolveEvidence(
      mappings: Vector[(String, String)]
  ): (Vector[ResolvedRule], Vector[KnownGap]) =
    val admitted = mappings.map { (action, kind) =>
      val s = sourceOf(kind).getOrElse(reject(kind, "evidence.kind-unknown"))
      if !schedule.contains(action) then reject(action, "evidence.action-unselected")
      action -> s
    }
    val (resolved, silent, gaps) =
      steps.foldLeft((Vector.empty[ResolvedRule], Vector.empty[Taken], Vector.empty[KnownGap])) {
        case ((resolved, silent, gaps), s) =>
          admitted.find(_._1 == s.action.id) match
            case None =>
              val gap = if gaps.exists(_.getSubject == s.action.id) then gaps
              else gaps :+ silentGap(s.action)
              (resolved, silent :+ s, gap)
            case Some((_, src)) =>
              if resolved.exists(_.rule.action.id == s.action.id) then
                reject(s.action.id, "evidence.action-repeated")
              (
                resolved :+ ResolvedRule(EvidenceRule(s.action, src), silent :+ s),
                Vector.empty,
                gaps
              )
      }
    if silent.nonEmpty then reject(silent.head.action.id, "evidence.action-unmapped")
    (resolved, gaps)

  /** The Known Gap a silent step records: the Contract infers it from the evidence of the step after it. */
  private def silentGap(action: Atom): KnownGap = KnownGap
    .newBuilder()
    .setKind(KnownGapKind.KNOWN_GAP_KIND_CAPABILITY)
    .setCode(s"${action.id}.unobserved")
    .setSubject(action.id)
    .setDetail(
      s"the step '${action.value}' records nothing an evidence line names, so the Contract infers it from " +
        "the evidence of the step after it rather than observing it"
    )
    .build()

  /**
   * Adds a rule for every other result of a witnessed row, declared by the first kind its facts
   * record, so a Run that took it reads as a violation rather than as nothing.
   */
  def alternativeRules(resolved: Vector[ResolvedRule]): Vector[ResolvedRule] =
    val (rules, _, _) = steps.foldLeft((resolved, initial, Vector.empty[Taken])) {
      case ((rules, prior, silent), s) =>
        val witness = rules.indexWhere(_.steps.last == s)
        val before = if witness >= 0 then rules(witness).steps.init else silent
        val added = alternativesOf(rules, prior, s, before)
        (rules ++ added, s.state, if witness >= 0 then Vector.empty else silent :+ s)
    }
    rules

  /**
   * One rule per other result of the witnessed row taken from prior, each confirming the silent
   * steps before it together with its own.
   */
  private def alternativesOf(
      rules: Vector[ResolvedRule],
      prior: Atom,
      taken: Taken,
      before: Vector[Taken]
  ): Vector[ResolvedRule] =
    resultsOf(prior.value, taken.action.value)
      .filter(_ != taken)
      .foldLeft(Vector.empty[ResolvedRule]) { (added, res) =>
        res.facts.iterator.flatMap(f => lineFor(f.value)).nextOption() match
          case None       => added
          case Some(kind) =>
            if (rules ++ added).exists(_.rule.source.eventKind == kind) then
              reject(kind, "evidence.kind-ambiguous")
            val src = sourceOf(kind).getOrElse(reject(kind, "evidence.kind-unknown"))
            added :+ ResolvedRule(EvidenceRule(taken.action, src), before :+ res)
      }

  /** Every result the machine gives an action from a state, as taken steps. */
  def resultsOf(state: String, action: String): Vector[Taken] =
    t.rowsFrom(state)
      .filter(_.action == action)
      .flatMap(
        _.results.map(res =>
          Taken(
            t.actionAtom(action),
            t.stateAtom(res.state),
            t.outcomeAtom(res.outcome),
            res.facts.map(t.factAtom)
          )
        )
      )
