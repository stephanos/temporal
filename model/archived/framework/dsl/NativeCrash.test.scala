package framework.dsl

import framework.checker.Effects.*

class NativeCrashSuite extends munit.FunSuite:
  enum Custody derives CanEqual:
    case idle, buffered, reserved, stored

  given Ok[Boolean] = Ok(true)

  object faults:
    private val actor = Actor("fault")
    val crash = framework.dsl.action("crash", actor).crash
    val second = framework.dsl.action("second", actor).responseLoss
    val authored = framework.dsl.action("authored", actor).responseLoss
    val extra = framework.dsl.action("extra", actor).responseLoss

  final private class Calls:
    var resets = 0
    var callbacks = 0
    def poison(): Unit =
      callbacks += 1
      throw AssertionError("effect evaluated")

  class Subject(mode: Int) extends Machine[Boolean, Boolean]:
    var kept = 13
    var pending = 8
    var custody = Custody.buffered
    var available = true
    var remaining = 2
    var bounded = UpTo[2](2)
    var other = 7
    durability(
      durable(kept),
      ephemeral(pending, 0),
      inMemory(custody)(Custody.buffered -> Custody.idle, Custody.reserved -> Custody.stored),
      durable(available),
      durable(remaining),
      durable(bounded),
      durable(other)
    )

    object transitions extends Transitions:
      mode match
        case 0 => crashes(faults.crash, false, true, false)
        case 1 => crashes(faults.crash.budgetedBy(available), false, true, false)
        case 2 => crashes(faults.crash.budgetedBy(remaining), false, true, false)
        case _ => crashes(faults.crash.budgetedBy(bounded), false, true, false)
      on(faults.second.budgetedBy(other)):
        effects.noop()

    object effects:
      def noop() =
        ()
      def authoredCrash() =
        remaining -= 1

  private def run[O, F](machine: NativeModel[State, O, F], state: State, cls: Class) =
    machine.nativeEffects.find(_._1 eq cls.decl).get._2(state, cls)

  test("a generated crash reconstructs sibling resets as one valid record"):
    final case class Paired(left: Int, right: Int):
      require(left == right)
    object PairedSubject extends Machine[Boolean, Boolean]:
      var pair = Paired(1, 1)
      durability(ephemeral(pair.left, 0), ephemeral(pair.right, 0))
      object transitions extends Transitions:
        crashes(faults.crash, true)
    assertEquals(
      run(PairedSubject, PairedSubject.init, faults.crash()),
      List(Step(true, PairedSubject.init.copy(pair = Paired(0, 0))))
    )

  test("a budgeted crash reconstructs nested resets and its own budget atomically"):
    final case class Paired(left: Int, right: Int):
      require(left == right)
    final case class Coupled(pair: Paired, remaining: Int):
      require(pair.left == remaining)
    object CoupledSubject extends Machine[Boolean, Boolean]:
      var coupled = Coupled(Paired(1, 1), 1)
      durability(
        ephemeral(coupled.pair.left, 0),
        ephemeral(coupled.pair.right, 0),
        durable(coupled.remaining)
      )
      object transitions extends Transitions:
        crashes(faults.crash.budgetedBy(coupled.remaining), true)
    val exhausted = CoupledSubject.init.copy(coupled = Coupled(Paired(0, 0), 0))
    assertEquals(
      run(CoupledSubject, CoupledSubject.init, faults.crash()),
      List(Step(true, exhausted))
    )
    assertEquals(run(CoupledSubject, exhausted, faults.crash()), Nil)

  test("derived crash matches ordinary independent field reset including unmapped values"):
    val machine = new Subject(0)
    for (from, to) <- List(
        Custody.idle -> Custody.idle,
        Custody.buffered -> Custody.idle,
        Custody.reserved -> Custody.stored,
        Custody.stored -> Custody.stored
      )
    do
      val state = machine.init.copy(custody = from)
      val expected = state.copy(pending = 0, custody = to)
      assertEquals(
        run(machine, state, faults.crash()),
        List(Step(false, expected, List(true, false)))
      )

  test("generated own budgets consume once preserving every other budget and durable value"):
    for mode <- List(1, 2, 3) do
      val machine = new Subject(mode)
      val state = machine.init.copy(custody = Custody.reserved)
      val reset = state.copy(pending = 0, custody = Custody.stored)
      val expected = mode match
        case 1 => reset.copy(available = false)
        case 2 => reset.copy(remaining = 1)
        case _ => reset.copy(bounded = UpTo[2](1))
      assertEquals(
        run(machine, state, faults.crash()),
        List(Step(false, expected, List(true, false)))
      )
      assertEquals(
        run(machine, state.copy(other = -7), faults.crash()),
        List(Step(false, expected.copy(other = -7), List(true, false)))
      )
      val spent = mode match
        case 1 => state.copy(available = false)
        case 2 => state.copy(remaining = 0)
        case _ => state.copy(bounded = UpTo[2](0))
      assertEquals(run(machine, spent, faults.crash()), Nil)

  test(
    "negative own Int refuses with binding location while authored overrides consume themselves"
  ):
    val machine = new Subject(2)
    val negative = machine.init.copy(remaining = -1)
    val error = intercept[IllegalArgumentException](run(machine, negative, faults.crash()))
    assert(error.getMessage.contains("crash"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    val overridden = machine.rebind(on(faults.crash):
      machine.effects.authoredCrash())
    assertEquals(
      run(overridden, machine.init, faults.crash()),
      List(Step(true, machine.init.copy(remaining = 1)))
    )
    assertEquals(
      run(overridden, negative, faults.crash()),
      List(Step(true, negative.copy(remaining = -2)))
    )
    val negativeOther = machine.init.copy(other = -1)
    assertEquals(run(machine, negativeOther, faults.second()), List(Step(true, negativeOther)))

  test("never-invoked crash and orphan classifications refuse at completed acquisition"):
    val calls = new Calls
    class Missing extends Machine[Boolean, Nothing]:
      var kept = 13
      override def end: Boolean =
        calls.poison()
        false
      object transitions extends Transitions:
        crashes(faults.crash, true)
    val error = intercept[IllegalArgumentException](new Missing().bindings)
    assert(error.getMessage.contains("classification"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    class Orphan extends Machine[Boolean, Nothing]:
      var kept = 13
      durability(durable(kept))
      object transitions extends Transitions
    intercept[IllegalArgumentException](new Orphan().bindings): Unit
    assertEquals(calls.callbacks, 0)

  test("crash-free native derivations reject newly introduced orphan classifications"):
    object Plain extends Machine[Boolean, Nothing]:
      var pending = 8
      object transitions extends Transitions
    object NewlyClassified extends Derived(Plain):
      durability(FieldDurability.durable(this, StateField.captureVar(Plain.pending, this)))
    val error = intercept[IllegalArgumentException](NewlyClassified.bindings)
    assert(error.getMessage.contains("requires a crash binding"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)

  test("exhausted and negative budgets stop before a poison reset while positives reach it"):
    val calls = new Calls
    class Poison(mode: Int) extends Machine[Boolean, Nothing]:
      var available = true
      var remaining = 1
      var bounded = UpTo[2](1)
      var pending = true
      private lazy val selected = StateField.captureVar(pending, this)
      private lazy val poisoned = StateField.make[State, Boolean](
        selected.ownerType,
        selected.fieldType,
        selected.path,
        selected.location,
        selected.rootFields,
        selected.valid,
        _ =>
          calls.resets += 1
          throw AssertionError("reset evaluated")
        ,
        selected.write,
        selected.rebuild
      )
      durability(
        durable(available),
        durable(remaining),
        durable(bounded),
        FieldDurability.inMemory(this, poisoned, Vector(true -> false))
      )
      object transitions extends Transitions:
        mode match
          case 1 => crashes(faults.crash.budgetedBy(available), true)
          case 2 => crashes(faults.crash.budgetedBy(remaining), true)
          case _ => crashes(faults.crash.budgetedBy(bounded), true)
    for mode <- List(1, 2, 3) do
      val subject = new Poison(mode)
      val exhausted = subject.init.copy(available = false, remaining = 0, bounded = UpTo[2](0))
      val before = calls.resets
      assertEquals(run(subject, exhausted, faults.crash()), Nil)
      assertEquals(calls.resets, before)
      if mode == 2 then
        val error = intercept[IllegalArgumentException](
          run(subject, exhausted.copy(remaining = -1), faults.crash())
        )
        assert(error.getMessage.contains("negative own budget"), error.getMessage)
        assertEquals(calls.resets, before)
      assertEquals(
        intercept[AssertionError](run(subject, subject.init, faults.crash())).getMessage,
        "reset evaluated"
      )
      assertEquals(calls.resets, before + 1)

  test("completed acquisition refuses duplicate generated crash bindings"):
    class Duplicate extends Machine[Boolean, Boolean]:
      var kept = 13
      durability(durable(kept))
      object transitions extends Transitions:
        crashes(faults.crash, true)
        crashes(faults.crash, false)
    val error = intercept[IllegalArgumentException](new Duplicate().bindings)
    assert(error.getMessage.contains("crash"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)

  test("completed metadata rejects malformed dormant classifications before any callback"):
    val calls = new Calls
    class Invalid(mode: Int) extends Machine[Boolean, Nothing]:
      var value = UpTo[2](2)
      var pending = true
      mode match
        case 0 => durability(durable(value))
        case 1 => durability(durable(value), durable(value), durable(pending))
        case 2 =>
          val forged = 3.asInstanceOf[UpTo[2]] // scalafix:ok DisableSyntax.asInstanceOf
          durability(ephemeral(value, forged), durable(pending))
        case 3 =>
          durability(
            inMemory(value)(UpTo[2](2) -> UpTo[2](1), UpTo[2](1) -> UpTo[2](0)),
            durable(pending)
          )
        case _ => durability(ephemeral(value, UpTo[2](0)), durable(pending))
      object transitions extends Transitions:
        on(faults.crash.budgetedBy(value)):
          effects.poison()
      object effects:
        def poison() =
          calls.poison()
    for mode <- 0 to 4 do
      val error = intercept[IllegalArgumentException](new Invalid(mode).nativeBudgets)
      assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    assertEquals(calls.callbacks, 0)

  test("nested field resets and own-budget writes compose in the same native transition"):
    final case class Inner(remaining: Int, pending: Boolean)
    object Nested extends Machine[Boolean, Nothing]:
      var inner = Inner(2, true)
      var kept = 15
      durability(durable(inner.remaining), ephemeral(inner.pending, false), durable(kept))
      object transitions extends Transitions:
        crashes(faults.crash.budgetedBy(inner.remaining), true)
    assertEquals(
      run(Nested, Nested.init, faults.crash()),
      List(Step(true, Nested.init.copy(inner = Inner(1, false))))
    )

  test("empty custom realization acquisition rejects dormant invalid crashes before dependencies"):
    import framework.realize.{ConstructedRealization, Performance}
    val calls = new Calls
    var exposedDependencies = 0
    class Admission(valid: Boolean) extends Machine[Boolean, Boolean]:
      var kept = 13
      override def end: Boolean =
        calls.poison()
        false
      if valid then durability(durable(kept))
      object transitions extends Transitions:
        crashes(faults.crash, true)
    class Kit(val machine: MachineDeclaration[?, ?, ?]) extends ConstructedRealization:
      val runs: Seq[AnyRef] = Nil
      protected def declaredPerformances: Vector[Performance] = Vector.empty
      protected def declaredDependencies: ConstructedRealization.Dependencies =
        exposedDependencies += 1
        ConstructedRealization.Dependencies.empty
      def toIR(
          context: ConstructedRealization.Context
      ): io.temporal.server.api.umpire.v1.Realization =
        throw AssertionError("conversion evaluated")
    val admitted = new Kit(new Admission(true))
    assertEquals(admitted.nativePerformances, Vector.empty)
    assertEquals(admitted.dependencies, ConstructedRealization.Dependencies.empty)
    assertEquals(exposedDependencies, 1)
    val invalid = new Kit(new Admission(false))
    intercept[IllegalArgumentException](invalid.nativePerformances): Unit
    intercept[IllegalArgumentException](invalid.dependencies): Unit
    assertEquals(exposedDependencies, 1)
    assertEquals(calls.callbacks, 0)

  test("generated own budgets refuse active aliases while authored overrides retain shared paths"):
    val calls = new Calls
    class Shared(shared: Boolean) extends Machine[Boolean, Boolean]:
      var remaining = 2
      var other = 7
      var pending = 8
      durability(durable(remaining), durable(other), ephemeral(pending, 0))
      object transitions extends Transitions:
        crashes(faults.crash.budgetedBy(remaining), false)
        if shared then
          on(faults.second.budgetedBy(remaining)):
            effects.poison()
        else
          on(faults.second.budgetedBy(other)):
            effects.poison()
      object effects:
        def poison() =
          calls.poison()
    val distinct = new Shared(false)
    assertEquals(distinct.nativeBudgets.size, 2)
    val authored = distinct
      .rebind(on(faults.crash):
        distinct.effects.poison())
      .extend(on(faults.extra.budgetedBy(distinct.remaining)):
        distinct.effects.poison())
    assertEquals(authored.nativeBudgets.size, 3)
    val error = intercept[IllegalArgumentException](new Shared(true).nativeBudgets)
    assert(error.getMessage.contains("aliases another fault budget"), error.getMessage)
    assert(error.getMessage.contains("remaining"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    assertEquals(calls.callbacks, 0)
