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

  object Subject extends Machine[Boolean, Nothing]:
    var inner = Inner(17, true)
    var queue = List(UpTo[2](1))
    var phase = UpTo[2](2)

    object transitions extends Transitions
    def selectedInner =
      StateField.captureVar(inner, this)
    def selectedQueue =
      StateField.captureVar(queue, this)
    def selectedPhase =
      StateField.captureVar(phase, this)
    def selectedPending =
      StateField.captureVar(inner.pending, this)

  object Foreign extends Machine[Boolean, Nothing]:
    var inner = Inner(17, true)
    var queue = List(UpTo[2](1))
    var phase = UpTo[2](2)

    object transitions extends Transitions
    def selectedInner =
      StateField.captureVar(inner, this)

  given Ok[Boolean] = Ok(true)

  private def inner = FieldDurability.durable(Subject, Subject.selectedInner)
  private def queue = FieldDurability.ephemeral(Subject, Subject.selectedQueue, Nil)
  private def phase(entries: (UpTo[2], UpTo[2])*) =
    FieldDurability.inMemory(Subject, Subject.selectedPhase, entries.toVector)

  test("whole record and collection classification resets only mapped and ephemeral values"):
    val descriptors = Vector(inner, queue, phase(UpTo[2](2) -> UpTo[2](0)))
    FieldDurability.validate(Subject, descriptors, Vector(List("inner", "remaining")))
    val state = Subject.init
    assertEquals(
      descriptors.foldLeft(state)((s, d) => d.reset(s)),
      state.copy(queue = List.empty[UpTo[2]], phase = UpTo[2](0))
    )
    val unmapped = state.copy(phase = UpTo[2](1))
    assertEquals(phase(UpTo[2](2) -> UpTo[2](0)).reset(unmapped), unmapped)

  test("coverage, overlap, owner and budget refusals identify their field and location"):
    val pending = FieldDurability.durable(Subject, Subject.selectedPending)
    val foreign = FieldDurability.durable(Foreign, Foreign.selectedInner)
    val cases = Vector(
      (Vector(inner, queue), Vector.empty[List[String]], "phase"),
      (Vector(inner, inner, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(inner, pending, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(foreign, queue, phase()), Vector.empty[List[String]], "inner"),
      (Vector(inner, queue, phase()), Vector(List("phase")), "phase")
    )
    cases.foreach: (descriptors, budgets, field) =>
      val failure = intercept[IllegalArgumentException](
        FieldDurability.validate(Subject, descriptors, budgets)
      )
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
    given Finite[Int] = Finite.of(1, 3)
    object Selected extends Machine[Boolean, Nothing]:
      var value = 1
      var values = List(1, 3)
      object transitions extends Transitions
      def selectedValue =
        StateField.captureVar(value, this)
      def selectedValues =
        StateField.captureVar(values, this)
    val invalid = FieldDurability.ephemeral(Selected, Selected.selectedValue, 2)
    val valid = FieldDurability.ephemeral(Selected, Selected.selectedValue, 3)
    val values = FieldDurability.ephemeral(Selected, Selected.selectedValues, List(1, 3))
    val invalidValues = FieldDurability.ephemeral(Selected, Selected.selectedValues, List(1, 2))
    intercept[IllegalArgumentException](
      FieldDurability.validate(Selected, Vector(invalid, values), Vector.empty)
    ): Unit
    intercept[IllegalArgumentException](
      FieldDurability.validate(Selected, Vector(valid, invalidValues), Vector.empty)
    ): Unit
    FieldDurability.validate(Selected, Vector(valid, values), Vector.empty)
    Vector(Vector(2 -> 1), Vector(1 -> 2)).foreach: entries =>
      val mapped = FieldDurability.inMemory(Selected, Selected.selectedValue, entries)
      intercept[IllegalArgumentException](
        FieldDurability.validate(Selected, Vector(mapped, values), Vector.empty)
      ): Unit

  test("bounded resets refuse forged out-of-range values independently of constructors"):
    val outside = 3.asInstanceOf[UpTo[2]] // scalafix:ok DisableSyntax.asInstanceOf
    val invalid = FieldDurability.ephemeral(Subject, Subject.selectedPhase, outside)
    val failure = intercept[IllegalArgumentException]:
      FieldDurability.validate(Subject, Vector(inner, queue, invalid), Vector.empty)
    assert(failure.getMessage.contains("phase"), failure.getMessage)

  test("a selected narrower bounded catalog retains its membership restriction"):
    given Finite[UpTo[2]] = Finite.of(UpTo[2](0), UpTo[2](1))
    val selected = StateField.captureVar(Subject.phase, Subject)
    val invalid = FieldDurability.ephemeral(Subject, selected, UpTo[2](2))
    intercept[IllegalArgumentException]:
      FieldDurability.validate(Subject, Vector(inner, queue, invalid), Vector.empty)

  test("unsupported custom reset evidence refuses with its selected field location"):
    val wire = channel[Boolean](capacity = 1, order = Order.fifo, loss = Loss.reliable)
    object InboxSubject extends Machine[Boolean, Nothing]:
      var queue = wire.empty
      object transitions extends Transitions
      def selectedQueue =
        StateField.captureVar(queue, this)
    val descriptor = FieldDurability.ephemeral(InboxSubject, InboxSubject.selectedQueue, wire.empty)
    val failure = intercept[IllegalArgumentException]:
      FieldDurability.validate(InboxSubject, Vector(descriptor), Vector.empty)
    assert(failure.getMessage.contains("queue"), failure.getMessage)
    assert(failure.getMessage.contains("DurabilityFields.test.scala"), failure.getMessage)
    assert(failure.getMessage.contains("unsupported reset validity evidence"), failure.getMessage)

  test("opaque numeric shape needs selected validity evidence rather than its Int representation"):
    object OpaqueSubject extends Machine[Boolean, Nothing]:
      var value = Positive(1)
      object transitions extends Transitions
    val unsupported = FieldDurability.ephemeral(
      OpaqueSubject,
      StateField.captureVar(OpaqueSubject.value, OpaqueSubject),
      Positive(1)
    )
    val error = intercept[IllegalArgumentException](
      FieldDurability.validate(OpaqueSubject, Vector(unsupported), Vector.empty)
    )
    assert(error.getMessage.contains("unsupported reset validity evidence"), error.getMessage)
    locally:
      given Finite[Positive.Value] = Finite.of(Positive(1), Positive(2))
      val selected = StateField.captureVar(OpaqueSubject.value, OpaqueSubject)
      val accepted = FieldDurability.ephemeral(OpaqueSubject, selected, Positive(1))
      val refused = FieldDurability.ephemeral(OpaqueSubject, selected, Positive(3))
      FieldDurability.validate(OpaqueSubject, Vector(accepted), Vector.empty)
      intercept[IllegalArgumentException](
        FieldDurability.validate(OpaqueSubject, Vector(refused), Vector.empty)
      ): Unit
