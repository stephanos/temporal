package framework

import framework.realize.*

class InputDefaults extends munit.FunSuite:
  test("required tokens refuse omissions with their declaration name at initialization"):
    val requiredAnswer = input[Boolean]
    val send = action(Actor("sender")).input(requiredAnswer)
    val error = intercept[IllegalArgumentException](send(): Unit)
    assert(error.getMessage.contains("requiredAnswer"), error.getMessage)
    assert(error.getMessage.contains("no default"), error.getMessage)
    assertEquals(send(true).values, List(true))

  test("explicit non-first, false, zero and structured defaults keep token identity and bindings"):
    enum Mode:
      case first, second
    enum Answer:
      case done
      case failed(retryable: Boolean)
    final case class Settings(mode: Mode, enabled: Boolean)
    val preferred = input[Mode](default = Mode.second)
    val backup = input[Mode](default = Mode.first)
    val flag = input[Boolean](default = false)
    val count = input[UpTo[2]](default = UpTo(0))
    val answer = input[Answer](default = Answer.failed(true))
    val settings = input[Settings](default = Settings(Mode.second, false))
    val alias = preferred
    val start = action(Actor("sender"))
      .input(preferred)
      .input(backup)
      .input(flag)
      .input(count)
      .input(answer)
      .input(settings)
    val full = start(
      Mode.second,
      Mode.first,
      false,
      UpTo[2](0),
      Answer.failed(true),
      Settings(Mode.second, false)
    )
    assertEquals(start(), full)
    assertEquals(
      start(settings := Settings(Mode.first, true), alias := Mode.first).values,
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
    assertEquals(bound.decl.tokens, start.decl.tokens)
    val optional = input[Option[Mode]](default = Some(Mode.second))
    val absent = input[Option[Mode]](default = None)
    assertEquals(
      action(Actor("sender")).input(optional).input(absent)().values,
      List(Some(Mode.second), None)
    )
    intercept[IllegalArgumentException](
      start(preferred := Mode.first, alias := Mode.second): Unit
    ): Unit
    val foreign = input[Mode](default = Mode.first)
    intercept[IllegalArgumentException](start(foreign := Mode.first): Unit): Unit
    assertEquals(action(Actor("sender"))().values, Nil)
    assertEquals(timer().values, Nil)
    val named = action(Actor("sender")).input[Boolean]("requiredNamed")
    assert(intercept[IllegalArgumentException](named(): Unit).getMessage.contains("requiredNamed"))
    val mailbox = channel[Boolean](capacity = 1, order = Order.fifo, loss = Loss.lossy)
    for omitted <- List(() => mailbox.deliver(), () => mailbox.lose()) do
      val error = intercept[IllegalArgumentException](omitted(): Unit)
      assert(error.getMessage.contains("input message"), error.getMessage)
      assert(error.getMessage.contains("no default"), error.getMessage)

  test("machine, Scenario and realization script initialization refuse required omissions"):
    val requiredAnswer = input[Boolean]
    val defaultFlag = input[Boolean](default = false)
    val send = action(Actor("sender")).input(requiredAnswer).input(defaultFlag)
    class Base extends Machine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: State) = true
      object rules extends Bindings()
    val base = new Base
    class MissingMachine extends Base:
      val path = scenario.actions(send())
    class MissingScenario:
      val path = base.scenario.actions(send(defaultFlag := true))
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
