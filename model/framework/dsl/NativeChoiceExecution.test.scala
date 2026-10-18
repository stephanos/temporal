package framework.dsl

import framework.checker.Effects.*

class NativeChoiceExecutionSuite extends munit.FunSuite:
  given Ok[Boolean] = Ok(true)

  private def run(machine: MachineDeclaration[State, Boolean, Boolean], cls: Class) =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[State, Boolean, Boolean](binding.decl, binding.function)(machine.init, cls)

  test("input effects receive actual records and scalar inputs through native derivation adapters"):
    final case class Request(flag: Boolean)
    val supplied = Request(true)
    val seen = scala.collection.mutable.ArrayBuffer.empty[Request]
    object caller:
      private val actor = Actor("caller")
      val record = action("record", actor).inputs[Request]
      val scalar = action("scalar", actor).input[Boolean](name = "flag")
    object Subject extends Machine[Boolean, Boolean]:
      var ready = false
      object effects:
        val first = choice
        val second = choice
        val scalarFirst = choice
        val scalarSecond = choice
        def received(in: Request): Unit =
          seen += in
          choose(
            first -> { ready = true; record(in.flag) },
            second -> record(false)
          )
        def scalar(in: Boolean): Unit =
          choose(
            scalarFirst -> { ready = true; record(in) },
            scalarSecond -> record(!in)
          )
      object transitions extends Transitions:
        on(caller.record): in =>
          effects.received(in)
        on(caller.scalar): in =>
          effects.scalar(in)
    for machine <- Vector(Subject, Subject.unmonitored) do
      seen.clear()
      machine.validateNativeChoices(caller.record(supplied), Vector(Subject.effects.first))
      machine.validateNativeChoices(caller.scalar(false), Vector(Subject.effects.scalarSecond))
      assertEquals(seen.toList, Nil)
      assertEquals(
        run(machine, caller.record(supplied)),
        List(
          Step(true, Subject.init.copy(ready = true), List(true)),
          Step(true, Subject.init, List(false))
        )
      )
      assertEquals(seen.size, 2)
      assert(seen.forall(_ eq supplied))
      for input <- Vector(false, true) do
        assertEquals(
          run(machine, caller.scalar(input)),
          List(
            Step(true, Subject.init.copy(ready = true), List(input)),
            Step(true, Subject.init, List(!input))
          )
        )

  test(
    "class filtering survives derivation and native rebinds replace their exact choice inventory"
  ):
    object caller:
      private val actor = Actor("caller")
      val selected = action("selected", actor).input[Boolean](name = "flag")
    var offCalls = 0
    var onCalls = 0
    object Subject extends Machine[Boolean, Boolean]:
      var ready = false
      object effects:
        val offToken = choice
        val offOther = choice
        val onToken = choice
        val onOther = choice
        val reboundToken = choice
        val reboundOther = choice
        def off(): Unit =
          offCalls += 1
          choose(
            offToken -> { ready = true; record(false) },
            offOther -> record(true)
          )
        def on(): Unit =
          onCalls += 1
          choose(
            onToken -> { ready = true; record(true) },
            onOther -> record(false)
          )
        def replacement(in: Boolean): Unit =
          choose(
            reboundToken -> record(in),
            reboundOther -> record(!in)
          )
        def plain(in: Boolean): Unit =
          record(!in)
      object transitions extends Transitions:
        on(caller.selected(false)):
          effects.off()
        on(caller.selected(true)):
          effects.on()
    for machine <- Vector(Subject, Subject.unmonitored) do
      offCalls = 0
      onCalls = 0
      machine.validateNativeChoices(caller.selected(false), Vector(Subject.effects.offToken))
      intercept[IllegalArgumentException](
        machine.validateNativeChoices(caller.selected(false), Vector(Subject.effects.onToken))
      ): Unit
      assertEquals(
        run(machine, caller.selected(false)),
        List(
          Step(true, Subject.init.copy(ready = true), List(false)),
          Step(true, Subject.init, List(true))
        )
      )
      assertEquals((offCalls, onCalls), (2, 0))
      assertEquals(
        run(machine, caller.selected(true)),
        List(
          Step(true, Subject.init.copy(ready = true), List(true)),
          Step(true, Subject.init, List(false))
        )
      )
      assertEquals((offCalls, onCalls), (2, 2))
    val replaced = Subject.rebind(on(caller.selected): in =>
      Subject.effects.replacement(in))
    for input <- Vector(false, true) do
      replaced.validateNativeChoices(caller.selected(input), Vector(Subject.effects.reboundToken))
      intercept[IllegalArgumentException](
        replaced.validateNativeChoices(caller.selected(input), Vector(Subject.effects.offToken))
      ): Unit
      assertEquals(
        run(replaced, caller.selected(input)),
        List(
          Step(true, Subject.init, List(input)),
          Step(true, Subject.init, List(!input))
        )
      )
    val plain = Subject.rebind(on(caller.selected): in =>
      Subject.effects.plain(in))
    for (input, token) <- Vector(false -> Subject.effects.offToken, true -> Subject.effects.onToken)
    do
      intercept[IllegalArgumentException](
        plain.validateNativeChoices(caller.selected(input), Vector(token))
      ): Unit
      assertEquals(run(plain, caller.selected(input)), List(Step(true, Subject.init, List(!input))))

  test("choice acquisition reads all matching branches without invoking guards effects or end"):
    var guards = 0
    object caller:
      private val actor = Actor("caller")
      val selected = action("selected", actor)
      val plain = action("plain", actor)
      val unknown = action("unknown", actor)
    object Subject extends Machine[Boolean, Boolean]:
      var ready = false
      override def end: Boolean = throw AssertionError("end evaluated")
      def allowed: Boolean =
        guards += 1
        !ready
      object effects:
        val first = choice
        val second = choice
        val third = choice
        val fourth = choice
        def off(): Unit =
          choose(
            first -> { throw AssertionError("off arm evaluated") },
            second -> { throw AssertionError("off arm evaluated") }
          )
        def on(): Unit =
          choose(
            third -> { throw AssertionError("on arm evaluated") },
            fourth -> { throw AssertionError("on arm evaluated") }
          )
        def plain(): Unit = throw AssertionError("plain effect evaluated")
        def unknown(): Unit = expectauthor.OpaqueChoiceHelper.run()
      object transitions extends Transitions:
        on(caller.selected):
          if allowed then effects.off()
          else effects.on()
        on(caller.plain):
          effects.plain()
        on(caller.unknown):
          effects.unknown()
    Subject.validateNativeChoices(
      caller.selected(),
      Vector(Subject.effects.first, Subject.effects.third)
    )
    assertEquals(guards, 0)
    val unknown = intercept[IllegalArgumentException]:
      Subject.validateNativeChoices(caller.unknown(), Vector(Subject.effects.first))
    assert(unknown.getMessage.contains("no retained native choice program"), unknown.getMessage)
    assert(unknown.getMessage.contains("available, static choice inventory"), unknown.getMessage)
    assertEquals(expectauthor.OpaqueChoiceHelper.calls, 0)
    assertEquals(Subject.ready, false)
    for token <- Vector(choice, Subject.effects.first) do
      intercept[IllegalArgumentException](
        Subject.validateNativeChoices(caller.plain(), Vector(token))
      ): Unit
    intercept[IllegalArgumentException](
      Subject.validateNativeChoices(caller.selected(), Vector(choice))
    ): Unit
    assertEquals(guards, 0)
