package framework.dsl

class NativeCrashSuite extends munit.FunSuite:
  enum Custody derives CanEqual:
    case idle, buffered, reserved, stored
  final case class CrashState(
      kept: Int,
      pending: Int,
      custody: Custody,
      available: Boolean,
      remaining: Int,
      bounded: UpTo[2],
      other: Int
  )

  object faults:
    val crash = action("crash", Actor("fault")).crash
    val second = action("second", Actor("fault")).responseLoss

  class Subject(mode: Int) extends Machine[CrashState, Boolean, Boolean]:
    def init: CrashState = CrashState(13, 8, Custody.buffered, true, 2, UpTo(2), 7)
    def end(s: CrashState): Boolean = false
    durability(
      durable(_.kept),
      ephemeral(_.pending, resetTo = 0),
      inMemory(_.custody)(Custody.buffered -> Custody.idle, Custody.reserved -> Custody.stored),
      durable(_.available),
      durable(_.remaining),
      durable(_.bounded),
      durable(_.other)
    )

    object rules extends Rules:
      mode match
        case 0 => crashes(faults.crash, false, true, false)
        case 1 => crashes(faults.crash.budgetedBy(_.available), false, true, false)
        case 2 => crashes(faults.crash.budgetedBy(_.remaining), false, true, false)
        case _ => crashes(faults.crash.budgetedBy(_.bounded), false, true, false)
      on(faults.second.budgetedBy(_.other)):
        always ~> ((s: CrashState) => List(Step(true, s)))

  private def run(
      machine: Machine[CrashState, Boolean, Boolean],
      state: CrashState
  ): List[Step[CrashState, Boolean, Boolean]] =
    val binding = machine.bindings.find(_.decl.eq(faults.crash.decl)).get
    effectOf[CrashState, Boolean, Boolean](binding.decl, binding.function)(state, Nil)

  private def ordinaryCrash(state: CrashState): List[Step[CrashState, Boolean, Boolean]] =
    val custody = state.custody match
      case Custody.idle     => Custody.idle
      case Custody.buffered => Custody.idle
      case Custody.reserved => Custody.stored
      case Custody.stored   => Custody.stored
    List(Step(false, state.copy(pending = 0, custody = custody), List(true, false)))

  test("a generated crash reconstructs sibling resets as one valid record"):
    final case class Paired(left: Int, right: Int):
      require(left == right)
    object PairedSubject extends Machine[Paired, Boolean, Boolean]:
      def init: Paired = Paired(1, 1)
      def end(state: Paired): Boolean = false
      durability(ephemeral(_.left, 0), ephemeral(_.right, 0))
      object rules extends Bindings(crashes(faults.crash, true))
    val binding = PairedSubject.bindings.head
    val result = effectOf[Paired, Boolean, Boolean](binding.decl, binding.function)(
      Paired(1, 1), Nil
    )
    assertEquals(result, List(Step(true, Paired(0, 0))))

  test("a budgeted crash reconstructs nested resets and its own budget atomically"):
    final case class Paired(left: Int, right: Int):
      require(left == right)
    final case class Coupled(pair: Paired, remaining: Int):
      require(pair.left == remaining)
    object CoupledSubject extends Machine[Coupled, Boolean, Boolean]:
      def init: Coupled = Coupled(Paired(1, 1), 1)
      def end(state: Coupled): Boolean = false
      durability(
        ephemeral(_.pair.left, 0),
        ephemeral(_.pair.right, 0),
        durable(_.remaining)
      )
      object rules extends Bindings(crashes(faults.crash.budgetedBy(_.remaining), true))
    val binding = CoupledSubject.bindings.head
    val execute = effectOf[Coupled, Boolean, Boolean](binding.decl, binding.function)
    assertEquals(execute(Coupled(Paired(1, 1), 1), Nil), List(Step(true, Coupled(Paired(0, 0), 0))))
    assertEquals(execute(Coupled(Paired(0, 0), 0), Nil), Nil)

  test("derived crash matches ordinary independent field reset including unmapped values"):
    val machine = new Subject(0)
    for (from, to) <- List(
        Custody.idle -> Custody.idle,
        Custody.buffered -> Custody.idle,
        Custody.reserved -> Custody.stored,
        Custody.stored -> Custody.stored
      )
    do
      val state = CrashState(13, 8, from, true, 2, UpTo(2), 7)
      assertEquals(
        run(machine, state),
        List(Step(false, CrashState(13, 0, to, true, 2, UpTo(2), 7), List(true, false)))
      )
      assertEquals(run(machine, state), ordinaryCrash(state))

  test("generated own budgets consume once preserving every other budget and durable value"):
    val state = CrashState(13, 8, Custody.reserved, true, 2, UpTo(2), 7)
    val expected = List(
      CrashState(13, 0, Custody.stored, false, 2, UpTo(2), 7),
      CrashState(13, 0, Custody.stored, true, 1, UpTo(2), 7),
      CrashState(13, 0, Custody.stored, true, 2, UpTo(1), 7)
    )
    for (mode, next) <- List(1, 2, 3).zip(expected) do
      assertEquals(run(new Subject(mode), state), List(Step(false, next, List(true, false))))
      assertEquals(
        run(new Subject(mode), state.copy(other = -7)),
        List(Step(false, next.copy(other = -7), List(true, false)))
      )
    assertEquals(run(new Subject(1), state.copy(available = false)), Nil)
    assertEquals(run(new Subject(2), state.copy(remaining = 0)), Nil)
    assertEquals(run(new Subject(3), state.copy(bounded = UpTo(0))), Nil)

  test(
    "negative own Int refuses with binding location while authored overrides consume themselves"
  ):
    val machine = new Subject(2)
    val error = intercept[IllegalArgumentException](run(machine, machine.init.copy(remaining = -1)))
    assert(error.getMessage.contains("crash"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    val overridden = machine.rebind(
      faults.crash ~> ((s: CrashState) => List(Step(true, s.copy(remaining = s.remaining - 1))))
    )
    assertEquals(
      run(overridden, machine.init),
      List(Step(true, CrashState(13, 8, Custody.buffered, true, 1, UpTo(2), 7)))
    )
    assertEquals(run(overridden, machine.init.copy(remaining = -1)).head.state.remaining, -2)
    val binding = machine.bindings.find(_.decl.eq(faults.second.decl)).get
    val authored = effectOf[CrashState, Boolean, Boolean](binding.decl, binding.function)
    val negativeOther = machine.init.copy(other = -1)
    assertEquals(authored(negativeOther, Nil), List(Step(true, negativeOther)))

  test("never-invoked crash and orphan classifications refuse at completed acquisition"):
    class Missing extends Machine[CrashState, Boolean, Nothing]:
      def init: CrashState = throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules:
        crashes(faults.crash, true)
    val missing = new Missing
    val error = intercept[IllegalArgumentException](missing.bindings)
    assert(error.getMessage.contains("classification"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    class Orphan extends Machine[CrashState, Boolean, Nothing]:
      def init: CrashState = throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean = throw AssertionError("end evaluated")
      durability(durable(_.kept))
      object rules extends Bindings()
    intercept[IllegalArgumentException](new Orphan().bindings): Unit

  test("crash-free restrictions retain valid classification without hiding derived invalid fields"):
    val restricted = new Subject(0).restrict(faults.second)
    assertEquals(restricted.bindings.map(_.decl), List(faults.second.decl))
    object Invalid extends Derived(restricted):
      durability(ephemeral(_.pending, resetTo = 0))
    val error = intercept[IllegalArgumentException](Invalid.bindings)
    assert(error.getMessage.contains("requires a crash binding"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    object Plain extends Machine[CrashState, Boolean, Boolean]:
      def init: CrashState = throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    object NewlyClassified extends Derived(Plain):
      durability(
        durable(_.kept),
        durable(_.pending),
        durable(_.custody),
        durable(_.available),
        durable(_.remaining),
        durable(_.bounded),
        durable(_.other)
      )
    val orphan = intercept[IllegalArgumentException](NewlyClassified.bindings)
    assert(orphan.getMessage.contains("requires a crash binding"), orphan.getMessage)
    val authored = action("authored", Actor("fault")).responseLoss
    val extended = restricted.extend(
      authored.budgetedBy(_.pending) ~> ((s: CrashState) => List(Step(true, s)))
    )
    val binding = extended.bindings.find(_.decl.eq(authored.decl)).get
    val state = CrashState(13, -4, Custody.buffered, true, 2, UpTo(2), 7)
    assertEquals(
      effectOf[CrashState, Boolean, Boolean](binding.decl, binding.function)(state, Nil),
      List(Step(true, state))
    )

  test("exhausted and negative budgets stop before a poison reset while positives reach it"):
    final case class PoisonState(
        available: Boolean,
        remaining: Int,
        bounded: UpTo[2],
        pending: Boolean
    )
    class Poison(mode: Int) extends Machine[PoisonState, Boolean, Nothing]:
      var resets = 0
      def init: PoisonState = throw AssertionError("init evaluated")
      def end(s: PoisonState): Boolean = throw AssertionError("end evaluated")
      private val selected = StateField.capture[PoisonState, Boolean](_.pending)
      private val poisoned = StateField.make[PoisonState, Boolean](
        selected.ownerType,
        selected.fieldType,
        selected.path,
        selected.location,
        selected.rootFields,
        selected.valid,
        selected.get,
        (_, _) =>
          resets += 1; throw AssertionError("reset evaluated")
      )
      durability(
        durable(_.available),
        durable(_.remaining),
        durable(_.bounded),
        FieldDurability.ephemeral(this, poisoned, false)
      )
      object rules extends Rules:
        mode match
          case 1 => crashes(faults.crash.budgetedBy(_.available), true)
          case 2 => crashes(faults.crash.budgetedBy(_.remaining), true)
          case _ => crashes(faults.crash.budgetedBy(_.bounded), true)
    val exhausted = PoisonState(false, 0, UpTo(0), true)
    for mode <- List(1, 2, 3) do
      val subject = new Poison(mode)
      val binding = subject.bindings.head
      val effect = effectOf[PoisonState, Boolean, Nothing](binding.decl, binding.function)
      assertEquals(effect(exhausted, Nil), Nil)
      assertEquals(subject.resets, 0)
      if mode == 2 then
        val error = intercept[IllegalArgumentException](effect(exhausted.copy(remaining = -1), Nil))
        assert(error.getMessage.contains("negative own budget"), error.getMessage)
        assertEquals(subject.resets, 0)
      val positive = PoisonState(true, 1, UpTo(1), true)
      assertEquals(intercept[AssertionError](effect(positive, Nil)).getMessage, "reset evaluated")
      assertEquals(subject.resets, 1)

  test("completed acquisition refuses duplicate generated crash bindings"):
    class Duplicate extends Machine[CrashState, Boolean, Boolean]:
      def init: CrashState = throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean = throw AssertionError("end evaluated")
      durability(
        durable(_.kept),
        durable(_.pending),
        durable(_.custody),
        durable(_.available),
        durable(_.remaining),
        durable(_.bounded),
        durable(_.other)
      )
      object rules extends Bindings(crashes(faults.crash, true), crashes(faults.crash, false))
    val error = intercept[IllegalArgumentException](new Duplicate().bindings)
    assert(error.getMessage.contains("crash"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)

  test("completed metadata rejects malformed dormant classifications before any callback"):
    final case class InvalidState(value: UpTo[2], pending: Boolean)
    class Invalid(mode: Int) extends Machine[InvalidState, Boolean, Nothing]:
      def init: InvalidState = throw AssertionError("init evaluated")
      def end(s: InvalidState): Boolean = throw AssertionError("end evaluated")
      mode match
        case 0 => durability(durable(_.value))
        case 1 => durability(durable(_.value), durable(_.value), durable(_.pending))
        case 2 =>
          durability(
            ephemeral(_.value, resetTo = 3.asInstanceOf[UpTo[2]]),
            durable(_.pending)
          ) // scalafix:ok DisableSyntax.asInstanceOf
        case 3 =>
          durability(
            inMemory(_.value)(UpTo[2](2) -> UpTo[2](1), UpTo[2](1) -> UpTo[2](0)),
            durable(_.pending)
          )
        case _ => durability(ephemeral(_.value, resetTo = UpTo[2](0)), durable(_.pending))
      object rules
          extends Bindings(
            faults.crash.budgetedBy(_.value) ~>
              ((_: InvalidState) => throw AssertionError("effect evaluated"))
          )
    for mode <- 0 to 4 do
      val subject = new Invalid(mode)
      val error = intercept[IllegalArgumentException](subject.nativeBudgets)
      assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)

  test("nested field resets and own-budget writes compose in the same native transition"):
    final case class Inner(remaining: Int, pending: Boolean)
    final case class Nested(inner: Inner, kept: Int)
    object Subject extends Machine[Nested, Boolean, Nothing]:
      def init: Nested = throw AssertionError("init evaluated")
      def end(s: Nested): Boolean = throw AssertionError("end evaluated")
      durability(durable(_.inner.remaining), ephemeral(_.inner.pending, false), durable(_.kept))
      object rules extends Rules:
        crashes(faults.crash.budgetedBy(_.inner.remaining), true)
    val binding = Subject.bindings.head
    val result = effectOf[Nested, Boolean, Nothing](binding.decl, binding.function)(
      Nested(Inner(2, true), 15),
      Nil
    )
    assertEquals(result, List(Step(true, Nested(Inner(1, false), 15))))

  test("empty custom realization acquisition rejects dormant invalid crashes before dependencies"):
    import framework.realize.{ConstructedRealization, Performance}
    var callbacks = 0
    var exposedDependencies = 0
    class Admission(valid: Boolean) extends Machine[CrashState, Boolean, Boolean]:
      def init: CrashState =
        callbacks += 1; throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean =
        callbacks += 1; throw AssertionError("end evaluated")
      if valid then
        durability(
          durable(_.kept),
          durable(_.pending),
          durable(_.custody),
          durable(_.available),
          durable(_.remaining),
          durable(_.bounded),
          durable(_.other)
        )
      object rules extends Rules:
        crashes(faults.crash, true)
    class Kit(val machine: Machine[?, ?, ?]) extends ConstructedRealization:
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
    assertEquals(callbacks, 0)

  test("generated own budgets refuse active aliases while authored overrides retain shared paths"):
    var callbacks = 0
    class Subject(shared: Boolean) extends Machine[CrashState, Boolean, Boolean]:
      def init: CrashState = throw AssertionError("init evaluated")
      def end(s: CrashState): Boolean = throw AssertionError("end evaluated")
      durability(
        durable(_.kept),
        ephemeral(_.pending, 0),
        durable(_.custody),
        durable(_.available),
        durable(_.remaining),
        durable(_.bounded),
        durable(_.other)
      )
      object rules
          extends Bindings(
            crashes(faults.crash.budgetedBy(_.remaining), false),
            (if shared then faults.second.budgetedBy(_.remaining)
             else faults.second.budgetedBy(_.other)) ~>
              ((_: CrashState) =>
                callbacks += 1; throw AssertionError("effect evaluated")
              )
          )
    val distinct = new Subject(false)
    assertEquals(distinct.nativeBudgets.size, 2)
    val extra = action("extra", Actor("fault")).responseLoss
    val authored = distinct
      .rebind(faults.crash ~> ((_: CrashState) =>
        callbacks += 1; throw AssertionError("override evaluated")
      ))
      .extend(extra.budgetedBy(_.remaining) ~> ((_: CrashState) =>
        callbacks += 1; throw AssertionError("extra evaluated")
      ))
    assertEquals(authored.nativeBudgets.size, 3)
    val error = intercept[IllegalArgumentException](new Subject(true).nativeBudgets)
    assert(error.getMessage.contains("aliases another fault budget"), error.getMessage)
    assert(error.getMessage.contains("remaining"), error.getMessage)
    assert(error.getMessage.contains("NativeCrash.test.scala"), error.getMessage)
    assertEquals(callbacks, 0)
