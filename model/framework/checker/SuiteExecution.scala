package framework.checker

import framework.dsl.{Checker as _, *}

object SuiteExecution:
  private[framework] def forSuite(exactName: String): NativeSuiteExecution =
    forSuite(exactName, BudgetPolicy.fromEnvironment(sys.env))

  private[checker] def forSuite(
      exactName: String,
      policy: BudgetPolicy
  ): NativeSuiteExecution =
    val execution = Checker.forSuite(policy, exactName)
    new NativeSuiteExecution:
      val budget: Int = execution.budget

      def claims[M <: Machine[?, ?]](model: M): SuiteCheck =
        val report = Checker.verifyHeld(model, Seq.empty, budget)
        SuiteCheck(
          report,
          report.results.forall(_.verdict == Verdict.verified),
          report.results.exists(_.verdict == Verdict.unknown),
          (report.results.map(diagnostics) ++ report.warnings).mkString("\n")
        )

      def replay[M <: Machine[?, ?]](body: RunBody[M]): SuiteCheck =
        val result = evaluate(body, budget)
        SuiteCheck(
          result,
          result.verdict == Verdict.verified,
          result.verdict == Verdict.unknown,
          diagnostics(result)
        )

      def find[M <: Machine[?, ?]](
          model: M,
          request: SpaceFind[M],
          from: Option[State]
      ): SuiteCheck =
        require(
          request.steps >= 1 && request.steps <= request.space.steps,
          s"${request.position}: a find horizon must be within 1..${request.space.steps}"
        )
        require(
          request.target.assertions.nonEmpty,
          s"${request.position}: find requires an expect target"
        )
        val descriptor = request.space.acquire(model, from)
        request.target.assertions.foreach: assertion =>
          assertion.owners.foreach: (position, owner) =>
            require(
              owner().eq(model),
              s"$position: an expectation belongs to another Model instance"
            )
        val space = Checker.spaceHeld(model, descriptor.steps, descriptor.starts, descriptor.select)
        var failed = Option.empty[(RunPosition, State)]
        val target = Expectation[State](
          request.position.line,
          request.position.text,
          state =>
            failed = None
            val evaluation = new RunAssertionEvaluation:
              def test(position: RunPosition, condition: () => Boolean): Boolean =
                if failed.nonEmpty then false
                else
                  val holds = condition()
                  if !holds && failed.isEmpty then failed = Some(position -> state)
                  holds
            request.target.assertions.forall: assertion =>
              try
                evaluation.test(
                  assertion.position,
                  () => assertion.evaluate(evaluation)
                ) && failed.isEmpty
              catch case _: HoleReached if failed.nonEmpty => false
        )
        val result = execution.find(space, request.steps)(target)
        val positioned = if result.verdict == Verdict.notFound then
          failed.toVector.map((position, state) => s"target fails at $position, state $state")
        else Vector.empty
        SuiteCheck(
          result,
          result.verdict == Verdict.found,
          result.verdict == Verdict.unknown,
          (Vector(diagnostics(result)) ++ positioned).mkString("\n")
        )

      def sample[M <: Machine[?, ?]](
          model: M,
          request: SpaceSample[M],
          from: Option[State]
      ): SuiteCheck =
        require(request.count >= 1, s"${request.position}: a sample count is at least one")
        val descriptor = request.space.acquire(model, from)
        val space = Checker.spaceHeld(model, descriptor.steps, descriptor.starts, descriptor.select)
        val result = execution.sample(space, request.count, request.seed)
        SuiteCheck(
          result,
          result.verdict == Verdict.verified && result.complete,
          result.verdict == Verdict.unknown,
          diagnostics(result)
        )

  private def diagnostics(result: Result): String =
    val initial = result.path.initial.map(state => s"initial: $state").toVector
    val steps = result.path.steps.zipWithIndex.map: (step, index) =>
      s"step ${index + 1}: ${NativeCheck.shown(step.action)} from ${step.before} to ${step.after}, outcome ${step.outcome}, facts ${step.facts.mkString("[", ", ", "]")}"
    val expectation = result.expectation
      .map(failure =>
        s"expectation at line ${failure.line}: ${failure.text}, state ${failure.state}"
      )
      .toVector
    (Vector(result.toString) ++ initial ++ steps ++ expectation).mkString("\n")

  def run[M <: Machine[?, ?]](body: RunBody[M], budget: Int): Result =
    NativeRuntime.installed.runBody(body, budget).asInstanceOf[Result]

  private[framework] def evaluate[M <: Machine[?, ?]](
      body: RunBody[M],
      budget: Int
  ): Result =
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
        require(
          count >= 1,
          s"${step.position}: repeat requires a positive count"
        )
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
                  def test(
                      position: RunPosition,
                      condition: () => Boolean
                  ): Boolean =
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
