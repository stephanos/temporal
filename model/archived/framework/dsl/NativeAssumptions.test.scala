package framework.dsl

import framework.checker.Effects.*

class NativeAssumptionsSuite extends munit.FunSuite:
  given Ok[Boolean] = Ok(true)
  private val fault = Actor("fault")
  private val loss = action("storageLoss", fault).storageLoss

  private def machine(actions: Action[EmptyTuple]*): MachineDeclaration[State, Boolean, Nothing] =
    new Machine[Boolean, Nothing]:
      @scala.annotation.unused
      var flag = false
      object transitions extends Transitions:
        actions.foreach: target =>
          transition(
            target.decl,
            None,
            "fault.storageLoss",
            () =>
              Vars.fired(); effects.toggle()
          )
      object effects:
        def toggle(): Unit =
          flag = !flag

  test(
    "storage loss assumptions share fault identity across machines copies and repeated bindings"
  ):
    val copied = loss.on(Entity("store")).results("answer")
    val first = machine(loss, loss)
    val second = machine(copied)
    assertEquals(first.assumptions.map(_.name), List("storageLoss"))
    assert(first.assumptions.head.eq(second.assumptions.head))
    val unrelated = action("storageLoss", fault).storageLoss
    val pair = machine(loss, unrelated).assumptions
    assertEquals(pair.map(_.name), List("storageLoss", "storageLoss"))
    assert(!pair.head.eq(pair.last))
    val derived = first.rebind()
    object Wrapped extends Derived(first)
    assert(derived.assumptions.head.eq(first.assumptions.head))
    assert(Wrapped.assumptions.head.eq(first.assumptions.head))
    val binding = first.bindings.head
    val result = effectOf[State, Boolean, Nothing](binding.decl, binding.function)(first.init, Nil)
    assertEquals(result, List(Step(true, first.init.copy(flag = true))))

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
