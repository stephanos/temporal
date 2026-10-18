package framework.dsl

import framework.checker.Effects.*

object InputExecutionFixture:

  object Trace:
    var constructors = 0
    var defaults = 0
    var defaultConstructors = 0
    val inputs = scala.collection.mutable.ArrayBuffer.empty[AnyRef]
    def default: Boolean =
      defaults += 1
      false
  final case class Request(
      a: Boolean = Trace.default,
      b: Boolean = false,
      c: Boolean = false,
      d: Boolean = false,
      e: Boolean = false,
      f: Boolean = false,
      g: Boolean = true
  ):
    Trace.constructors += 1
  final case class Other(flag: Boolean = true)
  final case class Defaulted(flag: Boolean = Trace.default):
    Trace.defaultConstructors += 1
  val defaultValue = Defaulted(false)
  val first = Request(false, true, false, false, false, false, true)
  val second = Request(true, false, false, false, false, false, false)
  val selected = Finite.of(first, second)

  object caller extends Actor:
    val start = action.inputs[Request](using
      summon[scala.deriving.Mirror.ProductOf[Request]],
      selected,
      summon[EmptyTuple =:= EmptyTuple]
    )
    val faceted = action.inputs[Request](using
      summon[scala.deriving.Mirror.ProductOf[Request]],
      selected,
      summon[EmptyTuple =:= EmptyTuple]
    )
    val product = action.input[Other](name = "settings", default = Other())
    val pair = action.input[(Boolean, Boolean)](name = "pair", default = (true, false))
    val single = action.input[Tuple1[Boolean]](name = "single", default = Tuple1(true))
    val alternate = action.inputs[Other]
    val scalar = action.input[Boolean](name = "result", default = false)
    val empty = action
    val defaulted = action.inputs[Defaulted](using
      summon[scala.deriving.Mirror.ProductOf[Defaulted]],
      Finite.of(defaultValue),
      summon[EmptyTuple =:= EmptyTuple]
    )
  def effect(state: Boolean, in: Request): List[Step[Boolean, Boolean, Boolean]] =
    Trace.inputs += in
    List(Step(in.a, state || in.g, List(in.b, in.a)))
  def replacement(state: Boolean, in: Request): List[Step[Boolean, Boolean, Boolean]] =
    Trace.inputs += in
    List(Step(in.g, !state, List(in.a)))

  given Ok[Boolean] = Ok(true)
  trait Applying extends Facet[Boolean, Boolean]:
    var ready: Boolean

    object applying:

      object effects:
        def receive(in: Request): Unit =
          Trace.inputs += in
          ready = ready || in.g
          record(in.b)
          record(in.a)

      object transitions extends Transitions:
        on(caller.faceted): in =>
          effects.receive(in)

  object Native extends Machine[Boolean, Boolean], Applying:
    var ready = false
    override def end: Boolean = ready

    object effects:
      def explicit(): Unit = record(true)
      def product(in: Other): Unit = record(in.flag)
      def pair(in: (Boolean, Boolean)): Unit = record(in._2)
      def single(in: Tuple1[Boolean]): Unit = record(in._1)
      def toggle(): Unit =
        ready = !ready
        record(false)
      def replacement(in: Request): Unit =
        Trace.inputs += in
        ready = !ready
        record(in.a)

    object transitions extends Transitions:
      on(caller.start): in =>
        if !ready then applying.effects.receive(in)
      on(caller.start(first)):
        if ready then effects.explicit()
      on(caller.product): in =>
        effects.product(in)
      on(caller.pair): in =>
        effects.pair(in)
      on(caller.single): in =>
        effects.single(in)
      on(caller.alternate(Other()), caller.scalar(true), caller.empty):
        effects.toggle()

  object Faceted extends Machine[Boolean, Boolean], Applying:
    var ready = false

    object transitions extends Facets(applying.transitions)

  object Rebound
      extends Derived(Native.rebind(on(caller.start): in =>
        if !Native.ready || (in eq first) then Native.effects.replacement(in)))

  object DefaultSource extends Machine[Boolean, Boolean]:
    var ready = false

    object effects:
      def receive(): Unit = record(true)

    object transitions extends Transitions:
      on(caller.defaulted()):
        effects.receive()

class InputExecution extends munit.FunSuite:
  private def runNative(machine: MachineDeclaration[State, Boolean, Boolean], decl: ActionDecl) =
    val binding = machine.bindings.find(_.decl eq decl).get
    val execute = effectOf[State, Boolean, Boolean](decl, binding.function)
    (ready: Boolean, cls: Class) =>
      execute(machine.init.copy(ready = ready), cls).map: step =>
        step.state.selectDynamic("ready") match
          case value: Boolean => Step(step.outcome, value, step.facts)
          case other          => fail(s"unexpected ready value $other")

  test("a scalar EmptyTuple remains one required value and cannot reopen declarations"):
    final case class Request(flag: Boolean = false)
    val unit = action(Actor("caller")).input[EmptyTuple](name = "unit")
    assertEquals(unit(EmptyTuple).values, List(EmptyTuple))
    assert(intercept[IllegalArgumentException](unit()).getMessage.contains("input unit"))
    val callback = (_: Boolean, in: EmptyTuple) => List(Step(true, false, List(in)))
    assertEquals(
      effectOf[Boolean, Boolean, EmptyTuple](unit.decl, callback)(false, unit(EmptyTuple)),
      List(Step(true, false, List(EmptyTuple)))
    )
    intercept[IllegalArgumentException](unit.input[Boolean](name = "other")): Unit
    intercept[IllegalArgumentException](unit.inputs[Request]): Unit
    val empty = action(Actor("caller"))
    assertEquals(
      effectOf[Boolean, Boolean, Nothing](empty.decl, (s: Boolean) => List(Step(true, s)))(
        false,
        Nil
      ),
      List(Step(true, false))
    )

  test("native effects receive one actual record beyond six fields"):
    final case class Request(
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean,
        f: Boolean,
        g: Boolean
    )
    val request = Request(true, false, false, false, false, false, true)
    val start = action(Actor("caller")).inputs[Request]
    val callback = (state: Boolean, in: Request) =>
      List(Step[Boolean, Boolean, Boolean](in.a, state || in.g, List(in.b)))
    val run = effectOf[Boolean, Boolean, Boolean](start.decl, callback)
    assertEquals(run(false, start(request).values), List(Step(true, true, List(false))))

  test("native record and scalar input blocks bind with their declared types"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      final case class Request(flag: Boolean = false, other: Boolean = true)
      object caller extends Actor:
        val start = action.inputs[Request]
        val finish = action.input[Boolean](name = "result", default = false)
      object Native extends Machine[Boolean, Nothing]:
        var ready = false
        object effects:
          def start(in: Request): Unit = ready = in.flag
          def finish(in: Boolean): Unit = ready = in
        object transitions extends Transitions:
          on(caller.start): in =>
            effects.start(in)
          on(caller.finish): in =>
            effects.finish(in)
    """),
      Nil
    )

  test(
    "supplied and enumerated records keep identity through callbacks, copies, aliases and facets"
  ):
    import InputExecutionFixture.*
    val alias: Action[Request] = caller.start
    val supplied = alias(first)
    val native = Native.bindings.find(_.decl eq caller.start.decl).get
    val facet = Faceted.bindings.find(_.decl eq caller.faceted.decl).get
    val before = Trace.constructors -> Trace.defaults
    val cases = inputClassesOf(caller.start.decl)
    assertEquals(
      cases.map(_.values),
      List(
        List(false, true, false, false, false, false, true),
        List(true, false, false, false, false, false, false)
      )
    )
    val run = runNative(Native, native.decl)
    Trace.inputs.clear()
    for cls <- List(supplied, supplied.copy(), cases.head) do
      assertEquals(run(false, cls), List(Step(true, true, List(true, false))))
    assert(Trace.inputs.forall(_ eq first))
    assertEquals(run(false, cases(1)), List(Step(true, false, List(false, true))))
    assert(Trace.inputs.last eq second)
    val facetClass = inputClassesOf(caller.faceted.decl).head
    assertEquals(
      runNative(Faceted, facet.decl)(false, facetClass),
      List(Step(true, true, List(true, false)))
    )
    assert(Trace.inputs.last eq first)
    assertEquals(Trace.constructors -> Trace.defaults, before)
    val changed = supplied.copy(values = cases(1).values)
    assert(changed.supplied.isEmpty)
    assertEquals(run(false, changed), List(Step(true, false, List(false, true))))
    val recovered = Trace.inputs.last
    assertEquals(run(false, changed), List(Step(true, false, List(false, true))))
    assert(Trace.inputs.last eq recovered)
    assertEquals(Trace.constructors -> Trace.defaults, (before._1 + 1, before._2))

  test("class restrictions and native replacements retain explicit step pins"):
    import InputExecutionFixture.*
    def run(machine: MachineDeclaration[State, Boolean, Boolean], state: Boolean, cls: Class) =
      runNative(machine, cls.decl)(state, cls)
    assertEquals(run(Native, true, caller.start(first)), List(Step(true, true, List(true))))
    assertEquals(run(Native, true, caller.start(second)), Nil)
    assertEquals(run(Rebound, false, caller.start(first)), List(Step(true, true, List(false))))
    assertEquals(run(Rebound, true, caller.start(first)), List(Step(true, false, List(false))))
    assertEquals(run(Rebound, true, caller.start(second)), Nil)
    assertEquals(
      run(Native, false, caller.product(Other(false))),
      List(Step(true, false, List(false)))
    )
    assertEquals(
      run(Native, false, caller.pair((true, false))),
      List(Step(true, false, List(false)))
    )
    assertEquals(
      run(Native, false, caller.single(Tuple1(true))),
      List(Step(true, false, List(true)))
    )
    for cls <- List(caller.alternate(Other()), caller.scalar(true), caller.empty()) do
      assertEquals(run(Native, false, cls), List(Step(true, true, List(false))))

  test("generic native effects retain ordered alternatives and the actual record"):
    import InputExecutionFixture.{*, given}
    object Subject extends Machine[Boolean, Boolean]:
      var ready = false
      object effects:
        val one = choice
        val two = choice
        def receive(in: Request): Unit =
          choose(
            one -> { ready = ready || in.g; record(in.b); record(in.a) },
            two -> { ready = false; record(in.g) }
          )
      object transitions extends Transitions:
        on(caller.start): in =>
          effects.receive(in)
    assertEquals(
      runNative(Subject, caller.start.decl)(false, caller.start(first)),
      List(
        Step(true, true, List(true, false)),
        Step(true, false, List(true))
      )
    )
    val scalar = (state: Boolean, in: Tuple1[Boolean]) => List(Step(in._1, state, List(in._1)))
    assertEquals(
      effectOf[Boolean, Boolean, Boolean](caller.single.decl, scalar)(
        false,
        caller.single(Tuple1(false))
      ),
      List(Step(false, false, List(false)))
    )

  test("a native class rule constructs its target and defaults once before repeated execution"):
    import InputExecutionFixture.*
    val before = Trace.defaults -> Trace.defaultConstructors
    val binding = DefaultSource.bindings.head
    assertEquals(Trace.defaults -> Trace.defaultConstructors, (before._1 + 1, before._2 + 1))
    val run = runNative(DefaultSource, binding.decl)
    for state <- List(false, true); cls <- inputClassesOf(binding.decl) do
      assertEquals(run(state, cls), List(Step(true, state, List(true))))
    assertEquals(Trace.defaults -> Trace.defaultConstructors, (before._1 + 1, before._2 + 1))

  test("wrong native input records scalar shapes and flattened arities refuse with controls"):
    val errors = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        given Ok[Boolean] = Ok(true)
        final case class Request(flag: Boolean)
        final case class Other(flag: Boolean)
        object caller extends Actor:
          val start = action.inputs[Request]
        object Bad extends Machine[Boolean, Nothing]:
          var ready = false
          object effects:
            def accept(in: Other): Unit = ()
          object transitions extends Transitions:
            on(caller.start) { (in: Other) => effects.accept(in) }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        given Ok[Boolean] = Ok(true)
        final case class Request(a: Boolean, b: Boolean)
        object caller extends Actor:
          val start = action.inputs[Request]
        object Bad extends Machine[Boolean, Nothing]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start) { (_: Boolean, _: Boolean) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        given Ok[Boolean] = Ok(true)
        object caller extends Actor:
          val start = action.input[Tuple1[Boolean]](name = "value")
        object Bad extends Machine[Boolean, Nothing]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start) { (_: Boolean) => effects.accept() }
      """)
    )
    errors.foreach(refusal => assert(refusal.nonEmpty, refusal.toString))
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      object caller extends Actor:
        val start = action.input[Tuple1[Boolean]](name = "value")
        val pair = action.input[(Boolean, Boolean)](name = "pair")
      object Good extends Machine[Boolean, Nothing]:
        var ready = false
        object effects:
          def single(in: Tuple1[Boolean]): Unit = ready = in._1
          def pair(in: (Boolean, Boolean)): Unit = ready = in._2
        object transitions extends Transitions:
          on(caller.start): in =>
            effects.single(in)
          on(caller.pair): in =>
            effects.pair(in)
    """),
      Nil
    )
