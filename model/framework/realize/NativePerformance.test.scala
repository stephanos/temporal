package framework.realize

import framework.dsl.*

class NativePerformanceSuite extends munit.FunSuite:
  private val instruction = Instruction.AwaitLearned("ready")
  private val command = Command("perform", instruction)

  test("perform and direct Performance refuse modelOnly declarations"):
    val hidden = action("hidden", Actor("fault")).crash.modelOnly(because = "no native control")
    val ordinary = action("ordinary", Actor("caller"))
    for construct <- Vector[() => Any](
        () => Performance(hidden, command),
        () => perform(hidden -> instruction),
        () => Performance(ordinary, command).copy(step = hidden)
      )
    do
      val error = intercept[IllegalArgumentException](construct())
      assert(error.getMessage.contains("modelOnly"), error.getMessage)
      assert(error.getMessage.contains("no native control"), error.getMessage)

  test("choosing retains exact ordered tokens and rejects empty or duplicate subsets"):
    val step = action("ordinary", Actor("caller"))
    val first = choice
    val second = choice
    val performance = Performance(step, command)
    assertEquals(performance.selectedChoices, None)
    val selected = performance.choosing(second, first)
    assert(selected.selectedChoices.get(0) eq second)
    assert(selected.selectedChoices.get(1) eq first)
    assertEquals(performance.selectedChoices, None)
    for selection <- Vector(Vector.empty[Choice], Vector(first, first)) do
      intercept[IllegalArgumentException](performance.choosing(selection*)): Unit
      intercept[IllegalArgumentException](Performance(step, command, Some(selection))): Unit
      intercept[IllegalArgumentException](performance.copy(selectedChoices = Some(selection))): Unit

  test("completed realization rechecks an action made modelOnly after Performance construction"):
    val hidden = action("later-hidden", Actor("fault")).crash
    val performance = Performance(hidden, command)
    val _ = hidden.modelOnly(because = "control withdrawn")
    object Subject extends Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules
    val error = intercept[IllegalArgumentException](realization(Subject, performance))
    assert(error.getMessage.contains("control withdrawn"), error.getMessage)

  private def realization(machine: Machine[?, ?, ?], performance: Performance): Realization =
    Realization(
      name = "native-test",
      machine = machine,
      producer = "test",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation("", "", "", "", 1, 1, 1, 1, 1, 1),
      scripts =
        Vector(script("controller", Activation.Controller)(Item(performs = Vector(performance))))
    )

  test("completed realization checks exact choice membership without evaluating machine callbacks"):
    object declaredTokens:
      val sameName = choice
    object foreignTokens:
      val sameName = choice
    val declared = declaredTokens.sameName
    val other = choice
    val outsider = foreignTokens.sameName
    val performed = action("performed", Actor("caller"))
    val foreign = action("performed", Actor("caller"))
    val program = choose[Boolean, Boolean, Nothing](
      declared -> ((_: Boolean) => throw AssertionError("choice arm evaluated")),
      other -> ((_: Boolean) => throw AssertionError("choice arm evaluated"))
    )
    class Subject extends Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Bindings(performed ~> program)

    val machine = new Subject
    val selected = Performance(performed, command).choosing(declared)
    assert(realization(machine, selected).scripts.head.items.head.performs.head eq selected)
    var exposedDependencies = 0
    final class NativeKit(val machine: Machine[?, ?, ?], performance: Performance)
        extends ConstructedRealization:
      protected def declaredPerformances: Vector[Performance] = Vector(performance)
      val runs: Seq[AnyRef] = Seq.empty
      protected def declaredDependencies: ConstructedRealization.Dependencies =
        exposedDependencies += 1
        ConstructedRealization.Dependencies.empty
      def toIR(
          context: ConstructedRealization.Context
      ): io.temporal.server.api.umpire.v1.Realization =
        throw AssertionError("kit conversion evaluated")
    val kit = new NativeKit(machine, selected)
    assert(kit.nativePerformances.head eq selected)
    assertEquals(kit.dependencies, ConstructedRealization.Dependencies.empty)
    assertEquals(exposedDependencies, 1)
    for invalid <- Vector(
        selected.choosing(outsider),
        selected.copy(step = foreign)
      )
    do
      val error = intercept[IllegalArgumentException](realization(machine, invalid))
      assert(error.getMessage.contains("NativePerformance.test.scala"), error.getMessage)
      val invalidKit = new NativeKit(machine, invalid)
      intercept[IllegalArgumentException](invalidKit.nativePerformances): Unit
      intercept[IllegalArgumentException](invalidKit.dependencies): Unit
      assertEquals(exposedDependencies, 1)
    object WrongMachine extends Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules
    intercept[IllegalArgumentException](realization(WrongMachine, selected)): Unit

  test("choosing membership belongs to the performed class rather than its whole action"):
    val first = choice
    val second = choice
    val otherFirst = choice
    val otherSecond = choice
    object caller:
      val performed = action("by-input", Actor("caller")).input[Boolean](name = "flag")
    val offProgram = choose[Boolean, Boolean, Nothing](
      first -> ((_: Boolean) => throw AssertionError("off arm evaluated")),
      second -> ((_: Boolean) => throw AssertionError("off arm evaluated"))
    )
    val onProgram = choose[Boolean, Boolean, Nothing](
      otherFirst -> ((_: Boolean) => throw AssertionError("on arm evaluated")),
      otherSecond -> ((_: Boolean) => throw AssertionError("on arm evaluated"))
    )
    object Subject extends Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules extends Rules:
        on(caller.performed(false)):
          always ~> offProgram
        on(caller.performed(true)):
          always ~> onProgram
    val selected = Performance(caller.performed(false), command).choosing(first)
    assert(realization(Subject, selected).scripts.nonEmpty)
    intercept[IllegalArgumentException](
      realization(Subject, selected.copy(step = caller.performed(true)))
    ): Unit

  test("explicit choosing refuses opaque callbacks while omitted selection accepts them"):
    val performed = action("opaque", Actor("caller"))
    val token = choice
    object Subject extends Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init evaluated")
      def end(state: Boolean): Boolean = throw AssertionError("end evaluated")
      object rules
          extends Bindings(
            performed ~> ((_: Boolean) => throw AssertionError("opaque callback evaluated"))
          )
    val performance = Performance(performed, command)
    assertEquals(realization(Subject, performance).machine, Subject)
    val error = intercept[IllegalArgumentException](
      realization(Subject, performance.choosing(token))
    )
    assert(error.getMessage.contains("opaque"), error.getMessage)
