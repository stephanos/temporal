package framework.dsl

class DurabilityFieldsSuite extends munit.FunSuite:

  object Positive:
    opaque type Value = Int
    def apply(value: Int): Value =
      require(value > 0)
      value
    given StateData[Value] = summon[StateData[Int]]
    given Data.Structural[Value] = summon[Data.Structural[Int]]
  final case class Inner(remaining: Int, pending: Boolean)
  final case class Record(inner: Inner, queue: List[UpTo[2]], phase: UpTo[2])

  object Subject extends Machine[Record, Boolean, Nothing]:
    def init: Record = throw AssertionError("init evaluated")
    def end(state: Record): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  object Foreign extends Machine[Record, Boolean, Nothing]:
    def init: Record = throw AssertionError("init evaluated")
    def end(state: Record): Boolean = throw AssertionError("end evaluated")

    object rules extends Bindings()

  private def inner = FieldDurability.durable(Subject, StateField.capture[Record, Inner](_.inner))
  private def queue =
    FieldDurability.ephemeral(Subject, StateField.capture[Record, List[UpTo[2]]](_.queue), Nil)
  private def phase(entries: (UpTo[2], UpTo[2])*) =
    FieldDurability.inMemory(
      Subject,
      StateField.capture[Record, UpTo[2]](_.phase),
      entries.toVector
    )

  test("whole record and collection classification resets only mapped and ephemeral values"):
    val descriptors = Vector(inner, queue, phase(UpTo[2](2) -> UpTo[2](0)))
    FieldDurability.validate(Subject, descriptors, Vector(List("inner", "remaining")))
    val state = Record(Inner(17, true), List(UpTo[2](1)), UpTo[2](2))
    assertEquals(
      descriptors.foldLeft(state)((s, d) => d.reset(s)),
      Record(Inner(17, true), Nil, UpTo[2](0))
    )
    assertEquals(
      phase(UpTo[2](2) -> UpTo[2](0)).reset(state.copy(phase = UpTo[2](1))).phase,
      UpTo[2](1)
    )

  test("coverage, overlap, owner and budget refusals identify their field and location"):
    val pending =
      FieldDurability.durable(Subject, StateField.capture[Record, Boolean](_.inner.pending))
    val foreign = FieldDurability.durable(Foreign, StateField.capture[Record, Inner](_.inner))
    val cases = Vector(
      (Vector(inner, queue), Vector.empty[List[String]], "phase"),
      (Vector(inner, inner, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(inner, pending, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(foreign, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(inner, queue, phase()), Vector(List("phase")), "phase")
    )
    cases.foreach: (descriptors, budgets, field) =>
      val failure =
        intercept[IllegalArgumentException](FieldDurability.validate(Subject, descriptors, budgets))
      assert(failure.getMessage.contains(field), failure.getMessage)
      assert(failure.getMessage.contains("DurabilityFields.test.scala"), failure.getMessage)

  test("inMemory refuses duplicate keys and every fallback into any mapped key"):
    val cases = Vector(
      Vector(UpTo[2](2) -> UpTo[2](0), UpTo[2](2) -> UpTo[2](1)),
      Vector(UpTo[2](2) -> UpTo[2](1), UpTo[2](1) -> UpTo[2](0))
    )
    cases.foreach: entries =>
      val failure = intercept[IllegalArgumentException]:
        FieldDurability.validate(Subject, Vector(inner, queue, phase(entries*)), Vector.empty)
      assert(failure.getMessage.contains("phase"), failure.getMessage)

  test("reset validity uses selected leaf evidence and recursively validates collection members"):
    final case class Selected(value: Int, values: List[Int])
    object Owner extends Machine[Selected, Boolean, Nothing]:
      def init: Selected = throw AssertionError("init evaluated")
      def end(state: Selected): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    given Finite[Int] = Finite.of(1, 3)
    val invalid = FieldDurability.ephemeral(Owner, StateField.capture[Selected, Int](_.value), 2)
    val valid = FieldDurability.ephemeral(Owner, StateField.capture[Selected, Int](_.value), 3)
    val values = FieldDurability.ephemeral(
      Owner,
      StateField.capture[Selected, List[Int]](_.values),
      List(1, 3)
    )
    val invalidValues = FieldDurability.ephemeral(
      Owner,
      StateField.capture[Selected, List[Int]](_.values),
      List(1, 2)
    )
    intercept[IllegalArgumentException](
      FieldDurability.validate(Owner, Vector(invalid, values), Vector.empty)
    ): Unit
    intercept[IllegalArgumentException](
      FieldDurability.validate(Owner, Vector(valid, invalidValues), Vector.empty)
    ): Unit
    FieldDurability.validate(Owner, Vector(valid, values), Vector.empty)
    Vector(Vector(2 -> 1), Vector(1 -> 2)).foreach: entries =>
      val mapped =
        FieldDurability.inMemory(Owner, StateField.capture[Selected, Int](_.value), entries)
      intercept[IllegalArgumentException](
        FieldDurability.validate(Owner, Vector(mapped, values), Vector.empty)
      ): Unit

  test("bounded resets refuse forged out-of-range values independently of constructors"):
    val outside = 3.asInstanceOf[UpTo[2]] // scalafix:ok DisableSyntax.asInstanceOf
    val invalid =
      FieldDurability.ephemeral(Subject, StateField.capture[Record, UpTo[2]](_.phase), outside)
    val failure = intercept[IllegalArgumentException]:
      FieldDurability.validate(Subject, Vector(inner, queue, invalid), Vector.empty)
    assert(failure.getMessage.contains("phase"), failure.getMessage)

  test("a selected narrower bounded catalog retains its membership restriction"):
    given Finite[UpTo[2]] = Finite.of(UpTo[2](0), UpTo[2](1))
    val invalid =
      FieldDurability.ephemeral(Subject, StateField.capture[Record, UpTo[2]](_.phase), UpTo[2](2))
    intercept[IllegalArgumentException]:
      FieldDurability.validate(Subject, Vector(inner, queue, invalid), Vector.empty)

  test("unsupported custom reset evidence refuses with its selected field location"):
    final case class InboxRecord(queue: Inbox[Boolean])
    object InboxOwner extends Machine[InboxRecord, Boolean, Nothing]:
      def init: InboxRecord = throw AssertionError("init evaluated")
      def end(state: InboxRecord): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    val wire = channel[Boolean](capacity = 1, order = Order.fifo, loss = Loss.reliable)
    val descriptor = FieldDurability.ephemeral(
      InboxOwner,
      StateField.capture[InboxRecord, Inbox[Boolean]](_.queue),
      wire.empty
    )
    val failure = intercept[IllegalArgumentException]:
      FieldDurability.validate(InboxOwner, Vector(descriptor), Vector.empty)
    assert(failure.getMessage.contains("queue"), failure.getMessage)
    assert(failure.getMessage.contains("DurabilityFields.test.scala"), failure.getMessage)
    assert(failure.getMessage.contains("unsupported reset validity evidence"), failure.getMessage)

  test("opaque numeric shape needs selected validity evidence rather than its Int representation"):
    final case class OpaqueState(value: Positive.Value)
    object Subject extends Machine[OpaqueState, Boolean, Nothing]:
      def init: OpaqueState = throw AssertionError("init evaluated")
      def end(s: OpaqueState): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings()
    val unsupported = FieldDurability.ephemeral(
      Subject,
      StateField.capture[OpaqueState, Positive.Value](_.value),
      Positive(1)
    )
    val error = intercept[IllegalArgumentException](
      FieldDurability.validate(Subject, Vector(unsupported), Vector.empty)
    )
    assert(error.getMessage.contains("unsupported reset validity evidence"), error.getMessage)
    locally:
      given Finite[Positive.Value] = Finite.of(Positive(1), Positive(2))
      val accepted = FieldDurability.ephemeral(
        Subject,
        StateField.capture[OpaqueState, Positive.Value](_.value),
        Positive(1)
      )
      val refused = FieldDurability.ephemeral(
        Subject,
        StateField.capture[OpaqueState, Positive.Value](_.value),
        Positive(3)
      )
      FieldDurability.validate(Subject, Vector(accepted), Vector.empty)
      intercept[IllegalArgumentException](
        FieldDurability.validate(Subject, Vector(refused), Vector.empty)
      ): Unit
