package framework.checker

import framework.dsl.*

sealed trait ReplayOperation[S]
final case class ReplayStep[S](target: Any, alternatives: Any*) extends ReplayOperation[S]
final case class ReplayExpectation[S](expectation: Expectation[S]) extends ReplayOperation[S]

final case class ReplayScript[S](
    operations: Vector[ReplayOperation[S]],
    starts: Seq[S] = Seq.empty
)

private[checker] object Replay:
  def run[S, O, F](
      instance: NativeModel[S, O, F],
      script: ReplayScript[S],
      budget: Int
  ): Result =
    require(budget >= 1, "a budget admits at least one state")
    NativeCheck.admitRoots(instance, script.starts)
    val engine = NativeCheck.explore(instance, Seq.empty, budget = budget)
    var retainedStates = 0
    var cut = false
    def admit(): Boolean =
      if retainedStates >= budget then
        cut = true
        false
      else
        retainedStates += 1
        true
    var frontier = (if script.starts.isEmpty then instance.initialStates else script.starts)
      .filter(_ => admit())
      .map(state => (state, Path(Some(state), Vector.empty), true))
      .toVector
    var failure = Option.empty[Vector[String]]
    var failedExpectation = Option.empty[ExpectationFailure]
    var evidence = frontier.headOption.fold(Path.empty)(_._2)
    var depth = 0
    val boundClasses = instance.nativeEffects.flatMap((decl, _) => NativeCheck.classesOf(decl))
    val operations = script.operations.iterator
    while failure.isEmpty && operations.hasNext do
      operations.next() match
        case ReplayExpectation(expectation) =>
          require(expectation != null, "replay requires an expectation")
          val retained = Vector.newBuilder[(S, Path, Boolean)]
          frontier.iterator
            .takeWhile(_ => failure.isEmpty)
            .foreach: (state, path, admitted) =>
              try
                if Expectations.holds(instance, expectation, state, path) then
                  retained += ((state, path, admitted))
                else
                  failedExpectation = Some(
                    ExpectationFailure(path.steps.size, expectation.line, expectation.text, state)
                  )
                  failure = Some(
                    Vector(
                      s"in $state",
                      s"fails at line ${expectation.line}: ${expectation.text}"
                    )
                  )
                  evidence = path
              catch
                case HoleReached(hole) =>
                  engine.unknown += s"expectation at $state reaches ${NativeCheck.holeName(hole)}": Unit
          frontier = retained.result()
        case operation: ReplayStep[?] =>
          val selected = (operation.target +: operation.alternatives).flatMap: target =>
            val classes = boundClasses.filter(ClaimEvaluation.selects(target, _))
            require(classes.nonEmpty, s"replay pins $target, which no rule binds")
            classes
          val reached = Vector.newBuilder[(S, Path, Boolean)]
          frontier.iterator
            .filter(_._3)
            .takeWhile(_ => failure.isEmpty && !cut)
            .foreach: (before, path, _) =>
              selected.iterator
                .takeWhile(_ => failure.isEmpty && !cut)
                .foreach: cls =>
                  val holes = engine.unknown.size
                  try
                    val outcomes = engine
                      .steps(
                        before,
                        selectClass = candidate => ClaimEvaluation.selects(cls, candidate),
                        prefix = path
                      )
                      .buffered
                    if outcomes.isEmpty && engine.unknown.size == holes then
                      failure = Some(
                        Vector(
                          s"disabled step ${path.steps.size + 1}: ${NativeCheck.shown(cls)}",
                          s"in $before"
                        )
                      )
                      evidence = path
                    outcomes
                      .takeWhile(_ => failure.isEmpty && !cut)
                      .foreach: (_, step) =>
                        val next = path.append(cls, before, step)
                        depth = depth.max(next.steps.size)
                        evidence = next
                        reached += ((step.state, next, admit()))
                  catch
                    case thrown: NativeCheck.EffectFailure =>
                      failure = Some(
                        Vector(
                          s"${NativeCheck.shown(thrown.action)} at ${thrown.before}: ${thrown.cause.getMessage}"
                        )
                      )
                      evidence = path
          frontier = reached.result()
    val complete = failure.isEmpty && !cut && engine.unknown.isEmpty
    val verdict =
      if failure.nonEmpty then Verdict.refuted
      else if complete then Verdict.verified
      else Verdict.unknown
    new Result(
      s"${instance.name}.replay",
      verdict,
      budget,
      depth,
      complete,
      evidence,
      failure.getOrElse(Vector.empty) ++ engine.unknown,
      None,
      expectation = failedExpectation
    )
