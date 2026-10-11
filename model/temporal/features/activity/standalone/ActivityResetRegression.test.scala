package framework

import framework.Finite.given

import temporal.features.activity.{deadline, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  completionClient,
  pausing,
  product,
  system,
  MaxAttempts,
  ResetPause
}
import temporal.features.activity.standalone.system.{
  ActivitySystem,
}
import framework.outcomes.{Outcome, Rejection}

class ActivityResetRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[system.Phase, system.Phase]]
  private type ActivityStep = Step[system.State, Outcome, system.Fact]

  private def take(s: system.State, c: Class): List[ActivityStep] =
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
    effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)(s, c.values)
      .map(_.copy(because = ""))

  private type Transition = (system.State, ActivityStep) => Boolean

  private def holds(property: Property[system.State], before: system.State, after: ActivityStep) =
    property.decl.holds2 match
      case Some(predicate) =>
        val claim = predicate.asInstanceOf[Transition] // scalafix:ok DisableSyntax.asInstanceOf
        claim(before, after)
      case None =>
        val claim = property.decl.holds.get
          .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        claim(after)

  private val states = summon[Finite[system.State]].values
  private val resets = List(client.reset(ResetPause.resume), client.reset(ResetPause.keepPaused))
  private val pending = List(system.Phase.resetRequested, system.Phase.resetKeepingPause)
  private def restarted(before: system.State): system.State = before.copy(
    phase =
      // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
      if before.phase == system.Phase.resetKeepingPause then
        // scalafix:on DisableSyntax.==
        system.Phase.paused
      else system.Phase.scheduled,
    dispatch = system.Dispatch.now,
    attempts = UpTo(0)
  )

  test("reset is one per-RPC client action whose one input is keep_paused") {
    assertEquals(client.reset.decl.tokens, List(Some(pausing)))
    assertEquals(client.reset.decl.domains.map(_.values.toList), List(ResetPause.values.toList))
    assert(
      ActivitySystem.bindings.exists(_.decl == client.reset.decl)
    ) // scalafix:ok DisableSyntax.==
    assert(
      product.ActivityProduct.bindings.exists(_.decl == client.reset.decl)
    ) // scalafix:ok DisableSyntax.==
    assertEquals(summon[Finite[system.State]].values.size, 5616)
  }

  test("a waiting or paused reset applies at once, clears backoff and lands by keep_paused") {
    for before <- states; c <- resets do
      val after = take(before, c)
      before.phase match
        case system.Phase.scheduled | system.Phase.paused =>
          val keeps =
            before.phase == system.Phase.paused && c == resets(1) // scalafix:ok DisableSyntax.==
          val dispatch =
            // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
            if before.dispatch == system.Dispatch.backoff then
              // scalafix:on DisableSyntax.==
              system.Dispatch.now
            else before.dispatch
          val landing = if keeps then system.Phase.paused else system.Phase.scheduled
          assertEquals(
            after,
            List(
              Step(
                Outcome.accepted,
                before.copy(phase = landing, attempts = UpTo(0), dispatch = dispatch),
                List(if keeps then system.Fact.statusPaused else system.Fact.statusScheduled)
              )
            )
          )
        case system.Phase.started =>
          assertEquals(after.map(_.state), List(before.copy(phase = system.Phase.resetRequested)))
          assertEquals(after.map(_.facts), List(List(system.Fact.statusStarted)))
        case system.Phase.pauseRequested =>
          val (phase, fact) =
            // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
            if c == resets(1) then
              // scalafix:on DisableSyntax.==
              system.Phase.resetKeepingPause -> system.Fact.statusPaused
            else system.Phase.resetRequested -> system.Fact.statusStarted
          assertEquals(after.map(_.state), List(before.copy(phase = phase)))
          assertEquals(after.map(_.facts), List(List(fact)))
        case phase @ (system.Phase.unstarted | system.Phase.cancelRequested |
            system.Phase.resetRequested | system.Phase.resetKeepingPause | system.Phase.completed |
            system.Phase.failed | system.Phase.canceled | system.Phase.terminated |
            system.Phase.timedOut) =>
          val why =
            // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
            if phase == system.Phase.unstarted || phase.in[Closed] then
              // scalafix:on DisableSyntax.==
              Rejection.notFound
            else Rejection.failedPrecondition
          assertEquals(after.map(_.state), List(before), s"rejected reset $before")
          assertEquals(after.map(_.facts), List(Nil))
          assertEquals(after.map(_.outcome), List(Outcome.rejected(why)))
  }

  test("a pending reset is applied, outranked or kept by every settlement and control") {
    val failures = List(
      activityWorker.respondFailed(Failure.fatal),
      activityWorker.respondFailed(Failure.retryable),
      completionClient.respondFailedByID(Failure.fatal),
      completionClient.respondFailedByID(Failure.retryable)
    )
    for before <- states if pending.contains(before.phase) do
      for c <- failures do
        val after = take(before, c)
        assertEquals(after.map(_.state), List(restarted(before)), s"$c $before")
        assertEquals(
          after.map(_.facts),
          List(
            List(
              // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
              if before.phase == system.Phase.resetKeepingPause then
                // scalafix:on DisableSyntax.==
                system.Fact.statusPaused
              else system.Fact.statusScheduled,
              system.Fact.attemptCount
            )
          )
        )
      for (timer, armed) <- List(
          deadline.startToClose() -> (before.startToClose == Timeout.expires), // scalafix:ok DisableSyntax.==
          deadline
            .heartbeat() -> (before.heartbeat == Timeout.expires) // scalafix:ok DisableSyntax.==
        )
      do
        val after = take(before, timer)
        assertEquals(after.map(_.state), if armed then List(restarted(before)) else Nil)
        for step <- after; fact <- ActivitySystem.states.timeoutFacts do
          assert(!step.records(fact), s"a deferred reset records no terminal timeout: $step")
      for step <- take(before, deadline.scheduleToClose()) do
        assertEquals(step.state, before.copy(phase = system.Phase.timedOut))
        assertEquals(step.facts, List(system.Fact.statusTimedOut(TimeoutType.scheduleToClose)))
      assertEquals(
        take(before, activityWorker.respondCompleted()).map(_.state),
        List(before.copy(phase = system.Phase.completed))
      )
      assertEquals(
        take(before, client.requestCancel()).map(_.state),
        List(before.copy(phase = system.Phase.cancelRequested))
      )
      assertEquals(
        take(before, client.terminate()).map(_.state),
        List(before.copy(phase = system.Phase.terminated))
      )
      for c <- List[Class](client.pause(), client.unpause()) ++ resets do
        assertEquals(
          take(before, c),
          List(Step(Outcome.rejected(Rejection.failedPrecondition), before, Nil))
        )
      for c <- List[Class](activityWorker.respondCanceled(), completionClient.respondCanceledByID())
      do
        assertEquals(
          take(before, c),
          List(Step(Outcome.rejected(Rejection.invalidArgument), before, Nil))
        )
      assertEquals(take(before, activityWorker.heartbeat()).map(_.state), List(before))
  }

  test("every reset row and every settlement of a pending reset has a Product carrier") {
    val classes = resets ++ List(
      activityWorker.respondFailed(Failure.fatal),
      activityWorker.respondFailed(Failure.retryable),
      completionClient.respondFailedByID(Failure.fatal),
      completionClient.respondFailedByID(Failure.retryable),
      activityWorker.respondCompleted(),
      client.requestCancel(),
      deadline.heartbeat()
    )
    for before <- states; c <- classes; after <- take(before, c) do
      val visible = after.facts.filter(ActivitySystem.refinement.visible).map(_.productPrefix)
      val source = ActivitySystem.refinement.toProduct(before)
      val target = ActivitySystem.refinement.toProduct(after.state)
      val binding =
        product.ActivityProduct.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        source,
        c.values
      )
      assert(
        carriers.exists(p =>
          p.outcome == after.outcome && p.state == target && p.facts.map(
            _.productPrefix
          ) == visible
        ) || (source == target && visible.isEmpty), // scalafix:ok DisableSyntax.==
        s"reset refinement $c $before -> $after"
      )
    assertEquals(Refinement.unclosed(ActivitySystem.refinement)(_.phase, _.phase), IndexedSeq.empty)
  }

  test("the settlement Property holds on every row and rejects every seeded settlement error") {
    val settles = ActivitySystem.properties.resetSettles
    val precedence = ActivitySystem.properties.controlPrecedence
    // A held attempt was dispatched, so no reachable held state waits on a dispatch delay.
    val dispatched = states.filter(s =>
      !s.phase.in[Held] || s.dispatch == system.Dispatch.now
    ) // scalafix:ok DisableSyntax.==
    val all = dispatched.flatMap(before =>
      ActivitySystem.bindings
        .flatMap(b =>
          classesOf(b.decl).flatMap(values =>
            effectOf[system.State, Outcome, system.Fact](b.decl, b.function)(before, values)
          )
        )
        .map(before -> _)
    )
    for (before, after) <- all do
      assert(holds(settles, before, after), s"$before -> $after")
      assert(holds(precedence, before, after), s"$before -> $after")
    for before <- states if pending.contains(before.phase) do
      val applied = Step(
        Outcome.accepted,
        restarted(before),
        List(
          // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
          if before.phase == system.Phase.resetKeepingPause then
            // scalafix:on DisableSyntax.==
            system.Fact.statusPaused
          else system.Fact.statusScheduled
        )
      )
      assert(holds(settles, before, applied))
      val seeded = List(
        "fatality recorded" -> applied.copy(facts = applied.facts :+ system.Fact.statusFailed),
        "keepPaused landing" -> applied.copy(state =
          applied.state.copy(phase =
            // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
            if applied.state.phase == system.Phase.paused then
              // scalafix:on DisableSyntax.==
              system.Phase.scheduled
            else system.Phase.paused
          )
        ),
        "retry backoff kept" -> applied.copy(state =
          applied.state.copy(dispatch = system.Dispatch.backoff)
        ),
        "count kept" -> applied.copy(state = applied.state.copy(attempts = UpTo(1))),
        "terminal timeout" -> applied.copy(facts =
          applied.facts :+ system.Fact.statusTimedOut(TimeoutType.startToClose)
        ),
        "exhausted failure" -> Step(
          Outcome.accepted,
          before.copy(phase = system.Phase.failed),
          List(system.Fact.statusFailed)
        ),
        "wrong terminal timeout" -> Step(
          Outcome.accepted,
          before.copy(phase = system.Phase.timedOut),
          List(system.Fact.statusTimedOut(TimeoutType.heartbeat))
        ),
        "policy changed" -> applied.copy(state =
          applied.state.copy(maxAttempts =
            if before.maxAttempts == MaxAttempts.one then MaxAttempts.two
            else MaxAttempts.one // scalafix:ok DisableSyntax.==
          )
        )
      )
      for (name, step) <- seeded do assert(!holds(settles, before, step), s"$name: $before")
      for phase <- List(system.Phase.started, system.Phase.pauseRequested) do
        val undone = Step(Outcome.accepted, before.copy(phase = phase))
        assert(!holds(precedence, before, undone), s"pause or unpause undid a reset: $before")
    // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
    for before <- states if before.phase == system.Phase.cancelRequested; phase <- pending
    // scalafix:on DisableSyntax.==
    do assert(!holds(precedence, before, Step(Outcome.accepted, before.copy(phase = phase))))
    for
      before <- states
      if before.phase == system.Phase.started && before.attempts > 0 // scalafix:ok DisableSyntax.==
    do
      val rewound = Step(Outcome.accepted, before.copy(attempts = UpTo(0)))
      assert(!holds(settles, before, rewound), s"a held attempt rewound without reset: $before")
  }

  test("direct reset Properties reject a kept backoff, a kept count and a wrong pause landing") {
    for (property, c) <- List(
        ActivitySystem.properties.resetResumes -> resets(0),
        ActivitySystem.properties.resetKeepsPaused -> resets(1)
      )
    do
      assert(property.decl.when.contains(c))
      for before <- states; after <- take(before, c) do
        assert(holds(property, before, after), s"$c $before -> $after")
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        if before.phase == system.Phase.scheduled || before.phase == system.Phase.paused
        // scalafix:on DisableSyntax.==
        then
          if before.dispatch == system.Dispatch.backoff then // scalafix:ok DisableSyntax.==
            assert(
              !holds(
                property,
                before,
                after.copy(state = after.state.copy(dispatch = before.dispatch))
              )
            )
          if before.attempts > 0 then
            assert(
              !holds(
                property,
                before,
                after.copy(state = after.state.copy(attempts = before.attempts))
              )
            )
          for
            landing <- List(system.Phase.scheduled, system.Phase.paused)
            if landing != after.state.phase // scalafix:ok DisableSyntax.!=
          do assert(!holds(property, before, after.copy(state = after.state.copy(phase = landing))))
  }

  test("the ordinary fatal witness is unchanged and claims no reset coverage") {
    val query = ActivitySystem.retry.queries.nonRetryableFailure
    assertEquals(query.form, QueryForm.find)
    assertEquals(query.property, ActivitySystem.retry.properties.nonRetryableFails.decl)
    assertEquals(
      query.scenario.actions,
      Vector[ClassRef](
        client.start(),
        activityWorker.poll,
        activityWorker.respondFailed(Failure.fatal)
      )
    )
    assertEquals(query.expectedRun, Some(temporal.realize.satisfied))
    val held =
      take(take(ActivitySystem.init, client.start()).head.state, activityWorker.poll()).head.state
    val failed = take(held, activityWorker.respondFailed(Failure.fatal)).head
    assert(holds(ActivitySystem.retry.properties.nonRetryableFails, held, failed))
  }

  test("reset witnesses keep Model-only checks apart from the realized deferred Case") {
    for query <- List(
        ActivitySystem.reset.queries.resetFatality,
        ActivitySystem.reset.queries.resetExhaustion,
        ActivitySystem.reset.queries.resetTimeout,
        ActivitySystem.reset.queries.resetKeptPause,
        ActivitySystem.reset.queries.resetCompletion,
        ActivitySystem.reset.queries.resetScheduleToClose,
        ActivitySystem.reset.queries.resetCancellation,
        ActivitySystem.reset.queries.resetOutranksPause,
        ActivitySystem.reset.queries.resetRepeated,
        ActivitySystem.reset.queries.keepPausedReset
      )
    do
      assertEquals(query.form, QueryForm.find)
      assertEquals(query.expectedRun, None)
      val path = query.scenario.actions.foldLeft(List(ActivitySystem.init)) { (seen, c) =>
        val cls = c match
          case c: Class              => c
          case a: Action[EmptyTuple] => a()
          case other                 => fail(s"unexpected $other")
        seen :+ take(seen.last, cls).head.state
      }
      assert(path.size == query.scenario.actions.size + 1) // scalafix:ok DisableSyntax.==
    assertEquals(
      ActivitySystem.reset.queries.deferredResetCompletes.expectedRun,
      Some(temporal.realize.satisfied)
    )
  }
