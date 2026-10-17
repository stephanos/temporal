package framework.dsl

import BankControls.Leaky
import framework.dsl.Finite.given
import samples.Bank

class BankTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[Bank.Outcome, Bank.Outcome]]

  test("Bank's four native predicates retain supply, refusal and exact-send requirements"):
    import Bank.*
    import NativeControls.invariant
    // a Bank state by its one var, and the balances a state holds
    def bank(balances: Balances): State = Model.init.copy(balances = balances)
    def held(state: State): Balances = Model.at(state)(Model.balances)
    val initial = Model.init
    val start = held(initial)
    val paid = bank(
      withBalance(withBalance(start, Addr.king, Denom.banana, 1), Addr.peaches, Denom.banana, 1)
    )
    val inflated = bank(withBalance(start, Addr.peaches, Denom.banana, 1))
    val supplyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, inflated), false)
    )
    val leakyTotalSupplyInv = invariant(check(totalSupply(Model.balances)))
    supplyCases.foreach: (after, expected) =>
      assertEquals(totalSupply(held(after.state)), expected)
      assertEquals(
        SampleClaims.judged(Model, Model, "invariants", "totalSupplyInv", after),
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
          Model,
          Model,
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
        Model,
        Model,
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
      (Step(Outcome.accepted, bank(withBalance(start, Addr.king, Denom.banana, 1))), false)
    ).foreach: (after, expected) =>
      assertEquals(
        balanceOf(held(after.state), Addr.king, Denom.banana) == 1 &&
          balanceOf(held(after.state), Addr.peaches, Denom.banana) == 1,
        expected
      )
      assertEquals(exact(after, send), Some(expected))
      assertEquals(exact(after, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 2))), None)

object BankControls:
  import samples.Bank.*

  val _ = () => summon[CanEqual[Outcome, Outcome]]

  object effects:
    // A send that credits the receiver without debiting the sender: the coins it pays come from
    // nowhere, and the total supply grows.
    def sendFromNowhere(in: Send) =
      val toBalance = balanceOf(Model.balances, in.toAddr, in.denom)
      if in.fromAddr == in.toAddr then ()
      else if toBalance + in.amount > supply then Model.stateVars.get.refuse(Outcome.overflow)
      else Model.balances = withBalance(Model.balances, in.toAddr, in.denom, toBalance + in.amount)

  // A send that pays from nowhere: the negative control of the total supply, which its test asserts
  // refuted.
  object Leaky
      extends Derived(
        Model.rebind(
          on(client.send): s =>
            effects.sendFromNowhere(s)
        )
      )
