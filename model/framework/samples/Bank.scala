package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/cosmos/bank/bank.qnt and bankTest.qnt
//
// The send logic of a bank module: accounts hold coins of several denominations, and a send moves
// coins of one denomination from one account to another, or fails and changes nothing. The total
// supply of each denomination is fixed, so the coins in all accounts always add up to it.
object Bank:
  // all addresses we would like to work with
  enum Addr:
    case king, donkeykong, peaches, mario

  // all denominations
  enum Denom:
    case banana, burger

  // the supply of each denomination, and so the most coins one account can hold
  val supply = 2

  // the coins of one account, per denomination; zero coins is no balance at all
  type Coins = Map[Denom, Int]

  // the balances of every address
  final case class Balances(of: Map[Addr, Coins])

  def coinsOf(balances: Balances, addr: Addr): Coins =
    balances.of.getOrElse(addr, Map.empty)

  def amountOf(coins: Coins, denom: Denom): Int = coins.getOrElse(denom, 0)

  def balanceOf(balances: Balances, addr: Addr, denom: Denom): Int =
    amountOf(coinsOf(balances, addr), denom)

  def withAmount(coins: Coins, denom: Denom, amount: Int): Coins =
    if amount == 0 then coins.removed(denom) else coins.updated(denom, amount)

  def withBalance(balances: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
    val coins = withAmount(coinsOf(balances, addr), denom, amount)
    balances.copy(
      of = if coins.isEmpty then balances.of.removed(addr) else balances.of.updated(addr, coins)
    )

  // sum up amounts over all balances for a given denomination
  def sumForDenom(balances: Balances, denom: Denom): Int =
    balanceOf(balances, Addr.king, denom) + balanceOf(balances, Addr.donkeykong, denom) +
      balanceOf(balances, Addr.peaches, denom) + balanceOf(balances, Addr.mario, denom)

  // the answer of a send, in place of the last error message
  enum Outcome:
    case accepted, invalidCoinsOrInsufficientFunds, overflow

  given Ok[Outcome] = Ok(Outcome.accepted)

  final case class Send(fromAddr: Addr, toAddr: Addr, denom: Denom, amount: Int)

  object client extends Actor:
    // an amount is zero up to one more than the supply, so that a send may ask too much
    val send = action(this).inputs[Send](inputField[Send](_.amount).in(domains.oneOf(0 to 3)))

  // the total supply of the network is equal to the sum of all coins from the accounts
  def totalSupply(balances: Balances): Boolean =
    sumForDenom(balances, Denom.banana) == supply && sumForDenom(balances, Denom.burger) == supply

  object Model extends RecordMachine[Balances, Outcome, Nothing]:
    // the king has it all
    val init = Balances(
      of = Map(Addr.king -> Map(Denom.banana -> 2, Denom.burger -> 2))
    )
    // a run may stop between any two sends
    def end(balances: Balances) = true

    object rules extends Rules:
      on(client.send):
        always ~> effects.send

    object effects:
      // Transfers `amount` coins of `denom` from `fromAddr` to `toAddr`. It fails where the sender
      // holds none of `denom` or fewer than `amount`, and where the receiver would hold more than
      // the type of a balance can.
      def send(
          balances: Balances,
          in: Send
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
        val fromBalance = balanceOf(balances, in.fromAddr, in.denom)
        val toBalance = balanceOf(balances, in.toAddr, in.denom)
        if fromBalance == 0 || fromBalance < in.amount then
          reject(Outcome.invalidCoinsOrInsufficientFunds, balances)
        else if in.fromAddr == in.toAddr then stay(balances)
        else if toBalance + in.amount > supply then reject(Outcome.overflow, balances)
        else
          enter(
            withBalance(
              withBalance(balances, in.fromAddr, in.denom, fromBalance - in.amount),
              in.toAddr,
              in.denom,
              toBalance + in.amount
            )
          )

      // A send that credits the receiver without debiting the sender: the coins it pays come from
      // nowhere, and the total supply grows.
      def sendFromNowhere(
          balances: Balances,
          in: Send
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
        val toBalance = balanceOf(balances, in.toAddr, in.denom)
        if in.fromAddr == in.toAddr then stay(balances)
        else if toBalance + in.amount > supply then reject(Outcome.overflow, balances)
        else enter(withBalance(balances, in.toAddr, in.denom, toBalance + in.amount))

    object invariants:
      val totalSupplyInv: Property = property.expect:
        (after: Expectation[Balances, Bank.Outcome, Nothing]) ?=>
          check(totalSupply(after.step.state))

    object postconditions:
      val failedSendChangesNothing: Property = property.across.expect:
        (before: Before[Balances], after: Expectation[Balances, Bank.Outcome, Nothing]) ?=>
          if after.step.outcome != Outcome.accepted then check(after.step.state == before.state)
      val peachesGetsBananas: Property =
        property
          .after(
            client.send(
              fromAddr = Addr.king,
              toAddr = Addr.peaches,
              denom = Denom.banana,
              amount = 1
            )
          )
          .expect: (after: Expectation[Balances, Bank.Outcome, Nothing]) ?=>
            check(balanceOf(after.step.state, Addr.king, Denom.banana) == 1)
            check(balanceOf(after.step.state, Addr.peaches, Denom.banana) == 1)

    object queries:
      val anySend: Scenario = scenario.starts(init).free
      // every distribution of the supply is at most four sends from the start
      val short = Limits(steps = 4, actions = 4, search = 65536)
      val totalSupplyHolds = query verify invariants.totalSupplyInv in anySend limits short
      val failedSendChangesNothingHolds =
        query verify postconditions.failedSendChangesNothing in anySend limits short
      val kingSendsPeachesBananas =
        query find postconditions.peachesGetsBananas in anySend limits short

  // A send that pays from nowhere: the negative control of the total supply.
  object Leaky
      extends Derived(Model.rebind(client.send ~> Model.effects.sendFromNowhere)),
        NegativeControl:

    object invariants:
      val leakyTotalSupplyInv: Property = property.expect:
        (after: Expectation[Balances, Bank.Outcome, Nothing]) ?=>
          check(totalSupply(after.step.state))

    object queries:
      val anyLeakySend: Scenario = scenario.starts(Model.init).free
      val refutesLeakyTotalSupply =
        query verify invariants.leakyTotalSupplyInv in anyLeakySend limits Model.queries.short
