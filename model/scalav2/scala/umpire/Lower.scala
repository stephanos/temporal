package umpire

/** What one lowered clause fixes. */
enum RequirementKind:
  case state, outcome, fact

/** One clause a predicate fixes: its label, what it fixes, and the key it fixes. */
final case class Requirement(label: String, kind: RequirementKind, value: String)

/** The clauses one trigger carries: an action key for a same-step claim. */
final case class Group(trigger: String, requirements: Vector[Requirement])

object Lower:
  /**
   * A same-step Property's predicate as the clauses that say the same thing over the machine's own
   * table, as `Umpire.Command.enumerateSameStep` does: over the steps the trigger admits, the
   * predicate fixes a value when every accepted step carries it, the domain has another value, and
   * the predicate rejects every accepted step with it changed. The fixed values must carry the
   * predicate exactly. A predicate the clause language cannot carry is refused with the reason.
   *
   * Only the whole-state, outcome and fact readings are ported, as in Go: the one-field reading a
   * composed claim needs is not, because no realized Query reads a composition.
   */
  def apply(p: PropertyDecl): Checked[Vector[Group]] = checked {
    val decl = s"property ${p.name}"
    if p.isTransition then fail(decl, "a transition claim is searched and verified, never realized")
    val t = p.machine.table.get
    t.actions.filter(p.triggers).map { action =>
      val results = t.rows.filter(_.action == action).flatMap(_.results)
      val trigger = s"`$action`"
      if results.isEmpty then
        fail(
          decl,
          s"no step of this machine is admitted at $trigger, so the predicate has nothing to hold on"
        )
      val reqs = fixedRequirements(p, t, results) match
        case Left(reason) => fail(decl, s"$reason at $trigger")
        case Right(r)     => r
      if reqs.isEmpty then
        fail(
          decl,
          s"the predicate holds on every step of this machine at $trigger and fixes no state, outcome or " +
            "fact, so it claims nothing"
        )
      Group(action, reqs)
    }
  }

  /** `fixedRequirements` without the field reading. */
  private def fixedRequirements(
      p: PropertyDecl,
      t: Table,
      results: Vector[RowResult]
  ): Either[String, Vector[Requirement]] =
    val accepted = results.filter(p.accepts)
    if accepted.isEmpty then Left("the predicate holds on no step of this machine")
    else fixedAmong(p, t, results, accepted)

  /** `fixedRequirements` over the steps the predicate accepts, of which there is at least one. */
  private def fixedAmong(
      p: PropertyDecl,
      t: Table,
      results: Vector[RowResult],
      accepted: Vector[RowResult]
  ): Either[String, Vector[Requirement]] =
    val first = accepted.head
    val alter = t.alter
    val state = first.state
    val outcome = first.outcome
    val stateFixed = t.states.exists(_ != state) && accepted.forall(step =>
      step.state == state && !t.states.exists(o => o != state && p.accepts(alter.state(step, o)))
    )
    val outcomeFixed = t.outcomes.exists(_ != outcome) && accepted.forall(step =>
      step.outcome == outcome && !t.outcomes.exists(o =>
        o != outcome && p.accepts(alter.outcome(step, o))
      )
    )
    val facts = dedupAdjacent(first.facts).filter(f =>
      accepted.forall(step => step.facts.contains(f) && !p.accepts(alter.without(step, f)))
    )
    def carried(step: RowResult) =
      (!stateFixed || step.state == state) && (!outcomeFixed || step.outcome == outcome) && facts
        .forall(step.facts.contains)
    results.find(step => carried(step) != p.accepts(step)) match
      case Some(step) =>
        Left(
          "the predicate is not a conjunction of one state, one outcome and facts: the clauses it fixes cannot " +
            s"tell the step to ${step.state} with outcome ${step.outcome} and facts ${step.facts.mkString("[", " ", "]")} " +
            "apart from the steps it accepts"
        )
      case None =>
        Right(
          Vector.concat(
            Option.when(stateFixed)(Requirement(s"state-$state", RequirementKind.state, state)),
            Option.when(outcomeFixed)(
              Requirement(s"outcome-$outcome", RequirementKind.outcome, outcome)
            ),
            facts.map(f => Requirement(s"fact-$f", RequirementKind.fact, f))
          )
        )

  // Go's `slices.Compact`: adjacent duplicates removed.
  private def dedupAdjacent(xs: Vector[String]): Vector[String] =
    xs.foldLeft(Vector.empty[String])((acc, x) =>
      if acc.lastOption.contains(x) then acc else acc :+ x
    )
