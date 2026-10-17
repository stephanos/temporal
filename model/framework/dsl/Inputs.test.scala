package framework.dsl

import framework.dsl.Finite.given
// Input records, inputs supplied by name and bounded counters, run as Scala. The IR generator's
// fixtures (irgen/testdata/lifts/Inputs.scala) prove the IR; these pin the framework's own values.

class Inputs extends munit.FunSuite:
  enum Timeout:
    case unset, expires

  enum Answer:
    case completed
    case failed(retryable: Boolean)

  final case class Counted(attempts: UpTo[2], timeout: Timeout)

  final case class Start(
      scheduleToClose: Timeout = Timeout.unset,
      scheduleToStart: Timeout = Timeout.unset,
      startToClose: Timeout = Timeout.unset
  )
  final case class Four(
      scheduleToClose: Timeout = Timeout.unset,
      scheduleToStart: Timeout = Timeout.unset,
      startToClose: Timeout = Timeout.unset,
      enabled: Boolean = false
  )
  final case class Five(
      scheduleToClose: Timeout = Timeout.unset,
      scheduleToStart: Timeout = Timeout.unset,
      startToClose: Timeout = Timeout.unset,
      enabled: Boolean = false,
      answer: Answer = Answer.completed
  )
  final case class Six(
      scheduleToClose: Timeout = Timeout.unset,
      scheduleToStart: Timeout = Timeout.unset,
      startToClose: Timeout = Timeout.unset,
      enabled: Boolean = false,
      answer: Answer = Answer.completed,
      count: UpTo[2] = UpTo(0)
  )
  private val start = action(Actor("p")).inputs[Start]
  private val respond =
    action(Actor("p")).input[Answer](name = "answer", default = Answer.completed)

  test("UpTo[2] lists 0, 1 and 2, and a record varies its last field fastest"):
    assertEquals(Finite[UpTo[2]].values.toList, List(0, 1, 2))
    assertEquals(
      Finite[Counted].values.toList.map(c => (c.attempts: Int, c.timeout)),
      for a <- List(0, 1, 2); t <- List(Timeout.unset, Timeout.expires) yield (a, t)
    )
    interceptMessage[IllegalArgumentException]("requirement failed: 3 is outside 0..2")(
      UpTo[2](3): Unit
    )

  test("a named call matches its record value, each omitted field at its declared default"):
    assertEquals(
      start(scheduleToStart = Timeout.expires),
      start(Start(Timeout.unset, Timeout.expires, Timeout.unset))
    )
    assertEquals(
      start(startToClose = Timeout.expires, scheduleToClose = Timeout.expires),
      start(Start(Timeout.expires, Timeout.unset, Timeout.expires))
    )
    assertEquals(respond(Answer.failed(true)).values, List(Answer.failed(true)))
    for a <- Finite[Timeout].values; b <- Finite[Timeout].values; c <- Finite[Timeout].values do
      assertEquals(
        start(scheduleToClose = a, scheduleToStart = b, startToClose = c),
        start(Start(a, b, c))
      )
      assertEquals(start(scheduleToStart = b).values, List(Timeout.unset, b, Timeout.unset))
      assertEquals(start(startToClose = c, scheduleToClose = a).values, List(a, Timeout.unset, c))
    for value <- Finite[Answer].values do assertEquals(respond(value).values, List(value))

  test("a fourth field keeps its type and order in record-value and named calls"):
    val four = action(Actor("p")).inputs[Four]
    assertEquals(
      four(startToClose = Timeout.expires, enabled = true),
      four(Four(Timeout.unset, Timeout.unset, Timeout.expires, true))
    )
    assertEquals(four().values, List(Timeout.unset, Timeout.unset, Timeout.unset, false))
    for
      a <- Finite[Timeout].values
      b <- Finite[Timeout].values
      c <- Finite[Timeout].values
      flag <- Finite[Boolean].values
    do
      assertEquals(
        four(scheduleToClose = a, scheduleToStart = b, startToClose = c, enabled = flag),
        four(Four(a, b, c, flag))
      )
      assertEquals(
        four(startToClose = c, enabled = flag),
        four(Four(Timeout.unset, Timeout.unset, c, flag))
      )

  test(
    "a fifth typed input reaches the rule in position, with named defaults and invalid calls refused"
  ):
    object caller:
      val five = action(Actor("p")).inputs[Five]
    val five = caller.five
    val supplied =
      five(enabled = true, answer = Answer.completed, startToClose = Timeout.expires)
    assertEquals(
      supplied,
      five(Five(Timeout.unset, Timeout.unset, Timeout.expires, true, Answer.completed))
    )

    assertEquals(
      five().values,
      List(Timeout.unset, Timeout.unset, Timeout.unset, false, Answer.completed)
    )
    object FiveInputs extends RecordMachine[Counted, outcomes.Outcome, Answer]:
      val init = Counted(UpTo(0), Timeout.unset)
      def end(s: Counted): Boolean = s.attempts == 1
      object rules extends Rules:
        on(caller.five):
          always ~> ((s: Counted, in: Five) =>
            enter(
              s.copy(
                attempts =
                  UpTo(if in.enabled && in.scheduleToClose == in.scheduleToStart then 1 else 0),
                timeout = in.startToClose
              ),
              in.answer
            )
          )
    val binding = FiveInputs.bindings.head
    val after = effectOf(binding.decl, binding.function)(FiveInputs.init, supplied.values).head
    assertEquals(after.state, Counted(UpTo(1), Timeout.expires))
    assertEquals(after.facts, List(Answer.completed))
    for state <- Finite[Counted].values; values <- classesOf(five.decl) do
      val expected = values match
        case List(a: Timeout, b: Timeout, c: Timeout, flag: Boolean, result: Answer) =>
          assertEquals(
            five(
              scheduleToClose = a,
              scheduleToStart = b,
              startToClose = c,
              enabled = flag,
              answer = result
            ).values,
            List(a, b, c, flag, result)
          )
          enter(
            state.copy(attempts = UpTo(if flag && a == b then 1 else 0), timeout = c),
            result
          )
        case other => fail(s"unexpected five-input class $other")
      assertEquals(effectOf(binding.decl, binding.function)(state, values), expected)
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: UpTo[2])
      val a = action(Actor("p")).inputs[Five]
      a(a = false, b = false, c = false, d = false, e = true)
    """).exists(e => e.message.contains("Required:") && e.message.contains("UpTo"))
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: UpTo[2])
      val a = action(Actor("p")).inputs[Five]
      a(a = false, b = false, c = false, d = false)
    """).exists(_.message.contains("parameter e"))
    )
    assert(compiletime.testing.typeChecks("""
      import framework.dsl.*
      final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: UpTo[2])
      val a = action(Actor("p")).inputs[Five]
      a(Five(false, false, false, false, UpTo[2](1)))
      a(a = false, b = false, c = false, d = false, e = UpTo[2](1))
    """))

  test("a sixth input reaches execution with its declared value and named default"):
    assert(compiletime.testing.typeChecks("""
      import framework.dsl.*
      final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)
      val a = action(Actor("p")).inputs[Six]
      a(Six(false, false, false, false, false, true))
      a(a = false, b = false, c = false, d = false, e = false, f = true)
    """))
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)
      val a = action(Actor("p")).inputs[Six]
      a(a = false, b = false, c = false, d = false, e = false, f = 1)
    """).exists(_.message.contains("Required: Boolean"))
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)
      val a = action(Actor("p")).inputs[Six]
      a(a = false, b = false, c = false, d = false, e = false)
    """).exists(_.message.contains("parameter f"))
    )
    object caller:
      val six = action(Actor("p")).inputs[Six]
    val six = caller.six
    val supplied = six(enabled = true, count = UpTo[2](2), startToClose = Timeout.expires)
    assertEquals(
      supplied,
      six(Six(Timeout.unset, Timeout.unset, Timeout.expires, true, Answer.completed, UpTo[2](2)))
    )
    assertEquals(
      supplied.values,
      List(Timeout.unset, Timeout.unset, Timeout.expires, true, Answer.completed, UpTo[2](2))
    )
    assertEquals(
      six().values,
      List(Timeout.unset, Timeout.unset, Timeout.unset, false, Answer.completed, UpTo[2](0))
    )
    val coreStep = (
        s: Counted,
        a: Timeout,
        b: Timeout,
        c: Timeout,
        flag: Boolean,
        result: Answer,
        attempts: UpTo[2]
    ) =>
      enter(
        s.copy(attempts = if flag && a == b then attempts else UpTo(0), timeout = c),
        result
      )
    val step = (s: Counted, in: Six) =>
      coreStep(
        s,
        in.scheduleToClose,
        in.scheduleToStart,
        in.startToClose,
        in.enabled,
        in.answer,
        in.count
      )
    val execute = effectOf[Counted, outcomes.Outcome, Answer](six.decl, step)
    assertEquals(
      execute(Counted(UpTo(0), Timeout.unset), six().values).head.state,
      Counted(UpTo(0), Timeout.unset)
    )
    assertEquals(
      execute(Counted(UpTo(0), Timeout.unset), supplied.values).head.state,
      Counted(UpTo(2), Timeout.expires)
    )
    val direct = six ~> step
    assertEquals(
      effectOf[Counted, outcomes.Outcome, Answer](direct.decl, direct.function)(
        Counted(UpTo(0), Timeout.unset),
        supplied.values
      ).head.state,
      Counted(UpTo(2), Timeout.expires)
    )
    object SixInputs extends RecordMachine[Counted, outcomes.Outcome, Answer]:
      val init = Counted(UpTo(0), Timeout.unset)
      def end(s: Counted): Boolean = s.attempts == 2
      object rules extends Rules:
        on(caller.six):
          always ~> step
    val binding = SixInputs.bindings.head
    assertEquals(
      effectOf[Counted, outcomes.Outcome, Answer](binding.decl, binding.function)(
        SixInputs.init,
        supplied.values
      ).head.state,
      Counted(UpTo(2), Timeout.expires)
    )
    val ruled = Bound.Ruled[Counted, outcomes.Outcome, Answer](
      Vector(
        Rule(1, "always", "six", six.decl, None, _ => true, execute)
      )
    )
    val lowered =
      effectOf[Counted, outcomes.Outcome, Answer](six.decl, stepFunction(six.decl, ruled))
    assertEquals(
      lowered(Counted(UpTo(0), Timeout.unset), supplied.values).head.facts,
      List(Answer.completed)
    )
    for state <- Finite[Counted].values; values <- classesOf(six.decl) do
      val expected = values match
        case List(
              a: Timeout,
              b: Timeout,
              c: Timeout,
              flag: Boolean,
              result: Answer,
              attempts: Int
            ) =>
          val bounded = UpTo[2](attempts)
          assertEquals(
            six(
              scheduleToClose = a,
              scheduleToStart = b,
              startToClose = c,
              enabled = flag,
              answer = result,
              count = bounded
            ).values,
            List(a, b, c, flag, result, bounded)
          )
          coreStep(state, a, b, c, flag, result, bounded)
        case other => fail(s"unexpected six-input class $other")
      assertEquals(execute(state, values), expected)
      assertEquals(
        effectOf[Counted, outcomes.Outcome, Answer](direct.decl, direct.function)(state, values),
        expected
      )
      assertEquals(effectOf(binding.decl, binding.function)(state, values), expected)
      assertEquals(lowered(state, values), expected)

  test("a record beyond six inputs executes and lowers without an arity cap"):
    final case class Seven(
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean,
        f: Boolean,
        g: Boolean
    )
    val seven = action(Actor("p")).inputs[Seven]
    val supplied = seven(Seven(false, false, false, false, false, false, true))
    assertEquals(supplied.values, List(false, false, false, false, false, false, true))
    val step = (s: Counted, in: Seven) =>
      enter(s.copy(attempts = UpTo(if in.g then 2 else 0)), Answer.completed)
    val execute = effectOf[Counted, outcomes.Outcome, Answer](seven.decl, step)
    val expected = enter(Counted(UpTo(2), Timeout.unset), Answer.completed)
    assertEquals(execute(Counted(UpTo(0), Timeout.unset), supplied.values), expected)
    val ruled = Bound.Ruled[Counted, outcomes.Outcome, Answer](
      Vector(Rule(1, "always", "seven", seven.decl, None, _ => true, execute))
    )
    val lowered = effectOf[Counted, outcomes.Outcome, Answer](
      seven.decl,
      stepFunction(seven.decl, ruled)
    )
    assertEquals(lowered(Counted(UpTo(0), Timeout.unset), supplied.values), expected)
    val disabled = effectOf[Counted, outcomes.Outcome, Answer](
      seven.decl,
      stepFunction[Counted, outcomes.Outcome, Answer](seven.decl, Bound.Disabled())
    )
    assertEquals(disabled(Counted(UpTo(0), Timeout.unset), supplied.values), Nil)

  test("repeated protobuf messages are typed by the selected repeated message field"):
    assert(compiletime.testing.typeChecks("""
      import framework.realize.*
      import com.google.protobuf.struct.{ListValue, Value}
      Proto[ListValue](ProtoField.typed(Field(_.values), ProtoValue.messages(Proto[Value]())))
    """))
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.realize.*
      import com.google.protobuf.struct.ListValue
      Proto[ListValue](ProtoField.typed(Field(_.values), ProtoValue.messages(Proto[ListValue]())))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.realize.*
      import com.google.protobuf.struct.{Struct, Value}
      Proto[Struct](ProtoField.typed(Field(_.fields), ProtoValue.messages(Proto[Value]())))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.realize.*
      import com.google.protobuf.wrappers.BytesValue
      Proto[BytesValue](ProtoField.typed(Field(_.value), ProtoValue.messages(Proto[BytesValue]())))
    """).nonEmpty
    )
