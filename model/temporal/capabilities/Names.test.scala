package framework.dsl

import temporal.capabilities.system.{Expirable, Retryable}

class CapabilityNamesSuite extends munit.FunSuite:
  test("compiled capability classes and their companions expose the same confirmed kind"):
    assertEquals(Retryable.name, "Retryable")
    assertEquals(Expirable.name, "Expirable")
    assertEquals(
      classOf[Retryable[Unit, Unit, 1]].getName,
      "temporal.capabilities.system.Retryable"
    )
    assertEquals(
      classOf[Expirable[Unit, Unit, Unit, Tuple1[Boolean]]].getName,
      "temporal.capabilities.system.Expirable"
    )
