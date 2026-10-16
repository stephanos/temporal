package framework.dsl

class NativeFaultSuite extends munit.FunSuite:
  private val fault = Actor("fault")
  private val caller = Actor("caller")

  test("fault kinds retain the original declaration and survive modifiers"):
    val crash = action("restart", fault)
    val declaration = crash.decl
    assert(crash.crash eq crash)
    assert(crash.decl eq declaration)
    assertEquals(crash.decl.faultKind, Some(FaultKind.crash))
    val loss = action("lost", fault).responseLoss.modelOnly(because = "no native control")
    val storage = action("storage", fault).storageLoss
    final case class Input(enabled: Boolean = false)
    val record: RecordAction[Input] = action("record", fault).inputs[Input].responseLoss
    val scalar: ScalarAction[Boolean] =
      action("scalar", fault).input[Boolean](name = "enabled", default = false).storageLoss
    assert(record(enabled = true).decl eq record.decl)
    assert(scalar(true).decl eq scalar.decl)
    assertEquals(record().values, List(false))
    assertEquals(scalar().values, List(false))
    for a <- Vector(crash, loss, storage) do
      a.decl.validateNativeFault("test")
      val copied = a.on(Entity("operation")).results("answer").example(EmptyTuple, "empty")
      assert(copied.decl.actor eq fault)
      assertEquals(copied.decl.faultKind, a.decl.faultKind)
      assertEquals(copied.decl.modelOnlyReason, a.decl.modelOnlyReason)

  test("completed binding acquisition refuses missing duplicate and invalid fault metadata"):
    def machine(a: Action[EmptyTuple]): Machine[Boolean, Boolean, Nothing] =
      new Machine[Boolean, Boolean, Nothing]:
        def init: Boolean = throw AssertionError("init evaluated")
        def end(s: Boolean): Boolean = throw AssertionError("end evaluated")
        object rules
            extends Bindings(a ~> ((_: Boolean) => throw AssertionError("effect evaluated")))
    val ordinary = machine(action("ordinary", caller))
    assertEquals(ordinary.bindings.size, 1)
    val invalid = Vector(
      action("missing", fault) -> "kind",
      action("duplicate", fault).crash.storageLoss -> "kind",
      action("reason", fault).responseLoss.modelOnly(because = " ") -> "reason",
      action("twice", fault).crash.modelOnly(because = "a").modelOnly(because = "b") -> "modelOnly",
      action("wrongActor", caller).crash -> "fault"
    )
    for (a, message) <- invalid do
      val m = machine(a)
      val error = intercept[IllegalArgumentException](m.bindings)
      assert(error.getMessage.contains(message), error.getMessage)
      assert(error.getMessage.contains(a.name), error.getMessage)
      assert(error.getMessage.contains("NativeFault.test.scala"), error.getMessage)

  test("native choice programs declare deferred ordered alternatives including Nil"):
    val first = choice
    val second = choice
    var called = Vector.empty[Int]
    val program = choose[Boolean, Boolean, Nothing](
      first -> ((s: Boolean) =>
        called :+= 1; List(Step(true, !s))
      ),
      second -> ((_: Boolean) =>
        called :+= 2; Nil
      )
    )
    assertEquals(called, Vector.empty)
    assert(program.tokens(0) eq first)
    assert(program.tokens(1) eq second)
    assertEquals(program(false), List(Step(true, true)))
    assertEquals(called, Vector(1, 2))
    intercept[IllegalArgumentException](
      choose[Boolean, Boolean, Nothing](
        first -> ((s: Boolean) => List(Step(true, s))),
        first -> ((s: Boolean) => List(Step(true, s)))
      )
    ): Unit
    val tooMany = choose[Boolean, Boolean, Nothing](
      first -> ((s: Boolean) => List(Step(true, s), Step(false, s))),
      second -> ((_: Boolean) => Nil)
    )
    intercept[IllegalArgumentException](tooMany(false))
