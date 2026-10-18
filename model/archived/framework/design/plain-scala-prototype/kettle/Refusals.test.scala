package proto

import plain.*

// The compile-time refusals the native DSL gives without a compiler plugin, as typeCheckErrors
// reports them (munit's compileErrors).
class RefusalsTest extends munit.FunSuite:
  private inline def refused(inline code: String): String =
    val errors = compileErrors(code)
    assert(errors.nonEmpty, s"compiled: $code")
    errors

  private val placement =
    "an action is declared in an Actor object; a step the system takes on its own in an Internal object"

  test("an action outside an Actor object, or a step in one, is refused at the declaration"):
    val loose = refused("object loose { val a = action }")
    assert(loose.contains(placement), loose)
    assert(refused("object hand extends Actor { val a = step }").contains(placement))
    assert(refused("object heater extends Internal { val a = action }").contains(placement))

  test("before outside postconditions is refused"):
    val errors = refused(
      "object M extends Machine:\n  var x = 1\n  object invariants:\n    val a = check(before.x == 1)\n"
    )
    assert(
      errors.contains("before is the state a step started from; use it in a postcondition"),
      errors
    )

  test("before is the typed State record: a field the Machine does not declare is a type error"):
    val errors = refused(
      "object M extends Machine:\n  var x = 1\n  object postconditions:\n    val a = check(before.y == 1)\n"
    )
    assert(errors.contains("value y is not a member of plain.Record"), errors)
    val typed = refused(
      "object M extends Machine:\n  var x = 1\n  object postconditions:\n    val a = check(before.x)\n"
    )
    assert(typed.contains("Required: Boolean"), typed)

  test("an assignment in a transition is refused: the state change belongs in effects"):
    val errors = refused(
      "object M extends Machine:\n  var x = 1\n  object transitions:\n    on(Kettle.user.pressButton):\n      if x > 0 then x = 2\n"
    )
    assert(errors.contains("a transition says when; write the state change in effects"), errors)

  test("a transition leaf that is not an effect call, a rejection or () is refused"):
    val errors = refused(
      "object M extends Machine:\n  var x = 1\n  object transitions:\n    on(Kettle.user.pressButton):\n      if x > 0 then println(x)\n"
    )
    assert(
      errors.contains(
        "a transition ends in an effect call, a rejection or (): println(M.this.x) is none of these"
      ),
      errors
    )

  test("on outside a transitions object is refused"):
    val errors = refused(
      "object M extends Machine:\n  var x = 1\n  object effects:\n    on(Kettle.user.pressButton)(())\n"
    )
    assert(
      errors.contains("on(action) is declared in a Machine's or a facet's transitions object"),
      errors
    )

  test("a Machine with no vars is refused where the runtime reads its schema"):
    val errors = refused(
      "object M extends Machine\nval n = Runtime(M)\n"
    )
    assert(errors.contains("a Machine declares at least one state var"), errors)
    assert(Runtime(Kettle.Model).schema.nonEmpty)

  test("a non-exhaustive phase match is the compiler's warning, an error under -Werror"):
    // typeCheckErrors reports typer errors only; exhaustiveness is checked after it (see the note)
    val errors = compileErrors(
      "object M extends Machine:\n  var light = Lamps.Lit.off\n  object transitions:\n    on(Lamps.hand.press):\n      light match\n        case Lamps.Lit.off => effects.on()\n  object effects:\n    def on() = light = Lamps.Lit.on\n"
    )
    assertEquals(errors, "")
