package framework.dsl

import framework.dsl.Finite.given

object ParametricBoundsFixture:
  given Ok[Boolean] = Ok(true)

  class Model(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
    var ready = false

    object transitions extends Transitions

    object progress:
      val direct =
        leadsTo(from = true, to = ready, within = resourceManagers.size)
      val deadline =
        whenever(true):
          eventually(ready, within = resourceManagers.size)

  class DeadlineOnly(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
    var ready = false

    object transitions extends Transitions

    object progress:
      val deadline =
        whenever(true):
          eventually(ready, within = resourceManagers.size)

class ParametricBoundsTest extends munit.FunSuite:
  test("constructor-derived bounds are captured per instance and remain positive"):
    for size <- List(3, 4) do
      val model = new ParametricBoundsFixture.Model((1 to size).toSet)
      assertEquals(model.progress.direct.within, Some(size))
      assertEquals(model.progress.deadline.within, Some(size))
    val direct = new ParametricBoundsFixture.Model(Set.empty)
    val deadline = new ParametricBoundsFixture.DeadlineOnly(Set.empty)
    val directError = intercept[IllegalArgumentException](direct.progress)
    val deadlineError = intercept[IllegalArgumentException](deadline.progress)
    assert(directError.getMessage.contains("within is a positive number of steps"))
    assert(deadlineError.getMessage.contains("within is a positive number of steps"))

  test("progress bounds admit an immutable constructor set's size"):
    val direct = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object progress:
          val direct = leadsTo(from = true, to = ready, within = resourceManagers.size)
    """)
    val deadline = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object progress:
          val deadline = whenever(true):
            eventually(ready, within = resourceManagers.size)
    """)
    assertEquals(direct, "")
    assertEquals(deadline, "")

  test("state-dependent and arbitrary callback bounds remain refused"):
    val state = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
        var remaining = resourceManagers
        object transitions extends Transitions
        object progress:
          val changes = leadsTo(from = true, to = false, within = remaining.size)
    """)
    val callback = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model(resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        def bound(): Int = resourceManagers.size
        object progress:
          val calls = whenever(true):
            eventually(ready, within = bound())
    """)
    val mutableConstructor = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model(var resourceManagers: Set[Int]) extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object progress:
          val changes = leadsTo(from = true, to = ready, within = resourceManagers.size)
    """)
    val negative = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      object Model extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object progress:
          val negative = whenever(true):
            eventually(ready, within = -1)
    """)
    for refusal <- List(state, callback, mutableConstructor, negative) do
      assert(refusal.contains("within is a positive number of steps"), refusal)
