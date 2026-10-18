package framework.prototype

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.Verdict
import samples.Bank
import samples.Bank.{Balances, Denom, Send, Outcome}

enum Addr:
  case king, donkeykong, peaches, mario

@scala.annotation.nowarn("msg=unused value of type .*Boolean.*")
class BankTest extends ModelSuite(new Bank.Model[Addr](holder = Addr.king, supply = 2)):
  import model.balances

  object client:
    def send(fromAddr: Addr, toAddr: Addr, denom: Denom, amount: Int): Unit =
      step(model.client.send(Send(fromAddr = fromAddr, toAddr = toAddr, denom = denom, amount = amount)))

  object examples:
    test("oneOf enumerates named record calls"):
      oneOf(
        model.client.send(Send(fromAddr = Addr.king, toAddr = Addr.donkeykong, denom = Denom.banana, amount = 1)),
        model.client.send(Send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1))
      )
      expect:
        balances(Addr.king, Denom.banana) == 1
        balances(Addr.king, Denom.burger) == 2

  object explorations:
    val anySends = explore(steps = 6)(any(model.client.send))
    test("overflow find within four sends"):
      anySends.find(steps = 4):
        expect:
          outcome == Outcome.overflow
    test("seeded sampling is reproducible"):
      val first = anySends.sample(count = 20, seed = 1)
      val second = anySends.sample(count = 20, seed = 1)
      assert(first.sampling.get.walks.forall(_.steps.size == 6))
      assertEquals(first.sampling.get.walks, second.sampling.get.walks)
    test("Leaky refutes the supply invariant"):
      val bank = model
      object effects:
        def mint(in: Send[Addr]): Unit =
          if in.fromAddr != in.toAddr then
            bank.balances = bank.balances.set(in.toAddr, in.denom, bank.balances(in.toAddr, in.denom) + in.amount)
      object Leaky extends Derived(bank.rebind(
        on(bank.client.send): in =>
          effects.mint(in)
      ))
      val result = refuted(Leaky, bank.invariants.totalSupplyInv)
      assertEquals(result.verdict, Verdict.refuted, result.toString)
      assert(result.path.steps.nonEmpty)
      println(result)

  object conformance:
    test("peaches gets a banana", expectation = "satisfied", through = Some("prototype-carrier")):
      client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)
      expect:
        balances(Addr.peaches, Denom.banana) == 1
        outcome == Outcome.accepted
    test("sampled paths", expectation = "conformant"):
      explorations.anySends.sample(count = 2, seed = 17)

  object unit:
    test("Balances.set removes zero"):
      assertEquals(Balances(Map.empty[Addr, Bank.Coins]).set(Addr.king, Denom.banana, 0), Balances(Map.empty[Addr, Bank.Coins]))
    test("fresh stable Bank instances own independent state"):
      val other = new Bank.Model[Addr](holder = Addr.peaches, supply = 2)
      assert(!(other eq model))
      assertEquals(other.balances(Addr.peaches, Denom.banana), 2)
      assertEquals(model.balances(Addr.peaches, Denom.banana), 0)

@scala.annotation.nowarn("msg=unused value of type .*Boolean.*")
class QuintBankTest extends ModelSuite(new Bank.Model[Addr](holder = Addr.king, supply = 10000)):
  import model.balances
  def before: State { val balances: Balances[Addr] } =
    NativeClaims.start(model.init).asInstanceOf[State { val balances: Balances[Addr] }]
  object examples:
    test("literal Quint sendTest"):
      step(model.client.send(Send(fromAddr = Addr.king, toAddr = Addr.donkeykong, denom = Denom.banana, amount = 2000)))
      expect:
        balances(Addr.king, Denom.banana) == 8000
        balances(Addr.donkeykong, Denom.banana) == 2000
        balances(Addr.king, Denom.burger) == 10000
        balances.of(Addr.donkeykong).keySet == Set(Denom.banana)
      step(model.client.send(Send(fromAddr = Addr.donkeykong, toAddr = Addr.peaches, denom = Denom.burger, amount = 1000)))
      expect:
        outcome == Outcome.invalidCoinsOrInsufficientFunds
        balances == before.balances

@scala.annotation.nowarn("msg=unused value of type .*Boolean.*")
class WrongExpectationProof extends ModelSuite(new Bank.Model[Addr](holder = Addr.king, supply = 2)):
  object examples:
    test("deliberately wrong first Boolean"):
      step(model.client.send(Send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)))
      expect:
        model.balances(Addr.peaches, Denom.banana) == 99
        model.balances(Addr.king, Denom.banana) == 1
