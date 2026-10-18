package framework.dsl

import framework.checker.Effects.*

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
    def machine(a: Action[EmptyTuple]): MachineDeclaration[State, Boolean, Nothing] =
      new Machine[Boolean, Nothing]:
        @scala.annotation.unused
        var flag = false
        override def end: Boolean = throw AssertionError("end evaluated")
        object transitions extends Transitions(using summon, Ok(true), summon):
          transition(a.decl, None, "caller.action", () => throw AssertionError("effect evaluated"))
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
