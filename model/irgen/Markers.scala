package umpire.irgen

import io.temporal.server.api.umpire.v1 as ir

/**
 * What a machine is for, as its markers say (umpire.FailureModel, umpire.NegativeControl), held to
 * what one IR file lifts with it (fn-126 decision 20). The markers are transparent: nothing here
 * changes the IR, so a marker is read from the object that declares the machine.
 *
 *   - A negative control is a deliberately wrong design the checks must refuse. A Query lifted with
 *     it asks something that can refute it: a `verify`, whose answer may be a counterexample, or a
 *     Query whose Run is expected to violate its Property. No machine refines it (a composition
 *     puts a provider in place only of a machine it refines), and it declares no refinement of its
 *     own, as a feature's System does: it is no feature's Product or System.
 *   - A failure model is the real design under a fault the environment can cause. It binds a fault,
 *     an action of the party `fault` or of a `faults` section (a composition binds its members'),
 *     and its promise holds: its Queries expect it to, unless one declares otherwise, so not every
 *     Query of it expects its Run violated.
 *   - A machine that binds a fault says what it is for: it is marked one or the other.
 *
 * A fault an action is bound to is one some state enables: an action `disabled` binds no fault.
 */
/**
 * A negative control's call on a run's Queries: its name, where its object is, and whether a Query
 * of one lift can refute it.
 */
final private[irgen] case class Refutation(control: String, at: String, refuted: Boolean)

/**
 * The refusal of each negative control no Query of the run can refute, once.
 */
private[irgen] def unrefuted(all: Seq[Refutation]): Seq[LiftError] =
  all
    .groupBy(_.control)
    .toSeq
    .sortBy(_._1)
    .collect {
      case (name, rs) if !rs.exists(_.refuted) =>
        LiftError(
          rs.head.at,
          s"$name is a negative control that no Query of the run can refute: ask a `verify` of it, " +
            "whose answer may be a counterexample, or a Query whose Run is expected violated"
        )
    }

private[irgen] trait Markers:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  /**
   * Each refusal of the markers of what this lift lifted, at the declaring object's position, and
   * what each negative control asks of the run's Queries: whether a Query this lift lifted can
   * refute it. A run holds the whole run's Queries to it (Lift.scala): the gate's, every negative
   * control any IR file lifts; a lift of roots, the ones its Queries ask about, since a fixture may
   * lift a control as a composition's member alone.
   */
  def markerRefusals(everyLifted: Boolean): (Seq[LiftError], Seq[Refutation]) =
    val refused = Seq.newBuilder[LiftError]
    val refutations = Seq.newBuilder[Refutation]
    def refuse(at: Option[ir.Position], message: String): Unit =
      val p = at.getOrElse(ir.Position())
      refused += LiftError(s"${p.file}:${p.line}", message)

    // The object each lifted machine and composition is declared by, where it is one.
    def objectOf(key: String): Option[Symbol] =
      defs.keys.find(s => s.fullName == key && objectForm(s)).map(moduleClassOf)
    def marked(cls: Option[Symbol], marker: Symbol): Boolean =
      cls.exists(_.typeRef.derivesFrom(marker))

    val faults = faultActions
    // The actions a machine binds that some state enables: its rules' actions, or, where it binds
    // by step functions, every one.
    def fired(m: ir.Machine): Seq[String] =
      val bound = m.steps.map(_.action)
      rulesOf.get(m.name).fold(bound)(ruled => bound.filter(ruled.contains))
    def faultsOf(m: ir.Machine): Seq[String] = fired(m).filter(faults).distinct
    val byName = machines.values.map(m => m.name -> m).toMap
    def compositionFaults(c: ir.Composition): Seq[String] =
      c.members.flatMap(member => byName.get(member.machine).toSeq.flatMap(faultsOf)).distinct

    def over(name: String): Seq[ir.Query] =
      queries.values.filter(_.getScenario.machine == name).toSeq
    def violated(q: ir.Query): Boolean =
      q.expectedRun.exists(_.property == ir.RunExpectation.Outcome.OUTCOME_VIOLATED)
    def refutable(q: ir.Query): Boolean = q.form == ir.Query.Form.FORM_VERIFY || violated(q)

    def negativeControl(name: String, at: Option[ir.Position], cls: Option[Symbol]): Unit =
      if everyLifted || over(name).nonEmpty then
        val p = at.getOrElse(ir.Position())
        refutations += Refutation(name, s"${p.file}:${p.line}", over(name).exists(refutable))
      for m <- machines.values if m.refines.exists(_.product == name) do
        refuse(
          at,
          s"$name is a negative control, and ${m.name} refines it: a deliberately wrong design is " +
            "no machine's Product, so nothing refines it"
        )
      for c <- cls; tree <- scala.util.Try(c.tree).toOption do
        tree match
          case body: ClassDef if sectionOf(body, "refinement").nonEmpty =>
            refuse(
              at,
              s"$name is a negative control, and declares a refinement of its own, as a feature's " +
                "System does: a deliberately wrong design derives from the design it gets wrong"
            )
          case _ => ()

    def failureModel(name: String, at: Option[ir.Position], bound: Seq[String]): Unit =
      if bound.isEmpty then
        refuse(
          at,
          s"$name is a failure model and binds no fault: bind an action of the party `fault` or " +
            "of a `faults` section that some state enables"
        )
      val asked = over(name)
      if asked.nonEmpty && asked.forall(violated) then
        refuse(
          at,
          s"$name is a failure model, and every Query of it expects its Run to violate the " +
            "promise: a failure model's promise holds under its fault; a design the checks must " +
            "refuse is a negative control"
        )

    for (key, m) <- machines do
      val cls = objectOf(key)
      val failure = marked(cls, failureModelClass)
      val negative = marked(cls, negativeControlClass)
      if failure && negative then
        refuse(m.position, s"${m.name} is marked a failure model and a negative control: it is one")
      if negative then negativeControl(m.name, m.position, cls)
      if failure then failureModel(m.name, m.position, faultsOf(m))
      // A machine the builder declares has no object to mark; the builder is being retired.
      if cls.nonEmpty && !failure && !negative && faultsOf(m).nonEmpty then
        refuse(
          m.position,
          s"${m.name} binds the fault ${faultsOf(m).map(actions(_).name).mkString(", ")} and is " +
            "marked neither a FailureModel, the real design under the fault, nor a " +
            "NegativeControl, a deliberately wrong one: mix in the one it is"
        )
    for (key, c) <- compositions do
      val cls = objectOf(key)
      val failure = marked(cls, failureModelClass)
      val negative = marked(cls, negativeControlClass)
      if failure && negative then
        refuse(c.position, s"${c.name} is marked a failure model and a negative control: it is one")
      if negative then negativeControl(c.name, c.position, None)
      if failure then failureModel(c.name, c.position, compositionFaults(c))
    (refused.result(), refutations.result())

  /**
   * The Definition IDs of the faults: the actions of the party `fault`, and every action declared in
   * a `faults` section.
   */
  private def faultActions: Set[String] =
    val inSections = defs.toSeq.flatMap {
      case (sym, v: ValDef)
          if sym.exists && isSection(sym.maybeOwner) &&
            sym.maybeOwner.name
              .stripSuffix("$") == "faults" && isNamed(v.tpt.tpe, "umpire.Action") =>
        scala.util.Try(definitionId(sym, v)).toOption
      case _ => None
    }
    (actions.values.filter(_.party == "fault").map(_.id) ++ inSections).toSet
