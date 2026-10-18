package framework.dsl

import framework.checker.*
import framework.checker.Effects.*
import BankControls.Leaky
import NativeControls.{assertRefuted, invariant}
import framework.dsl.Finite.given
import samples.Bank
import BankFixture.{model, Addr, Questions}
import BankFixture.model.{client, fits, totalSupply}

class BankTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[Bank.Outcome, Bank.Outcome]]

  private def run[S, O](
      machine: MachineDeclaration[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  // A Bank state by its one var, and the balances a state holds.
  private def bank(balances: Bank.Balances[Addr]): State = model.init.copy(balances = balances)
  private def held(state: State): Bank.Balances[Addr] = model.at(state)(model.balances)

  test("Bank realization retains its instantiated machine and native send"):
    import framework.realize.*
    import scala.compiletime.testing.typeChecks
    val selected = model.client.send(Bank.Send(Addr.king, Addr.peaches, Bank.Denom.banana, 1))
    val performance = Performance(selected, Command("send", Instruction.AwaitLearned("sent")))
    val root = Realization(
      name = "bank",
      machine = model,
      producer = "sample",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation("", "", "", "", 1, 1, 1, 1, 1, 1),
      scripts =
        Vector(script("controller", Activation.Controller)(Item(performs = Vector(performance))))
    )
    assert(root.machine eq model)
    assert(root.nativePerformances.head eq performance)
    assert(typeChecks("root.copy(machine = model)"))
    assert(!typeChecks("root.copy(machine = classOf[Bank.Model[Addr]])"))
    assert(!typeChecks(
      "root.copy(machine = (holder: Addr, supply: Int) => new Bank.Model[Addr](holder, supply))"
    ))

  test("Bank's four native predicates retain supply, refusal and exact-send requirements"):
    import Bank.*
    import NativeControls.invariant
    // a Bank state by its one var, and the balances a state holds
    def bank(balances: Balances[Addr]): State = model.init.copy(balances = balances)
    def held(state: State): Balances[Addr] = model.at(state)(model.balances)
    val initial = model.init
    val start = held(initial)
    val paid = bank(
      start.set(Addr.king, Denom.banana, 1).set(Addr.peaches, Denom.banana, 1)
    )
    val inflated = bank(start.set(Addr.peaches, Denom.banana, 1))
    val supplyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, inflated), false)
    )
    val leakyTotalSupplyInv = invariant(check(totalSupply(model.balances)))
    supplyCases.foreach: (after, expected) =>
      assertEquals(totalSupply(held(after.state)), expected)
      assertEquals(
        SampleClaims.judged(model, model, "invariants", "totalSupplyInv", after),
        Some(expected)
      )
      assertEquals(SampleClaims.holds(Leaky, leakyTotalSupplyInv, after.state), expected)
    val send = client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))
    List(
      (initial, Step(Outcome.accepted, paid), true),
      (initial, Step(Outcome.overflow, initial), true),
      (initial, Step(Outcome.overflow, paid), false)
    ).foreach: (before, after, expected) =>
      assertEquals(after.outcome == Outcome.accepted || after.state == before, expected)
      assertEquals(
        SampleClaims.judged(
          model,
          model,
          "postconditions",
          "failedSendChangesNothing",
          after,
          Some(before),
          Some(send)
        ),
        Some(expected)
      )
    // about the one exact send it names, and no other class of send
    def exact(after: Step[State, Outcome, Nothing], taken: Class) =
      SampleClaims.judged(
        model,
        Questions,
        "queries",
        "peachesGetsBananas",
        after,
        Some(initial),
        Some(taken)
      )
    List(
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, initial), false),
      (Step(Outcome.accepted, inflated), false),
      (Step(Outcome.accepted, bank(start.set(Addr.king, Denom.banana, 1))), false)
    ).foreach: (after, expected) =>
      assertEquals(
        held(after.state)(Addr.king, Denom.banana) == 1 &&
          held(after.state)(Addr.peaches, Denom.banana) == 1,
        expected
      )
      assertEquals(exact(after, send), Some(expected))
      assertEquals(exact(after, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 2))), None)

  test("Bank native balances do not invent a whole-state integer bound"):
    import Bank.*
    val large = held(model.init).set(Addr.king, Denom.banana, 7)
    assertEquals(large(Addr.king, Denom.banana), 7)
    assertEquals(large(Addr.king, Denom.burger), 2)

  test("Bank absent keys read zero and zero writes remove denominations then accounts"):
    import Bank.*
    val empty = Balances[Addr](Map.empty)
    assertEquals(empty(Addr.mario, Denom.banana), 0)
    val burgers = Balances[Addr](Map(Addr.king -> Map(Denom.burger -> 2)))
    assertEquals(burgers(Addr.king, Denom.banana), 0)
    assertEquals(held(model.init).set(Addr.king, Denom.banana, 0), burgers)
    assertEquals(burgers.set(Addr.king, Denom.burger, 0), empty)
    assertEquals(empty.set(Addr.mario, Denom.banana, 0), empty)

  test("Bank ordinary transfer debits and credits with sparse cleanup"):
    import Bank.*
    val first = run(model, model.init, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1)))
    val paid = bank(
      Balances[Addr](
        Map(
          Addr.king -> Map(Denom.banana -> 1, Denom.burger -> 2),
          Addr.peaches -> Map(Denom.banana -> 1)
        )
      )
    )
    assertEquals(first, List(Step(Outcome.accepted, paid)))
    val second = run(model, paid, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1)))
    assertEquals(
      second,
      List(
        Step(
          Outcome.accepted,
          bank(
            Balances[Addr](
              Map(Addr.king -> Map(Denom.burger -> 2), Addr.peaches -> Map(Denom.banana -> 2))
            )
          )
        )
      )
    )
    val last = bank(Balances[Addr](Map(Addr.mario -> Map(Denom.banana -> 1))))
    assertEquals(
      run(model, last, client.send(Send(Addr.mario, Addr.peaches, Denom.banana, 1))),
      List(Step(
        Outcome.accepted,
        bank(Balances[Addr](Map(Addr.peaches -> Map(Denom.banana -> 1))))
      ))
    )

  test("Bank zero sends preserve funded acceptance and empty refusal including self"):
    import Bank.*
    for to <- List(Addr.king, Addr.peaches) do
      assertEquals(
        run(model, model.init, client.send(Send(Addr.king, to, Denom.banana, 0))),
        List(Step(Outcome.accepted, model.init))
      )
    for to <- List(Addr.mario, Addr.king) do
      assertEquals(
        run(model, model.init, client.send(Send(Addr.mario, to, Denom.banana, 0))),
        List(Step(Outcome.invalidCoinsOrInsufficientFunds, model.init))
      )

  test("Bank insufficient funds precede self transfer and receiver overflow"):
    import Bank.*
    assertEquals(
      run(model, model.init, client.send(Send(Addr.king, Addr.king, Denom.banana, 2))),
      List(Step(Outcome.accepted, model.init))
    )
    for to <- List(Addr.king, Addr.peaches) do
      assertEquals(
        run(model, model.init, client.send(Send(Addr.king, to, Denom.banana, 3))),
        List(Step(Outcome.invalidCoinsOrInsufficientFunds, model.init))
      )
    assertEquals(
      run(Leaky, model.init, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 3))),
      List(Step(Outcome.overflow, model.init))
    )
    assertEquals(
      run(Leaky, model.init, client.send(Send(Addr.mario, Addr.mario, Denom.banana, 3))),
      List(Step(Outcome.accepted, model.init))
    )
    val fullReceiver = bank(
      Balances[Addr](Map(
        Addr.king -> Map(Denom.banana -> 2),
        Addr.peaches -> Map(Denom.banana -> 2)
      ))
    )
    assertEquals(
      run(model, fullReceiver, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))),
      List(Step(Outcome.overflow, fullReceiver))
    )

  test("Bank negative sends pull coins and leave debts within the balance range"):
    import Bank.*
    def send(state: State, from: Addr, to: Addr, amount: Int) =
      run(model, state, client.send(Send(from, to, Denom.banana, amount)))
    val split = bank(
      Balances[Addr](
        Map(
          Addr.king -> Map(Denom.banana -> 1, Denom.burger -> 2),
          Addr.peaches -> Map(Denom.banana -> 1)
        )
      )
    )
    // a negative send pulls the coins back from the receiver
    assertEquals(send(split, Addr.king, Addr.peaches, -1), List(Step(Outcome.accepted, model.init)))
    // from a receiver with none, it leaves a debt
    val owing = bank(
      Balances[Addr](
        Map(
          Addr.king -> Map(Denom.banana -> 2, Denom.burger -> 2),
          Addr.peaches -> Map(Denom.banana -> 1),
          Addr.mario -> Map(Denom.banana -> -1)
        )
      )
    )
    assertEquals(send(split, Addr.king, Addr.mario, -1), List(Step(Outcome.accepted, owing)))
    // a debtor holds an entry, so it may send what it owes on, and settles at zero
    assertEquals(
      send(owing, Addr.mario, Addr.donkeykong, -1),
      List(
        Step(
          Outcome.accepted,
          bank(
            Balances[Addr](
              Map(
                Addr.king -> Map(Denom.banana -> 2, Denom.burger -> 2),
                Addr.peaches -> Map(Denom.banana -> 1),
                Addr.donkeykong -> Map(Denom.banana -> -1)
              )
            )
          )
        )
      )
    )
    // a sender with no entry is refused whatever the amount
    assertEquals(
      send(model.init, Addr.mario, Addr.king, -1),
      List(Step(Outcome.invalidCoinsOrInsufficientFunds, model.init))
    )
    // the king already holds the supply, so pulling one more overflows the sender
    assertEquals(
      send(model.init, Addr.king, Addr.peaches, -1),
      List(Step(Outcome.overflow, model.init))
    )

  test("Bank Properties distinguish an ordinary transfer from credit without debit"):
    import Bank.*
    val cls = client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))
    val paid = run(model, model.init, cls).head
    val inflated = run(Leaky, model.init, cls).head
    assertEquals(
      inflated,
      Step(
        Outcome.accepted,
        bank(
          Balances[Addr](
            Map(
              Addr.king -> Map(Denom.banana -> 2, Denom.burger -> 2),
              Addr.peaches -> Map(Denom.banana -> 1)
            )
          )
        )
      )
    )
    def supply(step: Step[State, Outcome, Nothing]) =
      SampleClaims.judged(model, model, "invariants", "totalSupplyInv", step).get
    val leakyTotalSupplyInv = invariant(check(totalSupply(model.balances)))
    def leaky(step: Step[State, Outcome, Nothing]) =
      SampleClaims.holds(Leaky, leakyTotalSupplyInv, step.state)
    assertEquals(supply(paid), true)
    assertEquals(supply(inflated), false)
    assertEquals(leaky(paid), true)
    assertEquals(leaky(inflated), false)
    def refuses(before: State, step: Step[State, Outcome, Nothing]) =
      SampleClaims
        .judged(
          model,
          model,
          "postconditions",
          "failedSendChangesNothing",
          step,
          Some(before),
          Some(cls)
        )
        .get
    assertEquals(refuses(model.init, Step(Outcome.overflow, model.init)), true)
    assertEquals(refuses(model.init, Step(Outcome.overflow, paid.state)), false)

  test("Bank record inputs preserve ascending amount order and refuse outside-domain classes"):
    import Bank.*
    val classes = classesOf(client.send.decl)
    assertEquals(classes.size, 160)
    assertEquals(
      classes.take(6),
      List(
        List(Addr.king, Addr.king, Denom.banana, -1),
        List(Addr.king, Addr.king, Denom.banana, 0),
        List(Addr.king, Addr.king, Denom.banana, 1),
        List(Addr.king, Addr.king, Denom.banana, 2),
        List(Addr.king, Addr.king, Denom.banana, 3),
        List(Addr.king, Addr.king, Denom.burger, -1)
      )
    )
    assertEquals(classes.last, List(Addr.mario, Addr.mario, Denom.burger, 3))
    intercept[IllegalArgumentException](
      client.send(Send(Addr.king, Addr.peaches, Denom.banana, -2))
    ): Unit
    intercept[IllegalArgumentException](
      client.send(Send(Addr.king, Addr.peaches, Denom.banana, 4))
    ): Unit

  test("Bank's witness is found and Leaky's total supply is refuted"):
    import Bank.*
    import NativeCheck.Verdict.*
    // peachesGetsBananas is a find's target, not a promise of every such send: the find answers
    // found, and verified as a promise it is refuted
    assertEquals(
      NativeCheck.queries(Questions).map(r => r.subject -> r.verdict),
      Vector("questions.kingSendsPeachesBananas" -> found)
    )
    assertEquals(
      NativeCheck
        .query(verify(Questions.queries.peachesGetsBananas), "questions.peachesGetsBananas")
        .verdict,
      refuted
    )
    val leakyTotalSupplyInv = invariant(check(totalSupply(model.balances)))
    // Leaky never debits, so with negative amounts every balance ranges over the whole balance
    // range: a counterexample one send deep stands without exploring all of it
    val leak = assertRefuted(Leaky, leakyTotalSupplyInv, Seq.empty[State], 1024)
    assertEquals(leak.subject, "leaky.leakyTotalSupplyInv")
    assertEquals(leak.detail.head, "counterexample: send(king, donkeykong, banana, -1)")
    assertEquals(NativeCheck.queries(Leaky), Vector.empty)

object BankFixture:
  enum Addr:
    case king, donkeykong, peaches, mario

  val model: Bank.Model[Addr] = Bank.Model(holder = Addr.king, supply = 2)

  object Questions extends Derived(model):
    import model.{balances, client}
    import Bank.{Denom, Send}

    object queries:
      // A send of one banana from the king to peaches: not every such send pays, so it is a find's
      // target, not a promise.
      val peachesGetsBananas =
        after(
          client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))
        ):
          check(balances(Addr.king, Denom.banana) == 1)
          check(balances(Addr.peaches, Denom.banana) == 1)
      val kingSendsPeachesBananas =
        find(peachesGetsBananas)

object BankControls:
  import samples.Bank.*

  val _ = () => summon[CanEqual[Outcome, Outcome]]

  object effects:
    // A send that credits the receiver without debiting the sender: the coins it pays come from
    // nowhere, and the total supply grows.
    def sendFromNowhere(in: Send[Addr]) =
      val toBalance = model.balances(in.toAddr, in.denom)
      if in.fromAddr == in.toAddr then ()
      else if !fits(toBalance + in.amount) then model.stateVars.get.refuse(Outcome.overflow)
      else model.balances = model.balances.set(in.toAddr, in.denom, toBalance + in.amount)

  // A send that pays from nowhere: the negative control of the total supply, which its test asserts
  // refuted.
  object Leaky
      extends Derived(
        model.rebind(
          on(client.send): in =>
            effects.sendFromNowhere(in)
        )
      )
