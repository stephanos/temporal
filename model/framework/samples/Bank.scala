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
  final case class Coins(banana: UpTo[2], burger: UpTo[2])

  // the balances of every address
  final case class Balances(king: Coins, donkeykong: Coins, peaches: Coins, mario: Coins)

  def coinsOf(balances: Balances, addr: Addr): Coins = addr match
    case Addr.king       => balances.king
    case Addr.donkeykong => balances.donkeykong
    case Addr.peaches    => balances.peaches
    case Addr.mario      => balances.mario

  def amountOf(coins: Coins, denom: Denom): Int = denom match
    case Denom.banana => coins.banana
    case Denom.burger => coins.burger

  def balanceOf(balances: Balances, addr: Addr, denom: Denom): Int =
    amountOf(coinsOf(balances, addr), denom)

  def withAmount(coins: Coins, denom: Denom, amount: Int): Coins = denom match
    case Denom.banana => coins.copy(banana = UpTo(amount))
    case Denom.burger => coins.copy(burger = UpTo(amount))

  def withBalance(balances: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
    addr match
      case Addr.king       => balances.copy(king = withAmount(balances.king, denom, amount))
      case Addr.donkeykong =>
        balances.copy(donkeykong = withAmount(balances.donkeykong, denom, amount))
      case Addr.peaches => balances.copy(peaches = withAmount(balances.peaches, denom, amount))
      case Addr.mario   => balances.copy(mario = withAmount(balances.mario, denom, amount))

  // sum up amounts over all balances for a given denomination
  def sumForDenom(balances: Balances, denom: Denom): Int =
    balanceOf(balances, Addr.king, denom) + balanceOf(balances, Addr.donkeykong, denom) +
      balanceOf(balances, Addr.peaches, denom) + balanceOf(balances, Addr.mario, denom)

  // the answer of a send, in place of the last error message
  enum Outcome:
    case accepted, invalidCoinsOrInsufficientFunds, overflow

  given Ok[Outcome] = Ok(Outcome.accepted)

  object client extends Actor:
    // an amount is zero up to one more than the supply, so that a send may ask too much
    val send = action(this)
      .input[Addr]("fromAddr")
      .input[Addr]("toAddr")
      .input[Denom]("denom")
      .input[UpTo[3]]("amount")

  object Model extends Machine[Balances, Outcome, Nothing]:
    // the king has it all
    val init = Balances(
      king = Coins(banana = UpTo(2), burger = UpTo(2)),
      donkeykong = Coins(banana = UpTo(0), burger = UpTo(0)),
      peaches = Coins(banana = UpTo(0), burger = UpTo(0)),
      mario = Coins(banana = UpTo(0), burger = UpTo(0))
    )
    // a run may stop between any two sends
    def end(balances: Balances) = true

    object effects:
      // Transfers `amount` coins of `denom` from `fromAddr` to `toAddr`. It fails where the sender
      // holds none of `denom` or fewer than `amount`, and where the receiver would hold more than
      // the type of a balance can.
      def send(
          balances: Balances,
          fromAddr: Addr,
          toAddr: Addr,
          denom: Denom,
          amount: UpTo[3]
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
        val fromBalance = balanceOf(balances, fromAddr, denom)
        val toBalance = balanceOf(balances, toAddr, denom)
        if fromBalance == 0 || fromBalance < amount then
          reject(Outcome.invalidCoinsOrInsufficientFunds, balances)
        else if fromAddr == toAddr then stay(balances)
        else if toBalance + amount > supply then reject(Outcome.overflow, balances)
        else
          enter(
            withBalance(
              withBalance(balances, fromAddr, denom, fromBalance - amount),
              toAddr,
              denom,
              toBalance + amount
            )
          )

      // A send that credits the receiver without debiting the sender: the coins it pays come from
      // nowhere, and the total supply grows.
      def sendFromNowhere(
          balances: Balances,
          fromAddr: Addr,
          toAddr: Addr,
          denom: Denom,
          amount: UpTo[3]
      ): List[Step[Balances, Bank.Outcome, Nothing]] =
        val toBalance = balanceOf(balances, toAddr, denom)
        if fromAddr == toAddr then stay(balances)
        else if toBalance + amount > supply then reject(Outcome.overflow, balances)
        else enter(withBalance(balances, toAddr, denom, toBalance + amount))

    object rules extends Rules:
      on(client.send):
        always ~> effects.send

  object Leaky
      extends Derived(Model.rebind(client.send ~> Model.effects.sendFromNowhere)),
        NegativeControl

  // the total supply of the network is equal to the sum of all coins from the accounts
  def totalSupply(balances: Balances): Boolean =
    sumForDenom(balances, Denom.banana) == supply && sumForDenom(balances, Denom.burger) == supply

  val totalSupplyInv: Property[Balances] = Model.property.expect:
    (after: Expectation[Balances, Outcome, Nothing]) ?=> check(totalSupply(after.step.state))
  val leakyTotalSupplyInv: Property[Balances] =
    Leaky.property.expect: (after: Expectation[Balances, Outcome, Nothing]) ?=>
      check(totalSupply(after.step.state))
  val failedSendChangesNothing: Property[Balances] = Model.property.across.expect:
    (before: Before[Balances], after: Expectation[Balances, Outcome, Nothing]) ?=>
      if after.step.outcome != Outcome.accepted then check(after.step.state == before.state)
  val peachesGetsBananas: Property[Balances] =
    Model.property
      .after(client.send(Addr.king, Addr.peaches, Denom.banana, UpTo(1)))
      .expect: (after: Expectation[Balances, Outcome, Nothing]) ?=>
        check(balanceOf(after.step.state, Addr.king, Denom.banana) == 1)
        check(balanceOf(after.step.state, Addr.peaches, Denom.banana) == 1)

  val anySend: Scenario[Balances] = Model.scenario.starts(Model.init).free
  val anyLeakySend: Scenario[Balances] = Leaky.scenario.starts(Model.init).free
  // every distribution of the supply is at most four sends from the start
  val short = Limits(steps = 4, actions = 4, search = 65536)
  val queries = Vector(
    query("totalSupplyHolds") verify totalSupplyInv in anySend limits short,
    query("failedSendChangesNothingHolds") verify failedSendChangesNothing in anySend limits short,
    query("kingSendsPeachesBananas") find peachesGetsBananas in anySend limits short,
    query("refutesLeakyTotalSupply") verify leakyTotalSupplyInv in anyLeakySend limits short
  )
