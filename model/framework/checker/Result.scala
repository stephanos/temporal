package framework.checker

import framework.dsl.{Class, Step}

enum Verdict derives CanEqual:
  case verified, refuted, found, notFound, unknown

final case class PathStep(action: Class, before: Any, after: Any, outcome: Any, facts: List[Any])
    derives CanEqual

final case class Path(initial: Option[Any], steps: Vector[PathStep]) derives CanEqual:
  def states: Vector[Any] = initial.toVector ++ steps.map(_.after)
  private[checker] def append[S, O, F](action: Class, before: S, step: Step[S, O, F]): Path =
    Path(
      initial.orElse(Some(before)),
      steps :+ PathStep(action, before, step.state, step.outcome, step.facts)
    )

object Path:
  val empty: Path = Path(None, Vector.empty)

final class Result private[checker] (
    val subject: String,
    val verdict: Verdict,
    val budget: Int,
    val depth: Int,
    val complete: Boolean,
    val path: Path,
    val detail: Vector[String],
    val longest: Option[Int],
    private[checker] val checking: Option[() => Report] = None
):
  override def toString: String =
    val completion = if complete then "complete" else "cut"
    (s"$subject: $verdict (budget $budget states, $depth steps; exploration $completion)" +: detail)
      .mkString("\n  ")

final case class Report(results: Vector[Result], warnings: Vector[String]):
  def verdicts: Map[String, Verdict] =
    results.map(r => r.subject.split('.').last -> r.verdict).toMap
  def result(member: String): Result = results
    .find(_.subject.endsWith(s".$member"))
    .getOrElse(
      throw java.util.NoSuchElementException(s"no check named $member")
    )
  override def toString: String = (results.map(_.toString) ++ warnings).mkString("\n")
