package framework.checker

import framework.dsl.*

sealed trait ReplayOperation[S]
final case class ReplayStep[S](target: Any, alternatives: Any*) extends ReplayOperation[S]

final private[framework] case class ReplayDeferredStep[S](
    targets: () => Vector[Any],
    position: RunPosition
) extends ReplayOperation[S]

final private[framework] case class ReplayRepeatedStep[S](step: ReplayDeferredStep[S], count: Int)
    extends ReplayOperation[S]

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
    def select(targets: Seq[Any], position: Option[RunPosition]): Seq[Class] =
      targets.flatMap: target =>
        val classes = boundClasses.filter(ClaimEvaluation.selects(target, _))
        val location = position.fold("")(value => s"$value: ")
        require(classes.nonEmpty, s"${location}replay pins $target, which no rule binds")
        classes
    def targets(operation: ReplayDeferredStep[?], before: S, path: Path): Vector[Any] =
      val last = path.steps.lastOption
      val origin = last.flatMap(_.before match
        case value: State => Some(value)
        case _            => None)
      val context = ClaimEvaluation.Context(
        origin,
        last.map(_.action),
        last.map(_.outcome),
        last.fold(List.empty[Any])(_.facts)
      )
      def evaluate = ClaimEvaluation.within(context)(operation.targets())
      instance.stateVars.fold(evaluate)(_.at(before)(evaluate))
    val operations = script.operations.iterator.flatMap:
      case repeated: ReplayRepeatedStep[S] =>
        Iterator.fill(repeated.count)(repeated.step).takeWhile(_ => frontier.nonEmpty)
      case operation => Iterator.single(operation)
    while failure.isEmpty && operations.hasNext do
      operations.next() match
        case _: ReplayRepeatedStep[?] =>
          throw IllegalStateException("replay repetition was not expanded")
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
        case operation @ (_: ReplayStep[?] | _: ReplayDeferredStep[?]) =>
          val eager = operation match
            case step: ReplayStep[?] => Some(select(step.target +: step.alternatives, None))
            case _                   => None
          val reached = Vector.newBuilder[(S, Path, Boolean)]
          frontier.iterator
            .filter(_._3)
            .takeWhile(_ => failure.isEmpty && !cut)
            .foreach: (before, path, _) =>
              val selected = eager.getOrElse:
                operation match
                  case step: ReplayDeferredStep[?] =>
                    val values =
                      try Some(targets(step, before, path))
                      catch
                        case HoleReached(hole) =>
                          engine.unknown += s"${step.position}: inputs at $before reach ${NativeCheck.holeName(hole)}": Unit
                          None
                        case scala.util.control.NonFatal(cause) =>
                          failure = Some(
                            Vector(s"${step.position}: inputs at $before: ${cause.getMessage}")
                          )
                          evidence = path
                          None
                    values.fold(Seq.empty[Class])(select(_, Some(step.position)))
                  case _ => Seq.empty
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
