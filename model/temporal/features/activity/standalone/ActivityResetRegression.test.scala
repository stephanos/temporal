package framework

import framework.Finite.given

import temporal.features.activity.{deadline, Failure, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  completionClient,
  pausing,
  product,
  protocol,
  MaxAttempts,
  ResetPause
}
import temporal.features.activity.standalone.protocol.ActivityProtocol
import framework.outcomes.{Outcome, Rejection}

class ActivityResetRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private type ActivityStep = Step[protocol.State, Outcome, protocol.Fact]

  private def take(s: protocol.State, c: Class): List[ActivityStep] =
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get
    effectOf[protocol.State, Outcome, protocol.Fact](binding.decl, binding.function)(s, c.values)
      .map(_.copy(because = ""))

  private type Transition = (protocol.State, ActivityStep) => Boolean

  private def holds(
      property: Property[protocol.State],
      before: protocol.State,
      after: ActivityStep
  ) =
    property.decl.holds2 match
      case Some(predicate) =>
        val claim = predicate.asInstanceOf[Transition] // scalafix:ok DisableSyntax.asInstanceOf
        claim(before, after)
      case None =>
        val claim = property.decl.holds.get
          .asInstanceOf[ActivityStep => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        claim(after)

  private val states = summon[Finite[protocol.State]].values
  private val resets = List(client.reset(ResetPause.resume), client.reset(ResetPause.keepPaused))
  private val pending = List(protocol.Phase.resetRequested, protocol.Phase.resetKeepingPause)
  private def restarted(before: protocol.State): protocol.State = before.copy(
    phase =
      if before.phase == protocol.Phase.resetKeepingPause then protocol.Phase.paused
      else protocol.Phase.scheduled,
    dispatch = protocol.Dispatch.now,
    attempts = UpTo(0)
  )

  test("reset is one per-RPC client action whose one input is keep_paused") {
    assertEquals(client.reset.decl.tokens, List(Some(pausing)))
    assertEquals(client.reset.decl.domains.map(_.values.toList), List(ResetPause.values.toList))
    assert(
      ActivityProtocol.bindings.exists(_.decl == client.reset.decl)
    )
    assert(
      product.ActivityProduct.bindings.exists(_.decl == client.reset.decl)
    )
    assertEquals(summon[Finite[protocol.State]].values.size, 5616)
  }

  test("a waiting or paused reset applies at once, clears backoff and lands by keep_paused") {
    for before <- states; c <- resets do
      val after = take(before, c)
      before.phase match
        case protocol.Phase.scheduled | protocol.Phase.paused =>
          val keeps =
            before.phase == protocol.Phase.paused && c == resets(1)
          val dispatch =
            if before.dispatch == protocol.Dispatch.backoff then protocol.Dispatch.now
            else before.dispatch
          val landing = if keeps then protocol.Phase.paused else protocol.Phase.scheduled
          assertEquals(
            after,
            List(
              Step(
                Outcome.accepted,
                before.copy(phase = landing, attempts = UpTo(0), dispatch = dispatch),
                List(if keeps then protocol.Fact.statusPaused else protocol.Fact.statusScheduled)
              )
            )
          )
        case protocol.Phase.started =>
          assertEquals(after.map(_.state), List(before.copy(phase = protocol.Phase.resetRequested)))
          assertEquals(after.map(_.facts), List(List(protocol.Fact.statusStarted)))
        case protocol.Phase.pauseRequested =>
          val (phase, fact) =
            if c == resets(1) then protocol.Phase.resetKeepingPause -> protocol.Fact.statusPaused
            else protocol.Phase.resetRequested -> protocol.Fact.statusStarted
          assertEquals(after.map(_.state), List(before.copy(phase = phase)))
          assertEquals(after.map(_.facts), List(List(fact)))
        case phase @ (protocol.Phase.unstarted | protocol.Phase.cancelRequested |
            protocol.Phase.resetRequested | protocol.Phase.resetKeepingPause |
            protocol.Phase.completed | protocol.Phase.failed | protocol.Phase.canceled |
            protocol.Phase.terminated | protocol.Phase.timedOut) =>
          val why =
            if phase == protocol.Phase.unstarted || phase.in[Closed] then Rejection.notFound
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
              if before.phase == protocol.Phase.resetKeepingPause then protocol.Fact.statusPaused
              else protocol.Fact.statusScheduled,
              protocol.Fact.attemptCount
            )
          )
        )
      for (timer, armed) <- List(
          deadline.startToClose() -> (before.startToClose == Timeout.expires),
          deadline
            .heartbeat() -> (before.heartbeat == Timeout.expires)
        )
      do
        val after = take(before, timer)
        assertEquals(after.map(_.state), if armed then List(restarted(before)) else Nil)
        for step <- after; fact <- ActivityProtocol.states.timeoutFacts do
          assert(!step.records(fact), s"a deferred reset records no terminal timeout: $step")
      for step <- take(before, deadline.scheduleToClose()) do
        assertEquals(step.state, before.copy(phase = protocol.Phase.timedOut))
        assertEquals(step.facts, List(protocol.Fact.statusTimedOut(TimeoutType.scheduleToClose)))
      assertEquals(
        take(before, activityWorker.respondCompleted()).map(_.state),
        List(before.copy(phase = protocol.Phase.completed))
      )
      assertEquals(
        take(before, client.requestCancel()).map(_.state),
        List(before.copy(phase = protocol.Phase.cancelRequested))
      )
      assertEquals(
        take(before, client.terminate()).map(_.state),
        List(before.copy(phase = protocol.Phase.terminated))
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
      val visible = after.facts.filter(ActivityProtocol.refinement.visible).map(_.productPrefix)
      val source = ActivityProtocol.refinement.toProduct(before)
      val target = ActivityProtocol.refinement.toProduct(after.state)
      val binding =
        product.ActivityProduct.bindings.find(_.decl == c.decl).get
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        source,
        c.values
      )
      assert(
        carriers.exists(p =>
          p.outcome == after.outcome && p.state == target && p.facts.map(
            _.productPrefix
          ) == visible
        ) || (source == target && visible.isEmpty),
        s"reset refinement $c $before -> $after"
      )
    assertEquals(
      Refinement.unclosed(ActivityProtocol.refinement)(_.phase, _.phase),
      IndexedSeq.empty
    )
  }

  test("the settlement Property holds on every row and rejects every seeded settlement error") {
    val settles = ActivityProtocol.properties.resetSettles
    val precedence = ActivityProtocol.properties.controlPrecedence
    // A held attempt was dispatched, so no reachable held state waits on a dispatch delay.
    val dispatched = states.filter(s => !s.phase.in[Held] || s.dispatch == protocol.Dispatch.now)
    val all = dispatched.flatMap(before =>
      ActivityProtocol.bindings
        .flatMap(b =>
          classesOf(b.decl).flatMap(values =>
            effectOf[protocol.State, Outcome, protocol.Fact](b.decl, b.function)(before, values)
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
          if before.phase == protocol.Phase.resetKeepingPause then protocol.Fact.statusPaused
          else protocol.Fact.statusScheduled
        )
      )
      assert(holds(settles, before, applied))
      val seeded = List(
        "fatality recorded" -> applied.copy(facts = applied.facts :+ protocol.Fact.statusFailed),
        "keepPaused landing" -> applied.copy(state =
          applied.state.copy(phase =
            if applied.state.phase == protocol.Phase.paused then protocol.Phase.scheduled
            else protocol.Phase.paused
          )
        ),
        "retry backoff kept" -> applied.copy(state =
          applied.state.copy(dispatch = protocol.Dispatch.backoff)
        ),
        "count kept" -> applied.copy(state = applied.state.copy(attempts = UpTo(1))),
        "terminal timeout" -> applied.copy(facts =
          applied.facts :+ protocol.Fact.statusTimedOut(TimeoutType.startToClose)
        ),
        "exhausted failure" -> Step(
          Outcome.accepted,
          before.copy(phase = protocol.Phase.failed),
          List(protocol.Fact.statusFailed)
        ),
        "wrong terminal timeout" -> Step(
          Outcome.accepted,
          before.copy(phase = protocol.Phase.timedOut),
          List(protocol.Fact.statusTimedOut(TimeoutType.heartbeat))
        ),
        "policy changed" -> applied.copy(state =
          applied.state.copy(maxAttempts =
            if before.maxAttempts == MaxAttempts.one then MaxAttempts.two
            else MaxAttempts.one
          )
        )
      )
      for (name, step) <- seeded do assert(!holds(settles, before, step), s"$name: $before")
      for phase <- List(protocol.Phase.started, protocol.Phase.pauseRequested) do
        val undone = Step(Outcome.accepted, before.copy(phase = phase))
        assert(!holds(precedence, before, undone), s"pause or unpause undid a reset: $before")
    for before <- states if before.phase == protocol.Phase.cancelRequested; phase <- pending
    do assert(!holds(precedence, before, Step(Outcome.accepted, before.copy(phase = phase))))
    for
      before <- states
      if before.phase == protocol.Phase.started && before.attempts > 0
    do
      val rewound = Step(Outcome.accepted, before.copy(attempts = UpTo(0)))
      assert(!holds(settles, before, rewound), s"a held attempt rewound without reset: $before")
  }

  test("direct reset Properties reject a kept backoff, a kept count and a wrong pause landing") {
    for (property, c) <- List(
        ActivityProtocol.properties.resetResumes -> resets(0),
        ActivityProtocol.properties.resetKeepsPaused -> resets(1)
      )
    do
      assert(property.decl.when.contains(c))
      for before <- states; after <- take(before, c) do
        assert(holds(property, before, after), s"$c $before -> $after")
        if before.phase == protocol.Phase.scheduled || before.phase == protocol.Phase.paused
        then
          if before.dispatch == protocol.Dispatch.backoff then
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
            landing <- List(protocol.Phase.scheduled, protocol.Phase.paused)
            if landing != after.state.phase
          do assert(!holds(property, before, after.copy(state = after.state.copy(phase = landing))))
  }

  test("the ordinary fatal witness is unchanged and claims no reset coverage") {
    val query = ActivityProtocol.retry.queries.nonRetryableFailure
    assertEquals(query.form, QueryForm.find)
    assertEquals(query.property.when, Some(activityWorker.respondFailed(Failure.fatal)))
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
      take(take(ActivityProtocol.init, client.start()).head.state, activityWorker.poll()).head.state
    val failed = take(held, activityWorker.respondFailed(Failure.fatal)).head
    assert(holds(Property[protocol.State](query.property), held, failed))
  }

  test("reset witnesses keep Model-only checks apart from the realized deferred Case") {
    for query <- List(
        ActivityProtocol.reset.queries.resetFatality,
        ActivityProtocol.reset.queries.resetExhaustion,
        ActivityProtocol.reset.queries.resetTimeout,
        ActivityProtocol.reset.queries.resetKeptPause,
        ActivityProtocol.reset.queries.resetCompletion,
        ActivityProtocol.reset.queries.resetScheduleToClose,
        ActivityProtocol.reset.queries.resetCancellation,
        ActivityProtocol.reset.queries.resetOutranksPause,
        ActivityProtocol.reset.queries.resetRepeated,
        ActivityProtocol.reset.queries.keepPausedReset
      )
    do
      assertEquals(query.form, QueryForm.find)
      assertEquals(query.expectedRun, None)
      val path = query.scenario.actions.foldLeft(List(ActivityProtocol.init)) { (seen, c) =>
        val cls = c match
          case c: Class              => c
          case a: Action[EmptyTuple] => a()
          case other                 => fail(s"unexpected $other")
        seen :+ take(seen.last, cls).head.state
      }
      assert(path.size == query.scenario.actions.size + 1)
    assertEquals(
      ActivityProtocol.reset.queries.deferredResetCompletes.expectedRun,
      Some(temporal.realize.satisfied)
    )
  }
