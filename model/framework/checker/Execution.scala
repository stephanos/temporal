package framework.checker

import framework.dsl.{Checker as _, *}

final class Execution private[checker] (val budget: Int):
  inline def explore[S, O, F](
      inline instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty
  ): Result = Checker.exploreHeld(CheckerNaming.bind(instance), starts, budget)

  inline def verifyAll[S, O, F](
      inline instance: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty
  ): Report = Checker.verifyHeld(CheckerNaming.bind(instance), starts, budget)

  inline def replay[S, O, F](
      inline instance: NativeModel[S, O, F],
      path: ReplayScript[S]
  ): Result = Checker.replayHeld(CheckerNaming.bind(instance), path, budget)

  def find[S, O, F](space: Space[S, O, F], horizon: Int)(target: Expectation[S]): Result =
    Checker.find(space, horizon, budget)(target)

  def sample[S, O, F](space: Space[S, O, F], count: Int, seed: Long): Result =
    Checker.sample(space, count, seed, budget)

  inline def refuted[S, O, F](
      inline instance: NativeModel[S, O, F],
      inline claim: Any,
      starts: Seq[S] = Seq.empty
  ): Result =
    Checker.refutedHeld(CheckerNaming.bind(instance), framework.dsl.verify(claim), starts, budget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M]
  ): Result = Checker.refutedMonitorHeld(CheckerNaming.bind(instance), claim, Seq.empty, budget)

  inline def refuted[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      claim: Monitor[S, O, F, M],
      starts: Seq[S]
  ): Result = Checker.refutedMonitorHeld(CheckerNaming.bind(instance), claim, starts, budget)

  inline def monitor[S, O, F, M](
      inline instance: NativeModel[S, O, F],
      watch: Monitor[S, O, F, M],
      starts: Seq[S]
  ): Result = Checker.monitorHeld(CheckerNaming.bind(instance), watch, starts, budget)

  inline def progress[S, O, F](
      inline instance: NativeModel[S, O, F],
      claim: LeadsTo,
      starts: Seq[S]
  ): Result = Checker.progressHeld(CheckerNaming.bind(instance), claim, starts, budget)
