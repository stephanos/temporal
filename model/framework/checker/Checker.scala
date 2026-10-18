package framework.checker

import framework.dsl.{LeadsTo, Monitor, NativeModel}

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
  ): Report = report(NativeCheck.verifyAll(instance, starts, budget))

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
  ): Result = result(
    NativeCheck.monitor(
      instance,
      s"${instance.name}.${NativeCheck.claimName(watch, instance)}",
      watch,
      starts,
      budget
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
  ): Result = result(
    NativeCheck.progress(
      instance,
      s"${instance.name}.${claim.name}",
      claim,
      claim.scenario.bound(instance, claim.name),
      starts,
      budget
    )
  )

  private def report(value: NativeCheck.Report): Report =
    Report(value.results.map(result), value.warnings)

  private def result(value: NativeCheck.Result): Result =
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
