package framework.checker

import framework.dsl.{LeadsTo, Monitor, NativeModel, Progress}

object Checker:
  val defaultBudget: Int = NativeCheck.defaultBudget

  def explore[S, O, F](
      instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget
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

  def verifyAll[S, O, F](
      instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget
  ): Report = report(NativeCheck.verifyAll(instance, starts, budget))

  def monitor[S, O, F, M](
      instance: NativeModel[S, O, F],
      watch: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int = defaultBudget
  ): Result = result(
    NativeCheck.monitor(
      instance,
      s"${instance.name}.${NativeCheck.claimName(watch, instance)}",
      watch,
      starts,
      budget
    )
  )

  def progress[S, O, F](
      instance: NativeModel[S, O, F],
      claim: LeadsTo,
      starts: Seq[S],
      budget: Int = defaultBudget
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

  def progress[S, O, F](
      claim: Progress[S],
      instance: NativeModel[S, O, F],
      starts: Seq[S],
      budget: Int
  ): Result = result(NativeCheck.progress(claim, instance, starts, budget))

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
