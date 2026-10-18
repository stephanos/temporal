package framework.checker

import framework.dsl.*

final case class Expectation[S](line: Int, text: String, test: S => Boolean)

final class Space[S, O, F] private[checker] (
    val instance: NativeModel[S, O, F],
    val steps: Int,
    val starts: Vector[S],
    val select: (Int, Class) => Boolean
):
  require(steps >= 1, "a space admits at least one step")

private[checker] object Expectations:
  def holds[S, O, F](
      instance: NativeModel[S, O, F],
      expectation: Expectation[S],
      state: S,
      path: Path
  ): Boolean =
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
    def evaluate: Boolean = ClaimEvaluation.within(context)(expectation.test(state))
    instance.stateVars.fold(evaluate)(_.at(state)(evaluate))

private[checker] object Spaces:
  def find[S, O, F](
      space: Space[S, O, F],
      horizon: Int,
      budget: Int,
      target: Expectation[S]
  ): Result =
    require(target != null, "find requires a target")
    require(horizon >= 1, "a find horizon admits at least one step")
    require(horizon <= space.steps, "a find horizon cannot exceed the space's steps")
    val graph = NativeCheck.explore(
      space.instance,
      space.starts,
      budget = budget,
      horizon = horizon,
      positionSelect = space.select,
      positioned = true
    )
    val subject = s"${space.instance.name}.find"
    var witness = Option.empty[Path]
    var failure = Option.empty[NativeCheck.EffectFailure]
    def matches(state: S, path: Path): Boolean =
      try if Expectations.holds(space.instance, target, state, path) then witness = Some(path)
      catch
        case HoleReached(hole) =>
          graph.unknown += s"target at $state reaches ${NativeCheck.holeName(hole)}": Unit
      witness.nonEmpty
    try
      graph.walk(
        onState = i => graph.depths(i) == 0 && matches(graph.states(i), graph.trace(i)),
        onEdge = edge => matches(edge.step.state, graph.through(edge))
      )
    catch case thrown: NativeCheck.EffectFailure => failure = Some(thrown)
    val verdict =
      if failure.nonEmpty then Verdict.refuted
      else if witness.nonEmpty then Verdict.found
      else if graph.complete then Verdict.notFound
      else Verdict.unknown
    val detail = failure.fold(graph.unknown.toVector)(thrown =>
      Vector(s"${NativeCheck.shown(thrown.action)} at ${thrown.before}: ${thrown.cause.getMessage}")
    )
    new Result(
      subject,
      verdict,
      budget,
      witness.fold(graph.depth)(_.steps.size),
      graph.complete,
      witness.orElse(failure.map(_.path)).getOrElse(Path.empty),
      detail,
      None
    )
