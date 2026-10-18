package framework.checker

import framework.dsl.*

object SuiteExecution:
  def run[M <: Machine[?, ?]](body: RunBody[M], budget: Int): Result =
    NativeRuntime.installed.runBody(body, budget).asInstanceOf[Result]

  private[framework] def evaluate[M <: Machine[?, ?]](body: RunBody[M], budget: Int): Result =
    require(budget >= 1, "a budget admits at least one state")
    val model = body.model
    var failed = Option.empty[RunPosition]
    val starts = body.from match
      case Some(start) => Seq(start())
      case None        =>
        val initial = model.initialStates
        require(
          initial.size == 1,
          s"${body.position}: an authored run with several starts requires from"
        )
        initial
    val operations = body.operations.flatMap:
      case step: RunStep =>
        step.targets.foreach: target =>
          target.owners.foreach: (position, owner) =>
            require(
              owner().eq(model),
              s"$position: an input of another Model instance cannot read this Model's state"
            )
        val count = step.count()
        require(count >= 1, s"${step.position}: repeat requires a positive count")
        Vector(
          ReplayRepeatedStep(
            ReplayDeferredStep[State](
              step.targets.map(target => (target.position, () => target.evaluate()))
            ),
            count
          )
        )
      case conditions: RunAssertions =>
        conditions.assertions.map(assertion =>
          ReplayExpectation(
            Expectation[State](
              assertion.position.line,
              s"${assertion.position.file}: ${assertion.position.text}",
              _ =>
                failed = None
                assertion.owners.foreach: (position, owner) =>
                  require(
                    owner().eq(model),
                    s"$position: an expectation of another Model instance cannot read this Model's state"
                  )
                val evaluation = new RunAssertionEvaluation:
                  def test(position: RunPosition, condition: () => Boolean): Boolean =
                    if failed.nonEmpty then false
                    else
                      val holds = condition()
                      if !holds && failed.isEmpty then failed = Some(position)
                      holds
                try
                  evaluation.test(
                    assertion.position,
                    () => assertion.evaluate(evaluation)
                  ) && failed.isEmpty
                catch case _: HoleReached if failed.nonEmpty => false
            )
          )
        )
    val result = Replay.run(model, ReplayScript(operations, starts), budget)
    result.expectation
      .zip(failed)
      .headOption
      .fold(result): (failure, position) =>
        val text = s"${position.file}: ${position.text}"
        new Result(
          result.subject,
          result.verdict,
          result.budget,
          result.depth,
          result.complete,
          result.path,
          result.detail.map(detail =>
            if detail.startsWith("fails at line ") then s"fails at line ${position.line}: $text"
            else detail
          ),
          result.longest,
          expectation = Some(failure.copy(line = position.line, text = text))
        )
