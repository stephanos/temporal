package framework.checker

import framework.dsl.{Checker as _, *}

import scala.collection.mutable
import scala.util.Random

private[checker] object Sampling:
  final private case class Draw(
      path: Path,
      complete: Boolean,
      detail: Vector[String],
      failure: Option[Result],
      exhausted: Boolean = false
  )

  def run[S, O, F](space: Space[S, O, F], count: Int, seed: Long, budget: Int): Result =
    require(count >= 1, "a sample count is at least one")
    require(budget >= 1, "a budget admits at least one state")
    val machine = space.instance
    val random = Random(seed)
    val execution = NativeCheck.explore(machine, Seq.empty[S], budget = budget)
    val classes = machine.nativeEffects.flatMap((decl, _) => NativeCheck.classesOf(decl))
    val reached = mutable.HashSet.empty[S]
    def admit(state: S): Boolean =
      if reached(state) then true
      else if reached.size >= budget then false
      else
        reached += state: Unit
        true
    def draw(): Draw =
      if space.starts.isEmpty then
        Draw(Path.empty, false, Vector("no initial state to sample"), None)
      else
        var state = space.starts(random.nextInt(space.starts.size))
        var path = Path(Some(state), Vector.empty)
        var complete = admit(state)
        var stopped = !complete
        var detail = Vector.empty[String]
        var failure = Option.empty[Result]
        var exhausted = false
        while !stopped && path.steps.size < space.steps do
          val position = path.steps.size
          val candidates = random.shuffle(classes.filter(space.select(position, _)))
          var chosen = Option.empty[(Class, Step[S, O, F])]
          val iterator = candidates.iterator
          while !stopped && chosen.isEmpty && iterator.hasNext do
            val cls = iterator.next()
            val beforeUnknown = execution.unknown.size
            try
              val options = execution.steps(state, selectClass = _ == cls, prefix = path).toVector
              if options.nonEmpty then chosen = Some(options(random.nextInt(options.size)))
              else if execution.unknown.size > beforeUnknown then
                complete = false
                stopped = true
                detail ++= execution.unknown.drop(beforeUnknown).map(u => s"unknown: $u")
            catch
              case error: NativeCheck.EffectFailure =>
                failure = Some(Checker.effectResult(error, s"${machine.name}.sample", budget))
                complete = false
                stopped = true
          chosen match
            case Some((cls, step)) =>
              path = path.append(cls, state, step)
              state = step.state
              if !admit(state) then
                complete = false
                stopped = true
                detail :+= s"state budget cut at $state"
            case None =>
              exhausted = !stopped
              stopped = true
        Draw(path, complete, detail, failure, exhausted)
    val draws = Vector.fill(count)(draw())
    val checked = draws.map: draw =>
      if draw.path.initial.isEmpty then Vector.empty
      else
        NativeCheck.safetyPath(machine, draw.path, budget).map(Checker.result) ++
          monitors(machine, draw.path, budget) ++ progress(space, draw, classes, budget)
    val failures = draws
      .zip(checked)
      .zipWithIndex
      .flatMap:
        case ((draw, answers), index) =>
          (draw.failure.toVector ++ answers.filter(_.verdict == Verdict.refuted))
            .sortBy(_.path.steps.size)
            .headOption
            .map(index -> _)
    val failing = failures.headOption
    val uncertain =
      draws.exists(!_.complete) || checked.flatten.exists(_.verdict == Verdict.unknown)
    val evidence = SampleEvidence(
      seed,
      draws.map(_.path),
      failing.map(_._1),
      space.steps,
      count,
      draws.forall(_.complete)
    )
    val detail = Vector(
      s"finite experiment: $count seeded walks of at most ${space.steps} steps; no exhaustive safety or unbounded progress conclusion"
    ) ++
      failing.toVector.flatMap(_._2.detail) ++ draws.flatMap(_.detail) ++
      checked.flatten.filter(_.verdict == Verdict.unknown).flatMap(_.detail)
    new Result(
      s"${machine.name}.sample",
      if failing.nonEmpty then Verdict.refuted
      else if uncertain then Verdict.unknown
      else Verdict.verified,
      budget,
      draws.map(_.path.steps.size).max,
      draws.forall(_.complete) && !uncertain,
      failing.fold(draws.last.path)(_._2.path),
      detail,
      None,
      sampling = Some(evidence)
    )

  private def step[S, O, F](value: PathStep): Step[S, O, F] =
    Step(
      value.outcome.asInstanceOf[O],
      value.after.asInstanceOf[S],
      value.facts.asInstanceOf[List[F]]
    ) // scalafix:ok DisableSyntax.asInstanceOf

  private def monitors[S, O, F](
      machine: NativeModel[S, O, F],
      path: Path,
      budget: Int
  ): Vector[Result] =
    NativeCheck
      .section(machine, "monitors")
      .collect:
        case (member, value: Monitor[?, ?, ?, ?]) =>
          val watch =
            value.asInstanceOf[Monitor[S, O, F, Any]] // scalafix:ok DisableSyntax.asInstanceOf
          var state = watch.initial
          var violation = Option.empty[Path]
          var missing = Vector.empty[String]
          def atEnd(at: Int): Unit =
            val current = path.states(at).asInstanceOf[S] // scalafix:ok DisableSyntax.asInstanceOf
            if watch.atEnds && machine.end(current) && watch.violated(state) then
              violation = Some(path.copy(steps = path.steps.take(at)))
          try
            atEnd(0)
            path.steps.zipWithIndex.foreach: (edge, position) =>
              if violation.isEmpty then
                val actual = step[S, O, F](edge)
                state = watch.next(
                  state,
                  edge.before.asInstanceOf[S],
                  actual
                ) // scalafix:ok DisableSyntax.asInstanceOf
                if !watch.atEnds && watch.readAt.forall(_(actual)) && watch.violated(state) then
                  violation = Some(path.copy(steps = path.steps.take(position + 1)))
                if violation.isEmpty then atEnd(position + 1)
          catch
            case HoleReached(hole) =>
              missing :+= s"unknown: ${NativeCheck.holeName(hole)} in monitor $member"
          new Result(
            s"${machine.name}.$member",
            if violation.nonEmpty then Verdict.refuted
            else if missing.nonEmpty then Verdict.unknown
            else Verdict.verified,
            budget,
            path.steps.size,
            missing.isEmpty,
            violation.getOrElse(Path.empty),
            violation.toVector.map(p =>
              s"counterexample: ${p.steps.map(s => NativeCheck.shown(s.action)).mkString(", ")}"
            ) ++ missing,
            None
          )

  private def progress[S, O, F](
      space: Space[S, O, F],
      draw: Draw,
      classes: Vector[Class],
      budget: Int
  ): Vector[Result] =
    val machine = space.instance
    val path = draw.path
    NativeCheck
      .section(machine, "progress")
      .collect:
        case (member, claim: LeadsTo) =>
          val vars = machine.stateVars.getOrElse(
            throw IllegalArgumentException(s"progress claim $member reads vars")
          )
          claim.machine.foreach(ClaimEvaluation.requireOwner(vars, _))
          val scenario = claim.scenario.bound(machine, member)
          val selected =
            if scenario.machine eq machine then NativeCheck.scenarioPath(path, scenario) else None
          var pending = Option.empty[Int]
          var failure = Option.empty[Path]
          var deadlock = false
          var missing = Vector.empty[String]
          selected.foreach: chosen =>
            try
              chosen.states.zipWithIndex.foreach: (state, position) =>
                if failure.isEmpty then
                  val target = vars.at(state)(claim.to())
                  if target then pending = None
                  else
                    if pending.isEmpty && vars.at(state)(claim.from()) then pending = Some(position)
                    if pending.exists(start => claim.within.exists(position - start >= _)) then
                      failure = Some(chosen.copy(steps = chosen.steps.take(position)))
            catch
              case HoleReached(hole) =>
                missing :+= s"unknown: ${NativeCheck.holeName(hole)} in progress $member"
            if pending.nonEmpty && failure.isEmpty && missing.isEmpty && draw.exhausted &&
              chosen.steps.size == path.steps.size &&
              classes.filter(cls => scenario.admits(cls)).forall(space.select(path.steps.size, _))
            then
              failure = Some(chosen)
              deadlock = true
          if pending.nonEmpty && failure.isEmpty then
            missing :+= s"unknown: finite walk ends with an unmet progress obligation for $member"
          if scenario.machine ne machine then
            missing :+= s"unknown: progress $member reads a foreign environment"
          new Result(
            s"${machine.name}.$member",
            if failure.nonEmpty then Verdict.refuted
            else if missing.nonEmpty then Verdict.unknown
            else Verdict.verified,
            budget,
            path.steps.size,
            missing.isEmpty,
            failure.getOrElse(Path.empty),
            failure.toVector.flatMap(_ =>
              if deadlock then Vector("reason: deadlock", s"deadlock: ${path.states.last}")
              else Vector(s"missed deadline: $member")
            ) ++ missing,
            None
          )
