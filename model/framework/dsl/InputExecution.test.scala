package framework.dsl

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

  object Native extends RecordMachine[Boolean, Boolean, Boolean]:
    val init = false
    def end(s: Boolean): Boolean = s

    object rules extends Rules:
      on(caller.start):
        where(s => !s) ~> effect
      on(caller.start(first)):
        where(identity) ~> ((s: Boolean) => List(Step(false, s, List(true))))
      on(caller.product):
        always ~> ((s: Boolean, in: Other) => List(Step(in.flag, s, List(in.flag))))
      on(caller.pair):
        always ~> ((s: Boolean, in: (Boolean, Boolean)) => List(Step(in._1, s, List(in._2))))
      on(caller.single):
        always ~> ((s: Boolean, in: Tuple1[Boolean]) => List(Step(in._1, s, List(in._1))))
      on(caller.alternate(Other()), caller.scalar(true), caller.empty):
        always ~> ((s: Boolean) => List(Step(true, !s, List(false))))
  trait Applying extends Facet[Boolean, Boolean, Boolean]:

    object applying:

      object rules extends Rules:
        on(caller.faceted):
          always ~> effect

  object Faceted extends RecordMachine[Boolean, Boolean, Boolean], Applying:
    val init = false
    def end(s: Boolean): Boolean = s

    object rules extends Facets(applying.rules)

  object Rebound extends Derived(Native.rebind(caller.start ~> replacement))

  object DefaultSource extends RecordMachine[Boolean, Boolean, Boolean]:
    val init = false
    def end(s: Boolean): Boolean = s

    object rules extends Rules:
      on(caller.defaulted()):
        always ~> ((s: Boolean) => List(Step(false, s, List(true))))

class InputExecution extends munit.FunSuite:
  test("a scalar EmptyTuple remains one required value and cannot reopen declarations"):
    final case class Request(flag: Boolean = false)
    val unit = action(Actor("caller")).input[EmptyTuple](name = "unit")
    assertEquals(unit(EmptyTuple).values, List(EmptyTuple))
    assert(intercept[IllegalArgumentException](unit()).getMessage.contains("input unit"))
    val callback = (_: Boolean, in: EmptyTuple) => List(Step(true, false, List(in)))
    val binding = unit ~> callback
    assertEquals(
      effectOf[Boolean, Boolean, EmptyTuple](binding.decl, binding.function)(
        false,
        unit(EmptyTuple)
      ),
      List(Step(true, false, List(EmptyTuple)))
    )
    intercept[IllegalArgumentException](unit.input[Boolean](name = "other")): Unit
    intercept[IllegalArgumentException](unit.inputs[Request]): Unit
    intercept[IllegalArgumentException](unit ~> ((s: Boolean) => List(Step(true, s)))): Unit
    val empty = action(Actor("caller"))
    val stateOnly = empty ~> ((s: Boolean) => List(Step(true, s)))
    assertEquals(
      effectOf[Boolean, Boolean, Nothing](stateOnly.decl, stateOnly.function)(false, Nil),
      List(Step(true, false))
    )
    intercept[IllegalArgumentException](empty ~> callback): Unit

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

  test("native record and scalar callbacks bind through rules and generic actions"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Request(flag: Boolean = false, other: Boolean = true)
      object caller extends Actor:
        val start = action.inputs[Request]
        val finish = action.input[Boolean](name = "result", default = false)
      def bind[In](a: Action[In], f: (Boolean, In) => List[Step[Boolean, Boolean, Nothing]]) = a ~> f
      val direct = bind(caller.start, (s: Boolean, in: Request) => List(Step(in.flag, s)))
      object Native extends RecordMachine[Boolean, Boolean, Nothing]:
        val init = false
        def end(s: Boolean) = s
        object rules extends Rules:
          on(caller.start):
            always ~> ((s: Boolean, in: Request) => List(Step(in.flag, s)))
          on(caller.finish):
            always ~> ((s: Boolean, in: Boolean) => List(Step(in, s)))
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
    val run = effectOf[Boolean, Boolean, Boolean](native.decl, native.function)
    Trace.inputs.clear()
    for cls <- List(supplied, supplied.copy(), cases.head) do
      assertEquals(run(false, cls), List(Step(false, true, List(true, false))))
    assert(Trace.inputs.forall(_ eq first))
    assertEquals(run(false, cases(1)), List(Step(true, false, List(false, true))))
    assert(Trace.inputs.last eq second)
    val facetClass = inputClassesOf(caller.faceted.decl).head
    assertEquals(
      effectOf[Boolean, Boolean, Boolean](facet.decl, facet.function)(false, facetClass),
      List(Step(false, true, List(true, false)))
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
    def run(machine: RecordMachine[Boolean, Boolean, Boolean], state: Boolean, cls: Class) =
      val binding = machine.bindings.find(_.decl eq cls.decl).get
      effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)(state, cls)
    assertEquals(run(Native, true, caller.start(first)), List(Step(false, true, List(true))))
    assertEquals(run(Native, true, caller.start(second)), Nil)
    assertEquals(run(Rebound, false, caller.start(first)), List(Step(true, true, List(false))))
    assertEquals(run(Rebound, true, caller.start(first)), List(Step(true, false, List(false))))
    assertEquals(run(Rebound, true, caller.start(second)), Nil)
    assertEquals(
      run(Native, false, caller.product(Other(false))),
      List(Step(false, false, List(false)))
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

  test("generic native bindings retain ordered alternatives and the actual scalar or record"):
    import InputExecutionFixture.*
    def bind[In](
        action: Action[In],
        callback: (Boolean, In) => List[Step[Boolean, Boolean, Boolean]]
    ) =
      action ~> callback
    val one = choice
    val two = choice
    val binding = bind(
      caller.start,
      (s: Boolean, in: Request) =>
        choose(one -> effect(s, in), two -> List(Step(true, false, List(in.g))))
    )
    val run = effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)
    assertEquals(
      run(false, caller.start(first)),
      List(
        Step(false, true, List(true, false)),
        Step(true, false, List(true))
      )
    )
    val scalar =
      bind(caller.single, (s: Boolean, in: Tuple1[Boolean]) => List(Step(in._1, s, List(in._1))))
    assertEquals(
      effectOf[Boolean, Boolean, Boolean](scalar.decl, scalar.function)(
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
    val bare = new RecordMachine[Boolean, Boolean, Boolean]:
      val init = false
      def end(s: Boolean): Boolean = s
      def rules: RuleBook[Boolean, Boolean, Boolean] = new Rules:
        import caller.defaulted
        on(defaulted()):
          always ~> ((s: Boolean) => List(Step(false, s, List(true))))
    val error = intercept[IllegalArgumentException](bare.bindings)
    assert(error.getMessage.contains("names its action bare"), error.getMessage)
    val run = effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)
    for state <- List(false, true); cls <- inputClassesOf(binding.decl) do
      assertEquals(run(state, cls), List(Step(false, state, List(true))))
    assertEquals(Trace.defaults -> Trace.defaultConstructors, (before._1 + 1, before._2 + 1))

  test("wrong native callback records, scalar shapes and flattened arities refuse with controls"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Request(flag: Boolean)
        final case class Other(flag: Boolean)
        object caller extends Actor:
          val start = action.inputs[Request]
        val bad = caller.start ~> ((s: Boolean, in: Other) => List(Step(in.flag, s)))
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Request(a: Boolean, b: Boolean)
        object caller extends Actor:
          val start = action.inputs[Request]
        object Bad extends RecordMachine[Boolean, Boolean, Nothing]:
          val init = false
          def end(s: Boolean) = s
          object rules extends Rules:
            on(caller.start):
              always ~> ((s: Boolean, a: Boolean, b: Boolean) => List(Step(a, s)))
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        object caller extends Actor:
          val start = action.input[Tuple1[Boolean]](name = "value")
        object Bad extends RecordMachine[Boolean, Boolean, Nothing]:
          val init = false
          def end(s: Boolean) = s
          object rules extends Rules:
            on(caller.start):
              always ~> ((s: Boolean, in: Boolean) => List(Step(in, s)))
      """)
    )
    refusals
      .zip(List("Request", "Firing", "RuleInput"))
      .foreach: (errors, reason) =>
        assert(
          errors.exists(e =>
            (e.lineContent.contains("~>") || e.lineContent.contains("in: Other")) &&
              e.message.contains(reason)
          ),
          errors.toString
        )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      object caller extends Actor:
        val start = action.input[Tuple1[Boolean]](name = "value")
        val pair = action.input[(Boolean, Boolean)](name = "pair")
      val good = caller.start ~> ((s: Boolean, in: Tuple1[Boolean]) => List(Step(in._1, s)))
      val tuple = caller.pair ~> ((s: Boolean, in: (Boolean, Boolean)) => List(Step(in._1, s)))
    """),
      Nil
    )
