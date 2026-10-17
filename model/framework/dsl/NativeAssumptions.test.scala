package framework.dsl

class NativeAssumptionsSuite extends munit.FunSuite:
  private val fault = Actor("fault")
  private val tick = action("tick", Actor("caller"))
  private val loss = action("storageLoss", fault).storageLoss

  private def machine(actions: Action[EmptyTuple]*): RecordMachine[Boolean, Boolean, Nothing] =
    new RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = false
      def end(s: Boolean): Boolean = s
      object transitions
          extends Bindings(
            actions.map(a => a ~> ((s: Boolean) => List(Step(true, !s))))*
          )

  test("assuming preserves authored order multiplicity and fairness segments through derivations"):
    val a = framework.dsl.assume("A").fair(tick, tick).fair()
    val b = framework.dsl.assume("B")
    val c = framework.dsl.assume("C")
    val source = machine(tick).assuming(a, b).assuming(c, a)
    object Wrapped extends Derived(source)
    val variants = List(
      source,
      Wrapped,
      source.restrict(tick),
      source.rebind(tick ~> ((s: Boolean) => List(Step(true, s)))),
      source.extend(loss ~> ((s: Boolean) => List(Step(true, s)))).restrict(tick),
      source.refining(machine(tick))(identity),
      source.unmonitored
    )
    for m <- variants do
      assertEquals(m.assumptions, List(a, b, c, a))
      assertEquals(m.assumptions.head.fairness, List(tick.decl, tick.decl))
      assertEquals(
        m.assumptions.head.fairSegments,
        Vector(
          Assumption.Fairness.Actions(List(tick.decl, tick.decl)),
          Assumption.Fairness.Actions(Nil)
        )
      )

  test(
    "storage loss assumptions share fault identity across machines copies and repeated bindings"
  ):
    val authored = framework.dsl.assume("opaque")
    val later = framework.dsl.assume("later")
    val copied = loss.on(Entity("store")).results("answer")
    val first = machine(loss, loss, tick).assuming(authored)
    val second = machine(copied)
    assertEquals(first.assumptions.map(_.name), List("opaque", "storageLoss"))
    assert(first.assumptions.last.eq(second.assumptions.head))
    assertEquals(
      first.assuming(later).assumptions.map(_.name),
      List("opaque", "later", "storageLoss")
    )
    val unrelated = action("storageLoss", fault).storageLoss
    val pair = machine(loss, unrelated).assumptions
    assertEquals(pair.map(_.name), List("storageLoss", "storageLoss"))
    assert(!pair.head.eq(pair.last))
    assertEquals(first.restrict(tick).assumptions, List(authored))
    assert(
      first
        .rebind(loss ~> ((s: Boolean) => List(Step(true, s))))
        .assumptions
        .last
        .eq(second.assumptions.head)
    )
    val binding = first.bindings.head
    val result = effectOf[Boolean, Boolean, Nothing](binding.decl, binding.function)(false, Nil)
    assertEquals(result, List(Step(true, true)))

  test("authored storage-loss collisions refuse at acquisition before callbacks"):
    val generated = machine(loss).assumptions.head
    for duplicate <- List(framework.dsl.assume("storageLoss"), generated) do
      val source = new RecordMachine[Boolean, Boolean, Nothing]:
        def init: Boolean = throw AssertionError("init evaluated")
        def end(s: Boolean): Boolean = throw AssertionError("end evaluated")
        assumes(duplicate)
        object transitions
            extends Bindings(
              loss ~> ((_: Boolean) => throw AssertionError("effect evaluated"))
            )
      val error = intercept[IllegalArgumentException](source.bindings)
      assert(error.getMessage.contains("storageLoss"), error.getMessage)
      assert(error.getMessage.contains("NativeAssumptions.test.scala"), error.getMessage)
    val colliding = machine(tick)
      .assuming(framework.dsl.assume("storageLoss"))
      .extend(loss ~> ((s: Boolean) => List(Step(true, s))))
    intercept[IllegalArgumentException](colliding.bindings): Unit

  test("inferred storage-loss names belong to the definition across aliases and acquisition order"):
    object faults:
      val erased = action(fault).storageLoss
    object aliases:
      val first = faults.erased.on(Entity("first"))
      val second = faults.erased.on(Entity("second"))
    val first = machine(aliases.first)
    val second = machine(aliases.second)
    assertEquals(second.assumptions.map(_.name), List("erased"))
    assertEquals(first.assumptions.map(_.name), List("erased"))
    assert(first.assumptions.head.eq(second.assumptions.head))

  test("inferred authored assumptions remain usable beside generated storage-loss assumptions"):
    val opaque = framework.dsl.assume
    val subject = machine(loss).assuming(opaque)
    assertEquals(subject.assumptions.map(_.sourceName), List("opaque", "storageLoss"))
