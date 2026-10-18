package framework.checker

import framework.dsl.*
import framework.dsl.Finite.given

object ParametricActionNamesFixture:
  enum Answer:
    case ok
  given Ok[Answer] = Ok(Answer.ok)

  final class Calls:
    var guards = 0
    var effects = 0
    var claims = 0
    var getters = 0

    class Model(initial: Int) extends Machine[Answer, Nothing]:
      var value = initial

      object user extends Actor:
        lazy val advance = action
        lazy val unrelated: Boolean =
          Calls.this.getters += 1
          true

      object clock extends Internal:
        lazy val tick = step
        lazy val expire = timer

      def unrelated: user.type =
        Calls.this.getters += 1
        user

      def enabled: Boolean =
        Calls.this.guards += 1
        value == 0

      object transitions extends Transitions:
        on(user.advance):
          if enabled then effects.advance()
        on(clock.tick):
          effects.advance()
        on(clock.expire):
          effects.advance()

      object effects:
        def advance(): Unit =
          Calls.this.effects += 1
          value = 1

      object invariants:
        val safe = check:
          Calls.this.claims += 1
          value == 0

class ParametricActionNamesTest extends munit.FunSuite:
  import ParametricActionNamesFixture.*

  test("instance action discovery preserves ownership without evaluating callbacks"):
    val calls = new Calls
    val first = new calls.Model(0)
    val second = new calls.Model(0)
    val firstGraph = NativeCheck.explore(first, first.initialStates)
    val secondGraph = NativeCheck.explore(second, second.initialStates)
    for (machine, graph) <- Vector(first -> firstGraph, second -> secondGraph) do
      assertEquals(
        graph.admitted,
        Vector(machine.user.advance.decl, machine.clock.tick.decl, machine.clock.expire.decl)
      )
      assertEquals(graph.admitted.map(NativeCheck.actionName), Vector("advance", "tick", "expire"))
      assert(graph.admitted.head.actor eq machine.user)
    assert(firstGraph.admitted.zip(secondGraph.admitted).forall((left, right) => !(left eq right)))
    assertEquals((calls.guards, calls.effects, calls.claims, calls.getters), (0, 0, 0, 0))
