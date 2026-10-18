package framework.checker

import framework.dsl.*

object Checker:
  val defaultBudget: Int = NativeCheck.defaultBudget

  inline def explore[S, O, F](
      inline instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget
  ): Result = exploreHeld(CheckerNaming.bind(instance), starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def exploreHeld[S, O, F](
      instance: NativeModel[S, O, F],
      starts: Seq[S],
      budget: Int
  ): Result =
    require(budget >= 1, "a budget admits at least one state")
    val origins = if starts.isEmpty then instance.initialStates.toVector else starts.toVector
    val graph = NativeCheck.explore(instance, origins, budget = budget)
    try
      graph.drain()
      new Result(
        instance.name,
        if graph.complete then Verdict.verified else Verdict.unknown,
        budget,
        graph.depth,
        graph.complete,
        Path.empty,
        graph.unknown.toVector,
        None,
        Some(() =>
          report(NativeCheck.verifyAll(instance, origins, budget, exploration = Some(graph)))
        )
      )
    catch
      case failure: NativeCheck.EffectFailure =>
        val broken = effectResult(failure, instance.name, budget)
        new Result(
          broken.subject,
          broken.verdict,
          budget,
          broken.depth,
          false,
          broken.path,
          broken.detail,
          None,
          Some(() => Report(Vector(broken), Vector.empty))
        )

  def verifyAll(graph: Result): Report = graph.checking.getOrElse(
    throw IllegalArgumentException("verifyAll requires the Result of Checker.explore")
  )()

  inline def verifyAll[S, O, F](
      inline instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget
  ): Report = verifyHeld(CheckerNaming.bind(instance), starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def verifyHeld[S, O, F](
      instance: NativeModel[S, O, F],
      starts: Seq[S],
      budget: Int
  ): Report =
    try report(NativeCheck.verifyAll(instance, starts, budget))
    catch
      case failure: NativeCheck.EffectFailure =>
        Report(Vector(effectResult(failure, instance.name, budget)), Vector.empty)

  inline def space[S, O, F](
      inline instance: NativeModel[S, O, F],
      steps: Int,
      starts: Seq[S] = Seq.empty
  )(select: (Int, Class) => Boolean = (_, _) => true): Space[S, O, F] =
    spaceHeld(CheckerNaming.bind(instance), steps, starts, select)

  @scala.annotation.publicInBinary
  private[framework] def spaceHeld[S, O, F](
      instance: NativeModel[S, O, F],
      steps: Int,
      starts: Seq[S],
      select: (Int, Class) => Boolean
  ): Space[S, O, F] =
    new Space(
      instance,
      steps,
      (if starts.isEmpty then instance.initialStates else starts).toVector,
      select
    )

  inline def replay[S, O, F](
      inline instance: NativeModel[S, O, F],
      path: ReplayScript[S],
      budget: Int = defaultBudget
  ): Result = replayHeld(CheckerNaming.bind(instance), path, budget)

  @scala.annotation.publicInBinary
  private[framework] def replayHeld[S, O, F](
      instance: NativeModel[S, O, F],
      path: ReplayScript[S],
      budget: Int
  ): Result = Replay.run(instance, path, budget)

  def find[S, O, F](space: Space[S, O, F], horizon: Int, budget: Int = defaultBudget)(
      target: Expectation[S]
  ): Result = Spaces.find(space, horizon, budget, target)

  def sample[S, O, F](
      space: Space[S, O, F],
      count: Int,
      seed: Long,
      budget: Int = defaultBudget
  ): Result = Sampling.run(space, count, seed, budget)

  inline def refuted[S, O, F](
      inline instance: NativeModel[S, O, F],
      inline claim: Any,
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget
  ): Result = refutedHeld(CheckerNaming.bind(instance), framework.dsl.verify(claim), starts, budget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M]
  ): Result = refutedMonitorHeld(CheckerNaming.bind(instance), claim, Seq.empty, defaultBudget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M],
      budget: Int
  ): Result = refutedMonitorHeld(CheckerNaming.bind(instance), claim, Seq.empty, budget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M],
      starts: Seq[S]
  ): Result = refutedMonitorHeld(CheckerNaming.bind(instance), claim, starts, defaultBudget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int
  ): Result = refutedMonitorHeld(CheckerNaming.bind(instance), claim, starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def refutedMonitorHeld[S, O, F, M](
      instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int
  ): Result =
    monitorHeld(instance, claim, if starts.isEmpty then instance.initialStates else starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def refutedHeld[S, O, F](
      instance: NativeModel[S, O, F],
      claim: ClaimQuery,
      starts: Seq[S],
      budget: Int
  ): Result =
    require(budget >= 1, "a budget admits at least one state")
    val subject = s"${instance.name}.${claim.property.name}"
    guarded(subject, budget):
      val environment = if claim.scenario.machine eq claim.property.machine then
        claim.scenario.copy(machine = instance)
      else claim.scenario
      val origins =
        environment.start.fold(if starts.isEmpty then instance.initialStates else starts)(s =>
          Seq(s.asInstanceOf[S]) // scalafix:ok DisableSyntax.asInstanceOf
        )
      if environment.machine ne instance then
        new Result(
          subject,
          Verdict.unknown,
          budget,
          0,
          false,
          Path.empty,
          Vector("claim reads a foreign environment"),
          None
        )
      else if claim.property.leads.nonEmpty then
        result(
          NativeCheck
            .progress(instance, subject, claim.property.leads.get, environment, origins, budget)
        )
      else if environment.free then
        result(
          NativeCheck.verify(
            NativeCheck.explore(instance, origins, environment.admits, budget, environment.admits),
            subject,
            claim.property
          )
        )
      else
        result(
          NativeCheck
            .pinned(
              instance,
              subject,
              ClaimQuery(QueryForm.verify, claim.property, environment),
              origins,
              budget
            )
            .results
            .head
        )

  def forSuite(policy: BudgetPolicy, name: String): Execution =
    new Execution(policy.forSuite(name))

  inline def monitor[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      watch: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int = defaultBudget
  ): Result = monitorHeld(CheckerNaming.bind(instance), watch, starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def monitorHeld[S, O, F, M](
      instance: NativeModel[S, O, F],
      watch: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int
  ): Result = guarded(instance.name, budget)(
    result(
      NativeCheck.monitor(
        instance,
        s"${instance.name}.${NativeCheck.claimName(watch, instance)}",
        watch,
        starts,
        budget
      )
    )
  )

  inline def progress[S, O, F](
      inline instance: NativeModel[S, O, F],
      claim: LeadsTo,
      starts: Seq[S],
      budget: Int = defaultBudget
  ): Result = progressHeld(CheckerNaming.bind(instance), claim, starts, budget)

  @scala.annotation.publicInBinary
  private[framework] def progressHeld[S, O, F](
      instance: NativeModel[S, O, F],
      claim: LeadsTo,
      starts: Seq[S],
      budget: Int
  ): Result = guarded(instance.name, budget)(
    result(
      NativeCheck.progress(
        instance,
        s"${instance.name}.${claim.name}",
        claim,
        claim.scenario.bound(instance, claim.name),
        starts,
        budget
      )
    )
  )

  private[checker] def effectResult(
      failure: NativeCheck.EffectFailure,
      subject: String,
      budget: Int
  ): Result = new Result(
    subject,
    Verdict.refuted,
    budget,
    failure.path.steps.size,
    false,
    failure.path,
    Vector(
      s"${NativeCheck.shown(failure.action)} at ${failure.before}: ${failure.cause.getMessage}"
    ),
    None
  )

  private def guarded(subject: String, budget: Int)(body: => Result): Result =
    try body
    catch case failure: NativeCheck.EffectFailure => effectResult(failure, subject, budget)

  private def report(value: NativeCheck.Report): Report =
    Report(value.results.map(result), value.warnings)

  private[checker] def result(value: NativeCheck.Result): Result =
    val verdict = value.verdict match
      case NativeCheck.Verdict.verified => Verdict.verified
      case NativeCheck.Verdict.found    => Verdict.found
      case NativeCheck.Verdict.unknown  => Verdict.unknown
      case _                            => Verdict.refuted
    val detail = value.verdict match
      case NativeCheck.Verdict.deadlock | NativeCheck.Verdict.cycle |
          NativeCheck.Verdict.missedDeadline =>
        s"reason: ${value.verdict.shown}" +: value.detail
      case _ => value.detail
    new Result(
      value.subject,
      verdict,
      value.budget,
      value.depth,
      value.complete,
      value.path,
      detail,
      value.longest
    )
