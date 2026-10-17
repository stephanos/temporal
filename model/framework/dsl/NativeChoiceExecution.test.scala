package framework.dsl

class NativeChoiceExecutionSuite extends munit.FunSuite:
  private def run(
      machine: RecordMachine[Boolean, Boolean, Boolean],
      cls: Class
  ): List[Step[Boolean, Boolean, Boolean]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)(false, cls)

  test(
    "input programs receive actual records and scalar inputs through rule and derivation adapters"
  ):
    final case class Request(flag: Boolean)
    val supplied = Request(true)
    val seen = scala.collection.mutable.ArrayBuffer.empty[Request]
    object caller:
      val record = action("record", Actor("caller")).inputs[Request]
      val scalar = action("scalar", Actor("caller")).input[Boolean](name = "flag")
    val recordFirst = choice
    val recordSecond = choice
    val scalarFirst = choice
    val scalarSecond = choice
    val recordProgram = choose[Boolean, Request, Boolean, Boolean](
      recordFirst -> ((s: Boolean, in: Request) =>
        seen += in
        List(Step(in.flag, !s, List(in.flag)))
      ),
      recordSecond -> ((_: Boolean, in: Request) =>
        seen += in; Nil
      )
    )
    val scalarProgram = choose[Boolean, Boolean, Boolean, Boolean](
      scalarFirst -> ((s: Boolean, in: Boolean) => List(Step(in, !s, List(in)))),
      scalarSecond -> ((_: Boolean, _: Boolean) => Nil)
    )
    object Subject extends RecordMachine[Boolean, Boolean, Boolean]:
      val init = false
      def end(state: Boolean): Boolean = state
      object rules extends Rules:
        on(caller.record):
          always ~> recordProgram
        on(caller.scalar):
          always ~> scalarProgram
    val restricted = Subject.restrict(caller.record, caller.scalar)
    for machine <- Vector(Subject, restricted) do
      seen.clear()
      machine.validateNativeChoices(caller.record(supplied), Vector(recordFirst))
      machine.validateNativeChoices(caller.scalar(false), Vector(scalarSecond))
      assertEquals(seen.toList, Nil)
      assertEquals(run(machine, caller.record(supplied)), List(Step(true, true, List(true))))
      assertEquals(seen.size, 2)
      assert(seen.forall(_ eq supplied))
      assertEquals(run(machine, caller.scalar(false)), List(Step(false, true, List(false))))
      assertEquals(run(machine, caller.scalar(true)), List(Step(true, true, List(true))))

  test("class filtering survives derivation and unrelated rebinds discard old choice inventory"):
    object caller:
      val selected = action("selected", Actor("caller")).input[Boolean](name = "flag")
    val offToken = choice
    val offOther = choice
    val onToken = choice
    val onOther = choice
    var offCalls = 0
    var onCalls = 0
    val offProgram = choose[Boolean, Boolean, Boolean](
      offToken -> ((s: Boolean) =>
        offCalls += 1; List(Step(false, !s, List(false)))
      ),
      offOther -> ((_: Boolean) => Nil)
    )
    val onProgram = choose[Boolean, Boolean, Boolean](
      onToken -> ((s: Boolean) =>
        onCalls += 1; List(Step(true, !s, List(true)))
      ),
      onOther -> ((_: Boolean) => Nil)
    )
    object Subject extends RecordMachine[Boolean, Boolean, Boolean]:
      val init = false
      def end(state: Boolean): Boolean = state
      object rules extends Rules:
        on(caller.selected(false)):
          always ~> offProgram
        on(caller.selected(true)):
          always ~> onProgram
    val restricted = Subject.restrict(caller.selected)
    for machine <- Vector(Subject, restricted) do
      offCalls = 0
      onCalls = 0
      assertEquals(run(machine, caller.selected(false)), List(Step(false, true, List(false))))
      assertEquals((offCalls, onCalls), (1, 0))
      assertEquals(run(machine, caller.selected(true)), List(Step(true, true, List(true))))
      assertEquals((offCalls, onCalls), (1, 1))
    val reboundToken = choice
    val reboundOther = choice
    val replacementProgram = choose[Boolean, Boolean, Boolean, Boolean](
      reboundToken -> ((s: Boolean, in: Boolean) => List(Step(!in, s, List(in)))),
      reboundOther -> ((_: Boolean, _: Boolean) => Nil)
    )
    val replacedProgram = Subject.rebind(caller.selected ~> replacementProgram)
    for input <- Vector(false, true) do
      replacedProgram.validateNativeChoices(caller.selected(input), Vector(reboundToken))
    assertEquals(run(replacedProgram, caller.selected(false)), List(Step(true, false, List(false))))
    assertEquals(run(replacedProgram, caller.selected(true)), List(Step(false, false, List(true))))
    val rebound = Subject.rebind(
      caller.selected ~> ((s: Boolean, in: Boolean) => List(Step(in, s, List(!in))))
    )
    for (input, token) <- Vector(false -> offToken, true -> onToken) do
      intercept[IllegalArgumentException](
        rebound.validateNativeChoices(caller.selected(input), Vector(token))
      ): Unit
    assertEquals(run(rebound, caller.selected(false)), List(Step(false, false, List(true))))
    assertEquals(run(rebound, caller.selected(true)), List(Step(true, false, List(false))))

  test(
    "selection ranges over every truthful program for its class and rejects matching opaque rules"
  ):
    object caller:
      val selected = action("guarded", Actor("caller"))
    val offToken = choice
    val offOther = choice
    val onToken = choice
    val onOther = choice
    var guards = 0
    val offProgram = choose[Boolean, Boolean, Nothing](
      offToken -> ((_: Boolean) => throw AssertionError("off arm evaluated")),
      offOther -> ((_: Boolean) => throw AssertionError("off arm evaluated"))
    )
    val onProgram = choose[Boolean, Boolean, Nothing](
      onToken -> ((_: Boolean) => throw AssertionError("on arm evaluated")),
      onOther -> ((_: Boolean) => throw AssertionError("on arm evaluated"))
    )
    object Subject extends RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules:
        on(caller.selected):
          where { s => guards += 1; !s } ~> offProgram
          where { s => guards += 1; s } ~> onProgram
    Subject.validateNativeChoices(caller.selected(), Vector(offToken, onToken))
    assertEquals(guards, 0)
    object OpaqueSubject extends RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules:
        on(caller.selected):
          where((s: Boolean) => !s) ~> offProgram
          where((s: Boolean) => s) ~>
            ((_: Boolean) => throw AssertionError("opaque arm evaluated"))
    intercept[IllegalArgumentException](
      OpaqueSubject.validateNativeChoices(caller.selected(), Vector(offToken))
    ): Unit
