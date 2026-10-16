package framework.dsl

import framework.realize.*

class InputDefaults extends munit.FunSuite:
  test("class copies reconstruct another action's record while retaining metadata-copy inputs"):
    final case class A(flag: Boolean)
    final case class B(flag: Boolean)
    val source = action(Actor("sender")).inputs[A]
    val original = source(A(true))
    for declaration <- List(source.decl, source.results("answer").decl) do
      val copied = original.copy(decl = declaration)
      assert(copied.supplied.get eq original.supplied.get)
    val whole = action(Actor("sender")).inputs[B]
    val fields = action(Actor("sender")).inputs[B](
      inputField[B](_.flag).in(domains.oneOf(false, true))
    )
    for target <- List(whole, fields) do
      val copied = original.copy(decl = target.decl)
      val callback = (state: Boolean, in: B) => List(Step[Boolean, Boolean, Nothing](in.flag, state))
      val run = effectOf[Boolean, Boolean, Nothing](target.decl, callback)
      assertEquals(run(false, copied), List(Step(true, false)))

  test("required scalars refuse omissions with their declaration name at initialization"):
    val send = action(Actor("sender")).input[Boolean](name = "requiredAnswer")
    val error = intercept[IllegalArgumentException]:
      val _ = send()
      ()
    assert(error.getMessage.contains("requiredAnswer"), error.getMessage)
    assert(error.getMessage.contains("no default"), error.getMessage)
    assertEquals(send(true).values, List(true))

  test("explicit non-first, false, zero and structured defaults keep field order and bindings"):
    enum Mode:
      case first, second
    enum Answer:
      case done
      case failed(retryable: Boolean)
    final case class Settings(mode: Mode, enabled: Boolean)
    final case class Start(
        preferred: Mode = Mode.second,
        backup: Mode = Mode.first,
        flag: Boolean = false,
        count: UpTo[2] = UpTo(0),
        answer: Answer = Answer.failed(true),
        settings: Settings = Settings(Mode.second, false)
    )
    val start = action(Actor("sender")).inputs[Start]
    val full = start(
      Start(
        Mode.second,
        Mode.first,
        false,
        UpTo[2](0),
        Answer.failed(true),
        Settings(Mode.second, false)
      )
    )
    assertEquals(start(), full)
    assertEquals(
      full.values,
      List(
        Mode.second,
        Mode.first,
        false,
        UpTo[2](0),
        Answer.failed(true),
        Settings(Mode.second, false)
      )
    )
    assertEquals(
      start(settings = Settings(Mode.first, true), preferred = Mode.first).values,
      List(
        Mode.first,
        Mode.first,
        false,
        UpTo[2](0),
        Answer.failed(true),
        Settings(Mode.first, true)
      )
    )
    val bound = start.on(Entity(key = "id"))
    assertEquals(bound().values, full.values)
    assertEquals(
      bound.decl.inputs,
      List("preferred", "backup", "flag", "count", "answer", "settings")
    )
    assert(bound.decl.nativeInputs.get eq start.decl.nativeInputs.get)
    for modified <- List(start.creates(Entity(key = "newId")), start.results("answer")) do
      assertEquals(modified().values, full.values)
      assert(modified.decl.nativeInputs.get eq start.decl.nativeInputs.get)
    final case class Optional(
        optional: Option[Mode] = Some(Mode.second),
        absent: Option[Mode] = None
    )
    val optional = action(Actor("sender")).inputs[Optional]
    assertEquals(
      optional().values,
      List(Some(Mode.second), None)
    )
    assertEquals(optional.decl.nativeInputs.flatMap(_.defaultAt(1)), Some(None))
    assertEquals(action(Actor("sender"))().values, Nil)
    assertEquals(timer().values, Nil)
    val named = action(Actor("sender")).input[Boolean](name = "requiredNamed")
    val missing = intercept[IllegalArgumentException]:
      val _ = named()
      ()
    assert(missing.getMessage.contains("requiredNamed"))
    val mailbox = channel[Boolean](capacity = 1, order = Order.fifo, loss = Loss.lossy)
    for omitted <- List(() => mailbox.deliver(), () => mailbox.lose()) do
      val error = intercept[IllegalArgumentException](omitted(): Unit)
      assert(error.getMessage.contains("input message"), error.getMessage)
      assert(error.getMessage.contains("no default"), error.getMessage)

  test("record calls refuse unknown, duplicate and wrong-type fields with valid controls"):
    assert(compiletime.testing.typeChecks("""
      import framework.dsl.*
      final case class Start(preferred: Boolean = false)
      val start = action(Actor("sender")).inputs[Start]
      start(preferred = true)
    """))
    val unknown = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(preferred: Boolean = false)
      val start = action(Actor("sender")).inputs[Start]
      start(foreign = true)
    """)
    assert(unknown.exists(_.message.contains("unknown input foreign")), unknown.toString)
    val duplicate = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(preferred: Boolean = false)
      val start = action(Actor("sender")).inputs[Start]
      start(preferred = false, preferred = true)
    """)
    assert(duplicate.exists(_.message.contains("supplied twice")), duplicate.toString)
    val wrongType = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(preferred: Boolean = false)
      val start = action(Actor("sender")).inputs[Start]
      start(preferred = "wrong")
    """)
    assert(wrongType.exists(_.message.contains("Required: Boolean")), wrongType.toString)

  test("required record fields refuse omissions at compilation with a supplied control"):
    assert(compiletime.testing.typeChecks("""
      import framework.dsl.*
      final case class Send(requiredAnswer: Boolean, defaultFlag: Boolean = false)
      val send = action(Actor("sender")).inputs[Send]
      send(requiredAnswer = true)
      send(requiredAnswer = true, defaultFlag = true)
    """))
    val omitted = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Send(requiredAnswer: Boolean, defaultFlag: Boolean = false)
      val send = action(Actor("sender")).inputs[Send]
      send()
    """)
    val partial = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Send(requiredAnswer: Boolean, defaultFlag: Boolean = false)
      val send = action(Actor("sender")).inputs[Send]
      send(defaultFlag = true)
    """)
    for errors <- List(omitted, partial) do
      assert(errors.exists(_.message.contains("parameter requiredAnswer")), errors.toString)

  test("machine, Scenario and realization script initialization refuse required scalar omissions"):
    val send = action(Actor("sender")).input[Boolean](name = "requiredAnswer")
    class Base extends Machine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: State) = true
      object rules extends Bindings()
    val base = new Base
    class MissingMachine extends Base:
      val path = scenario.actions(send())
    class MissingScenario:
      val path = base.scenario.actions(send())
    class MissingRealization:
      val controller = script("controller", Activation.Controller)(
        onPath(send())(Instruction.AwaitLearned("answer"))
      )
    for initialize <- List[() => Unit](
        () =>
          val _ = new MissingMachine; ()
        ,
        () =>
          val _ = new MissingScenario; ()
        ,
        () =>
          val _ = new MissingRealization; ()
      )
    do
      val error = intercept[IllegalArgumentException](initialize())
      assert(error.getMessage.contains("input requiredAnswer"), error.getMessage)
