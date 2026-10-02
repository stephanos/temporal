package umpire

import java.nio.charset.StandardCharsets.UTF_8
import java.security.MessageDigest

/** Canonical encodings and Behavior Fingerprints, byte-compatible with the Lean ones in
  * model/lean/Umpire/Model/Canonical.lean, Scenario/Check.lean, Property/Check.lean and
  * Query/Check.lean. A fingerprint is "sha256:" and the hex SHA-256 of a domain line and the
  * canonical content (`Umpire.Fingerprint.derive`). */
object Canonical:
  /** The Behavior Fingerprint of already-canonical content. */
  def fingerprint(canonical: String): String =
    val digest = MessageDigest.getInstance("SHA-256").digest(("umpire.behavior-fingerprint/v1\n" + canonical).getBytes(UTF_8))
    "sha256:" + digest.map(b => f"${b & 0xff}%02x").mkString

  /** `Lean.Json.compress` of a string: JSON escaping without HTML escaping. */
  def quote(s: String): String =
    val b = StringBuilder("\"")
    s.foreach {
      case '"'                => b ++= "\\\""
      case '\\'               => b ++= "\\\\"
      case '\n'               => b ++= "\\n"
      case '\r'               => b ++= "\\r"
      case '\t'               => b ++= "\\t"
      case c if c < ' '       => b ++= f"\\u${c.toInt}%04x"
      case ' '           => b ++= "\\u2028"
      case ' '           => b ++= "\\u2029"
      case c                  => b += c
    }
    (b += '"').result()

  def array(items: Iterable[String]): String = items.mkString("[", ",", "]")
  def quoted(items: Iterable[String]): Vector[String] = items.map(quote).toVector
  /** `Canonical.canonicalStrings`: sorted, duplicates removed. */
  def sortedUnique(items: Iterable[String]): Vector[String] = items.toVector.distinct.sorted

  private final case class Decl(id: String, kind: String)
  private given Ordering[Decl] = Ordering.by(d => (d.id, d.kind))

  extension (t: Table)
    /** The catalog members a machine's provider gives meaning to. */
    private def meanings: Vector[Decl] =
      val ids = t.ids
      (ids.states.map(Decl(_, "state")) ++ ids.stateFields.map(f => Decl(f._2, "state")) ++
        ids.actions.map(Decl(_, "action")) ++ ids.outcomes.map(Decl(_, "outcome")) ++ ids.facts.map(Decl(_, "fact"))).sorted

    private def meaningsJson: String = array(t.meanings.map(m =>
      s"""{"id":${quote(m.id)},"kind":${quote(m.kind)},"behaviorVersion":${quote(m.id)}}"""))

    /** `Canonical.targetSemanticJson` for a declared machine. */
    def targetSemantic: String =
      val decls = (t.meanings ++ Vector(Decl(t.family.target(t.owner), "target"), Decl(t.kernelID, "machine"),
        Decl(t.capabilityID, "capability"), Decl(t.lawID, "law"), Decl(t.providerID, "provider")) ++
        t.rows.map(r => Decl(t.family.id("relation", t.owner, r.key), "relation"))).sorted
      val declJson = decls.map(d =>
        s"""{"id":${quote(d.id)},"kind":${quote(d.kind)},"version":1,"behaviorVersion":${quote(d.id)}}""")
      val provider = s"""{"id":${quote(t.providerID)},"capabilityId":${quote(t.capabilityID)},""" +
        s""""capabilityVersion":1,"behaviorVersion":${quote(t.capabilityID)},"meanings":${t.meaningsJson},""" +
        s""""laws":[{"id":${quote(t.lawID)},"body":${quote(t.lawID)}}]}"""
      s"""{"id":${quote(t.family.target(t.owner))},"declarations":${array(declJson)},""" +
        s""""requiredCapabilities":${array(Seq(quote(t.capabilityID)))},"providers":${array(Seq(provider))},""" +
        s""""connectors":[],"kernel":{"id":${quote(t.kernelID)},"version":1},"behavior":${t.behaviorJson}}"""

    /** The machine's Behavior Fingerprint. */
    def targetFingerprint: String = fingerprint(t.targetSemantic)

    /** `targetBehaviorDescriptionJson`: sorted domains, the initial states under the machine's one
      * setup, and every row result sorted as the derived `Ord` on the row orders it. */
    private def behaviorJson: String =
      given Ordering[Iterable[String]] = Ordering.Implicits.seqOrdering[Seq, String].on(_.toSeq)
      val rows = t.rows.flatMap(r => r.results.map(res => (r.source, r.action, res.outcome, res.state, res.facts: Iterable[String])))
        .distinct.sorted
      val transitions = rows.map((p, a, o, s, f) =>
        s"""{"priorState":${quote(p)},"action":${quote(a)},"outcome":${quote(o)},"state":${quote(s)},"facts":${array(quoted(f))}}""")
      val setup = t.setupKey
      val initial = sortedUnique(t.starts).map(s => s"""{"setup":${quote(setup)},"state":${quote(s)}}""")
      s"""{"domains":{"setups":${array(Seq(quote(setup)))},"states":${array(quoted(sortedUnique(t.states)))},""" +
        s""""actions":${array(quoted(sortedUnique(t.actions)))},"outcomes":${array(quoted(sortedUnique(t.outcomes)))},""" +
        s""""observations":${array(quoted(sortedUnique(t.facts)))}},"initialStates":${array(initial)},""" +
        s""""transitions":${array(transitions)}}"""

    /** The machine's one setup, spelled as the Lean `starts:` line names the start: the start
      * state's first field. */
    def setupKey: String = Keys.actionName(t.starts.head)

    /** A Property semantic string up to its meanings: the part every Property of the machine shares. */
    def propertyHeader(propertyID: String): String =
      s"""{"id":${quote(propertyID)},"version":1,"requires":${array(Seq(quote(t.capabilityID)))},""" +
        s""""capabilities":[{"id":${quote(t.capabilityID)},"version":1,"behaviorVersion":${quote(t.capabilityID)}}],""" +
        s""""meanings":${t.meaningsJson}"""

    /** `propertySemanticJson` for a same-step Property lowered to its clauses. */
    def propertySemantic(propertyID: String, groups: Seq[Group]): String =
      val clauses = groups.flatMap(g => g.requirements.map(r => s"$propertyID.${r.label}" -> clauseJson(propertyID, g.trigger, r)))
        .sortBy(_._1).map(_._2)
      t.propertyHeader(propertyID) + s""","logicalTimeSource":null,"clauses":${array(clauses)}}"""

    private def clauseJson(propertyID: String, action: String, r: Requirement): String =
      val id = quote(s"$propertyID.${r.label}")
      val trigger = pattern("selected-action", t.family.id("action", t.owner, action), action)
      r.kind match
        case RequirementKind.fact =>
          s"""{"id":$id,"kind":"input-output","input":$trigger,"output":${pattern("observation", t.family.id("fact", t.owner, r.value), r.value)}}"""
        case RequirementKind.outcome =>
          s"""{"id":$id,"kind":"transition-contract","precondition":$trigger,"postcondition":${pattern("outcome", t.family.id("outcome", t.owner, r.value), r.value)}}"""
        case RequirementKind.state =>
          s"""{"id":$id,"kind":"transition-contract","precondition":$trigger,"postcondition":${pattern("resulting-state", t.family.id("state", t.owner, r.value), r.value)}}"""

  /** One clause pattern: a trace field, the value it references and the constraint. */
  private def pattern(field: String, reference: String, value: String): String =
    s"""{"field":${quote(field)},"reference":${quote(reference)},"constraint":{"kind":"equals","value":${quote(value)}}}"""

  extension (s: ScenarioDecl)
    /** `behaviorSemanticJson` for a pinned Scenario. */
    def scenarioSemantic(t: Table): String =
      def actionID(key: String) = t.family.id("action", t.owner, key)
      def occurrence(n: Int) = t.family.id("occurrence", s.name, n.toString)
      val allowed = sortedUnique(s.actions.map(actionID))
      val counts = s.actions.map(actionID).groupMapReduce(identity)(_ => 1)(_ + _)
      val occurrences = s.actions.zipWithIndex.map((a, i) => s"""{"id":${quote(occurrence(i + 1))},"action":${quote(actionID(a))}}""")
      val ordering = (1 until s.actions.size).map(i => s"""{"before":${quote(occurrence(i))},"after":${quote(occurrence(i + 1))}}""")
      val bounds = allowed.map(a => s"""{"action":${quote(a)},"minimum":${counts(a)},"maximum":${counts(a)}}""")
      s"""{"id":${quote(s.scenarioID(t))},"version":1,"requires":${array(Seq(quote(t.capabilityID)))},""" +
        s""""roles":[{"id":${quote(t.roleID)},"valueKind":"state"}],""" +
        s""""setup":[{"id":${quote(t.family.id("setup", s.name, t.entity))},"relation":"equal","left":{"role":${quote(t.roleID)}},""" +
        s""""right":{"value":{"identity":${quote(t.family.id("state", t.owner, s.start))},"value":${quote(s.start)}}}}],""" +
        s""""allowedActions":${array(quoted(allowed))},"requiredOccurrences":${array(occurrences)},"forbiddenActions":[],""" +
        s""""occurrenceBounds":${array(bounds)},"ordering":${array(ordering)},"sequences":[],"adjacencies":[],""" +
        s""""actionsExactly":${array(s.actions.map(a => quote(actionID(a))))},"traceExactly":null,"spaceStatus":"unclassified"}"""

  extension (q: Query)
    /** The Query's canonical form (`Umpire.Query.Check`), which its fingerprint hashes whole. */
    def queryCanonical(t: Table, propertyFingerprint: String): String =
      val start = q.scenario.start
      val role = s"""[[{"role":${quote(t.roleID)},"value":{"definitionId":${quote(t.family.id("state", t.owner, start))},"value":${quote(start)}}}]]"""
      val actions = t.actions.map(a => s"""{"definitionId":${quote(t.family.id("action", t.owner, a))},"value":${quote(a)}}""")
      def limit(v: Int, unit: String) = s"""{"value":$v,"unit":${quote(unit)}}"""
      s"""{"id":${quote(s"${t.family.root}.query.${q.name}")},"version":1,"form":${quote(q.form.toString)},""" +
        s""""properties":[{"id":${quote(q.property.propertyID(t))},"behaviorFingerprint":${quote(propertyFingerprint)}}],""" +
        s""""behavior":{"id":${quote(q.scenario.scenarioID(t))},"behaviorFingerprint":${quote(fingerprint(q.scenario.scenarioSemantic(t)))}},""" +
        s""""limits":{"steps":${limit(q.limits.steps, "steps")},"actions":${limit(q.limits.actions, "actions")},"search":${limit(q.limits.search, "search")}},""" +
        s""""policy":{"strategy":"shortest","seed":17},""" +
        s""""target":{"id":${quote(t.family.target(t.owner))},"behaviorFingerprint":${quote(t.targetFingerprint)},""" +
        s""""composition":[${quote(t.capabilityID)},${quote(t.providerID)}],"kernel":{"id":${quote(t.kernelID)}}},""" +
        s""""finiteCompleteness":{"roleDomainFingerprint":${quote(fingerprint("query-role-domain/v1\n" + role))},""" +
        s""""actionDomainFingerprint":${quote(fingerprint("query-action-domain/v1\n" + array(actions)))}}}"""
