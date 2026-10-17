package framework.dsl

import samples.*

class NativeMapsTest extends munit.FunSuite:
  private def run[S, O](
      machine: RecordMachine[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  // A Bank state by its one var, and the balances a state holds.
  private def bank(balances: Bank.Balances): State = Bank.Model.init.copy(balances = balances)
  private def held(state: State): Bank.Balances = Bank.Model.at(state)(Bank.Model.balances)

  // A TwoPhaseCommit state by its vars, as the Transaction record holds them, and the record a
  // state holds.
  private def transaction(record: TwoPhaseCommit.Transaction): State =
    TwoPhaseCommit.Model.init.copy(
      rmStates = record.rmStates,
      tmState = record.tmState,
      tmPrepared = record.tmPrepared,
      messages = record.messages
    )
  private def recorded(state: State): TwoPhaseCommit.Transaction =
    TwoPhaseCommit.Model.at(state)(TwoPhaseCommit.Model.current)

  test("Bank native balances do not invent a whole-state integer bound"):
    import Bank.*
    val large = withBalance(held(Model.init), Addr.king, Denom.banana, 7)
    assertEquals(balanceOf(large, Addr.king, Denom.banana), 7)
    assertEquals(balanceOf(large, Addr.king, Denom.burger), 2)

  test("Bank absent keys read zero and zero writes remove denominations then accounts"):
    import Bank.*
    val empty = Balances(Map.empty)
    assertEquals(balanceOf(empty, Addr.mario, Denom.banana), 0)
    val burgers = Balances(Map(Addr.king -> Map(Denom.burger -> 2)))
    assertEquals(balanceOf(burgers, Addr.king, Denom.banana), 0)
    assertEquals(withBalance(held(Model.init), Addr.king, Denom.banana, 0), burgers)
    assertEquals(withBalance(burgers, Addr.king, Denom.burger, 0), empty)
    assertEquals(withBalance(empty, Addr.mario, Denom.banana, 0), empty)

  test("Bank ordinary transfer debits and credits with sparse cleanup"):
    import Bank.*
    val first = run(Model, Model.init, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1)))
    val paid = bank(
      Balances(
        Map(
          Addr.king -> Map(Denom.banana -> 1, Denom.burger -> 2),
          Addr.peaches -> Map(Denom.banana -> 1)
        )
      )
    )
    assertEquals(first, List(Step(Outcome.accepted, paid)))
    val second = run(Model, paid, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1)))
    assertEquals(
      second,
      List(
        Step(
          Outcome.accepted,
          bank(
            Balances(
              Map(Addr.king -> Map(Denom.burger -> 2), Addr.peaches -> Map(Denom.banana -> 2))
            )
          )
        )
      )
    )
    val last = bank(Balances(Map(Addr.mario -> Map(Denom.banana -> 1))))
    assertEquals(
      run(Model, last, client.send(Send(Addr.mario, Addr.peaches, Denom.banana, 1))),
      List(Step(Outcome.accepted, bank(Balances(Map(Addr.peaches -> Map(Denom.banana -> 1))))))
    )

  test("Bank zero sends preserve funded acceptance and empty refusal including self"):
    import Bank.*
    for to <- List(Addr.king, Addr.peaches) do
      assertEquals(
        run(Model, Model.init, client.send(Send(Addr.king, to, Denom.banana, 0))),
        List(Step(Outcome.accepted, Model.init))
      )
    for to <- List(Addr.mario, Addr.king) do
      assertEquals(
        run(Model, Model.init, client.send(Send(Addr.mario, to, Denom.banana, 0))),
        List(Step(Outcome.invalidCoinsOrInsufficientFunds, Model.init))
      )

  test("Bank insufficient funds precede self transfer and receiver overflow"):
    import Bank.*
    assertEquals(
      run(Model, Model.init, client.send(Send(Addr.king, Addr.king, Denom.banana, 2))),
      List(Step(Outcome.accepted, Model.init))
    )
    for to <- List(Addr.king, Addr.peaches) do
      assertEquals(
        run(Model, Model.init, client.send(Send(Addr.king, to, Denom.banana, 3))),
        List(Step(Outcome.invalidCoinsOrInsufficientFunds, Model.init))
      )
    assertEquals(
      run(Leaky, Model.init, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 3))),
      List(Step(Outcome.overflow, Model.init))
    )
    assertEquals(
      run(Leaky, Model.init, client.send(Send(Addr.mario, Addr.mario, Denom.banana, 3))),
      List(Step(Outcome.accepted, Model.init))
    )
    val fullReceiver = bank(
      Balances(Map(Addr.king -> Map(Denom.banana -> 2), Addr.peaches -> Map(Denom.banana -> 2)))
    )
    assertEquals(
      run(Model, fullReceiver, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))),
      List(Step(Outcome.overflow, fullReceiver))
    )

  test("Bank Properties distinguish an ordinary transfer from credit without debit"):
    import Bank.*
    val cls = client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))
    val paid = run(Model, Model.init, cls).head
    val inflated = run(Leaky, Model.init, cls).head
    assertEquals(
      inflated,
      Step(
        Outcome.accepted,
        bank(
          Balances(
            Map(
              Addr.king -> Map(Denom.banana -> 2, Denom.burger -> 2),
              Addr.peaches -> Map(Denom.banana -> 1)
            )
          )
        )
      )
    )
    val supply =
      Model.invariants.totalSupplyInv.decl.holds.get
        .asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
    val leaky =
      Leaky.invariants.leakyTotalSupplyInv.decl.holds.get
        .asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
    assertEquals(supply(paid), true)
    assertEquals(supply(inflated), false)
    assertEquals(leaky(paid), true)
    assertEquals(leaky(inflated), false)
    assert(Leaky.invariants.leakyTotalSupplyInv.decl.machine eq Leaky)
    val refuses = Model.postconditions.failedSendChangesNothing.decl.holds2.get
      .asInstanceOf[(State, Step[State, Outcome, Nothing]) => Boolean]
    assertEquals(refuses(Model.init, Step(Outcome.overflow, Model.init)), true)
    assertEquals(refuses(Model.init, Step(Outcome.overflow, paid.state)), false)

  test("Bank record inputs preserve ascending amount order and refuse outside-domain classes"):
    import Bank.*
    val classes = classesOf(client.send.decl)
    assertEquals(classes.size, 128)
    assertEquals(
      classes.take(5),
      List(
        List(Addr.king, Addr.king, Denom.banana, 0),
        List(Addr.king, Addr.king, Denom.banana, 1),
        List(Addr.king, Addr.king, Denom.banana, 2),
        List(Addr.king, Addr.king, Denom.banana, 3),
        List(Addr.king, Addr.king, Denom.burger, 0)
      )
    )
    assertEquals(classes.last, List(Addr.mario, Addr.mario, Denom.burger, 3))
    intercept[IllegalArgumentException](
      client.send(Send(Addr.king, Addr.peaches, Denom.banana, -1))
    ): Unit
    intercept[IllegalArgumentException](
      client.send(Send(Addr.king, Addr.peaches, Denom.banana, 4))
    ): Unit

  test("TwoPC ordinary guards separate sent preparation from received preparation"):
    import TwoPhaseCommit.*
    val initial = Model.init
    assertEquals(run(Model, initial, tm.addPrepared(ResourceManager.rm1)), Nil)
    assertEquals(run(Model, initial, tm.decideOnCommit()), Nil)
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(run(Model, prepared, tm.decideOnCommit()), Nil)
    val received = run(Model, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(
      run(Model, received, tm.addPrepared(ResourceManager.rm1)),
      List(Step(Outcome.accepted, received))
    )
    assertEquals(run(Model, received, tm.decideOnCommit()), Nil)
    assertEquals(
      recorded(run(EagerCommit, received, tm.decideOnCommit()).head.state).tmState,
      TransactionManagerState.committed
    )

  test("TwoPC prepare and receive retain exact maps and persistent sent messages"):
    import TwoPhaseCommit.*
    val initial = Model.init
    assertEquals(
      initial,
      transaction(
        Transaction(
          Map(
            ResourceManager.rm1 -> ResourceManagerState.working,
            ResourceManager.rm2 -> ResourceManagerState.working,
            ResourceManager.rm3 -> ResourceManagerState.working
          ),
          TransactionManagerState.init,
          Set.empty,
          Set.empty
        )
      )
    )
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(
      prepared,
      transaction(
        Transaction(
          Map(
            ResourceManager.rm1 -> ResourceManagerState.prepared,
            ResourceManager.rm2 -> ResourceManagerState.working,
            ResourceManager.rm3 -> ResourceManagerState.working
          ),
          TransactionManagerState.init,
          Set.empty,
          Set(Message.prepared(ResourceManager.rm1))
        )
      )
    )
    assertEquals(run(Model, prepared, rm.prepare(ResourceManager.rm1)), Nil)
    val received = run(Model, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(recorded(received).tmPrepared, Set(ResourceManager.rm1))
    assertEquals(recorded(received).messages, Set(Message.prepared(ResourceManager.rm1)))
    assertEquals(
      run(Model, received, tm.addPrepared(ResourceManager.rm1)),
      List(Step(Outcome.accepted, received))
    )
    intercept[NoSuchElementException](
      mayPrepare(recorded(initial.copy(rmStates = Map.empty)), ResourceManager.rm1)
    ): Unit

  test("TwoPC all received managers enable commit and every manager follows"):
    import TwoPhaseCommit.*
    val prepared = List(
      rm.prepare(ResourceManager.rm1),
      rm.prepare(ResourceManager.rm2),
      rm.prepare(ResourceManager.rm3)
    ).foldLeft(Model.init)((s, cls) => run(Model, s, cls).head.state)
    val received = List(
      tm.addPrepared(ResourceManager.rm1),
      tm.addPrepared(ResourceManager.rm2),
      tm.addPrepared(ResourceManager.rm3)
    ).foldLeft(prepared)((s, cls) => run(Model, s, cls).head.state)
    assertEquals(
      recorded(received).tmPrepared,
      Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)
    )
    val sent = Set(
      Message.prepared(ResourceManager.rm1),
      Message.prepared(ResourceManager.rm2),
      Message.prepared(ResourceManager.rm3)
    )
    assertEquals(recorded(received).messages, sent)
    val decided = run(Model, received, tm.decideOnCommit()).head.state
    assertEquals(recorded(decided).tmState, TransactionManagerState.committed)
    assertEquals(recorded(decided).messages, sent.incl(Message.commit))
    assertEquals(Model.end(decided), false)
    assertEquals(run(Model, decided, tm.decideOnAbort()), Nil)
    assertEquals(run(Model, decided, tm.addPrepared(ResourceManager.rm1)), Nil)
    val committed = List(
      rm.commit(ResourceManager.rm1),
      rm.commit(ResourceManager.rm2),
      rm.commit(ResourceManager.rm3)
    ).foldLeft(decided)((s, cls) => run(Model, s, cls).head.state)
    assertEquals(
      recorded(committed).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.committed,
        ResourceManager.rm3 -> ResourceManagerState.committed
      )
    )
    assertEquals(
      recorded(committed).tmPrepared,
      Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)
    )
    assertEquals(recorded(committed).messages, sent.incl(Message.commit))
    assertEquals(Model.end(committed), true)
    assertEquals(run(Model, committed, rm.commit(ResourceManager.rm1)), Nil)
    val everyone =
      Model.postconditions.everyoneCommits.decl.holds.get
        .asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
    assertEquals(everyone(Step(Outcome.accepted, committed)), true)
    assertEquals(everyone(Step(Outcome.accepted, decided)), false)

  test("TwoPC spontaneous abort and abort messages preserve the original guards"):
    import TwoPhaseCommit.*
    val initial = Model.init
    val aborted = run(Model, initial, rm.abort(ResourceManager.rm2)).head.state
    assertEquals(
      recorded(aborted).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.working,
        ResourceManager.rm2 -> ResourceManagerState.aborted,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(recorded(aborted).messages, Set.empty[Message])
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(run(Model, prepared, rm.abort(ResourceManager.rm1)), Nil)
    assertEquals(run(Model, prepared, rm.commit(ResourceManager.rm1)), Nil)
    val decision = run(Model, prepared, tm.decideOnAbort()).head.state
    assertEquals(
      recorded(decision).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.abort)
    )
    val after = run(Model, decision, rm.abort(ResourceManager.rm1)).head.state
    assertEquals(
      recorded(after).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.aborted,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(after).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.abort)
    )
    assertEquals(
      run(Model, after, rm.abort(ResourceManager.rm1)),
      List(Step(Outcome.accepted, after))
    )
    val commitOnly = initial.copy(messages = Set(Message.commit))
    assertEquals(
      recorded(run(Model, commitOnly, rm.commit(ResourceManager.rm1)).head.state)
        .rmStates(ResourceManager.rm1),
      ResourceManagerState.committed
    )
    val both = after.copy(rmStates =
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(run(Model, both, rm.abort(ResourceManager.rm1)).head.state)
        .rmStates(ResourceManager.rm1),
      ResourceManagerState.aborted
    )

  test("TwoPC EagerCommit admits the independent committed-aborted negative control"):
    import TwoPhaseCommit.*
    val prepared = run(EagerCommit, Model.init, rm.prepare(ResourceManager.rm1)).head.state
    val received = run(EagerCommit, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(run(Model, received, tm.decideOnCommit()), Nil)
    val decided = run(EagerCommit, received, tm.decideOnCommit()).head.state
    val committed = run(EagerCommit, decided, rm.commit(ResourceManager.rm1)).head.state
    val conflict = run(EagerCommit, committed, rm.abort(ResourceManager.rm2)).head
    assertEquals(
      recorded(conflict.state).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.aborted,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(conflict.state).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.commit)
    )
    assertEquals(recorded(conflict.state).tmPrepared, Set(ResourceManager.rm1))
    val normal =
      Model.invariants.consistencyInv.decl.holds.get
        .asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
    val eager = EagerCommit.invariants.eagerConsistencyInv.decl.holds.get
      .asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
    assertEquals(normal(Step(Outcome.accepted, Model.init)), true)
    assertEquals(eager(Step(Outcome.accepted, Model.init)), true)
    assertEquals(normal(conflict), false)
    assertEquals(eager(conflict), false)
    assert(EagerCommit.invariants.eagerConsistencyInv.decl.machine eq EagerCommit)

  test("NativeInputs accepts an amount within its ceiling once, by an input guard on its vars"):
    import NativeInputs.*
    def accepted(seen: Set[Int]): State = Accepted.init.copy(seen = seen)
    assertEquals(
      run(Accepted, Accepted.init, caller.submit(Request(0, 1))),
      List(Step(true, accepted(Set(0))))
    )
    assertEquals(run(Accepted, Accepted.init, caller.submit(Request(2, 1))), Nil)
    assertEquals(run(Accepted, accepted(Set(0)), caller.submit(Request(0, 2))), Nil)
    assertEquals(
      run(Accepted, accepted(Set(0)), caller.submit(Request(2, 2))),
      List(Step(true, accepted(Set(0, 2))))
    )
    assert(!Accepted.end(Accepted.init))
    assert(Accepted.end(accepted(Set(2))))
    val zero = Accepted.postconditions.zeroWasAccepted.decl.holds.get
      .asInstanceOf[Step[State, Boolean, Nothing] => Boolean]
    assert(zero(Step(true, accepted(Set(0)))))
    assert(zero(Step(true, accepted(Set(0, 2)))))
    assert(!zero(Step(true, accepted(Set(2)))))
