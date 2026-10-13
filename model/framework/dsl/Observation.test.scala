package framework.dsl

import scala.compiletime.testing.typeCheckErrors

class ObservationTest extends munit.FunSuite:
  test("Observation has only name/on and preserves explicit and default native values"):
    val order = Entity("order", "orderId")
    val inferred = Observation(on = order)
    val explicit = Observation("attempts", order)
    assertEquals(inferred.productElementNames.toList, List("name", "on"))
    assertEquals(inferred.productArity, 2)
    assertEquals(inferred.name, "")
    assert(inferred.on eq order)
    assertEquals(explicit.name, "attempts")
    assert(explicit.on eq order)

  test("retired named and positional read arguments are compile errors"):
    val named = typeCheckErrors(
      "framework.dsl.Observation(name = \"attempts\", on = framework.dsl.Entity(), read = \"info.attempt\")"
    )
    val positional =
      typeCheckErrors("framework.dsl.Observation(\"attempts\", framework.dsl.Entity(), \"info.attempt\")")
    assertEquals(
      named.map(_.message).filter(_.contains("does not have a parameter read")).size,
      1,
      named
    )
    assert(
      named.forall(e =>
        e.message.contains("does not have a parameter read") || e.message.contains(
          "too many arguments"
        )
      ),
      named
    )
    assertEquals(positional.size, 1, positional)
    assert(positional.head.message.toLowerCase.contains("too many arguments"), positional)
