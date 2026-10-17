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

  // The total coins of each denomination. Scaling the supply to two keeps the sample small;
  // the same magnitude bounds a signed balance in place of the SDK's Int256 range.
  val supply = 2

  // Whether a signed balance fits the sample's range. A negative send can increase the sender's
  // balance or leave the receiver owing coins, so both resulting balances must fit.
  def fits(amount: Int): Boolean =
    -supply <= amount && amount <= supply

  // the coins of one account, per denomination; zero coins is no balance at all
  type Coins = Map[Denom, Int]

  // the balances of every address
  final case class Balances(of: Map[Addr, Coins]):
    // the coins of `denom` that `addr` holds
    def apply(addr: Addr, denom: Denom): Int =
      of.getOrElse(addr, Map.empty).getOrElse(denom, 0)

    // `addr` holding `amount` coins of `denom`. Neither a zero amount nor an empty account keeps an
    // entry, so equal balances are one state. Quint's receiver keeps a zero it is sent, against
    // its own comment that x/bank invariants prohibit zero balances; this follows the comment.
    def set(addr: Addr, denom: Denom, amount: Int): Balances =
      val coins = of.getOrElse(addr, Map.empty)
      val next = if amount == 0 then coins.removed(denom) else coins.updated(denom, amount)
      Balances(if next.isEmpty then of.removed(addr) else of.updated(addr, next))

    // sum up amounts over all balances for a given denomination
    def total(denom: Denom): Int =
      Addr.values.map(addr => this(addr, denom)).sum

  // the answer of a send, in place of the last error message
  enum Outcome:
    case accepted, invalidCoinsOrInsufficientFunds, overflow

  given Ok[Outcome] = Ok(Outcome.accepted)

  final case class Send(fromAddr: Addr, toAddr: Addr, denom: Denom, amount: Int)

  object client extends Actor:
    // an amount is zero up to one more than the supply, so that a send may ask too much. It is
    // also negative, as Quint's test sends from -10: SendCoins itself does not require a positive
    // amount, only coins built by NewCoins are.
    val send = action.inputs[Send](inputField[Send](_.amount).in(domains.oneOf(-1 to supply + 1)))

  // the total supply of the network is equal to the sum of all coins from the accounts
  def totalSupply(balances: Balances): Boolean =
    Denom.values.forall(balances.total(_) == supply)

  object Model extends Machine[Outcome, Nothing]:
    // the balances of every address: the king has it all
    var balances = Balances(
      of = Map(Addr.king -> Map(Denom.banana -> supply, Denom.burger -> supply))
    )

    object transitions extends Transitions:
      on(client.send): in =>
        effects.send(in)

    object effects:
      // Transfers `amount` coins of `denom` from `fromAddr` to `toAddr`. It fails where the sender
      // holds none of `denom` or fewer than `amount`, and where either account would hold more, or
      // owe more, than the type of a balance can.
      def send(in: Send) =
        import in.*, Outcome.*
        val sender = balances(fromAddr, denom)
        val receiver = balances(toAddr, denom)
        // Zero is no entry, and the SDK refuses a denomination the sender has no entry for,
        // whatever the amount: so an empty sender's zero send is refused and a funded one's is not.
        if sender == 0 || sender < amount then reject(invalidCoinsOrInsufficientFunds)
        // The sender can pay, and pays itself: accepted, and nothing changes. Quint checks the
        // overflow first; with a bound as small as the supply, the king's self-send of one banana
        // would then overflow.
        else if fromAddr == toAddr then ()
        else if !fits(sender - amount) || !fits(receiver + amount) then reject(overflow)
        else
          balances = balances
            .set(fromAddr, denom, sender - amount)
            .set(toAddr, denom, receiver + amount)

    object invariants:
      val totalSupplyInv =
        check(totalSupply(balances))

    object postconditions:
      val failedSendChangesNothing =
        if outcome != Outcome.accepted then check(balances == before.balances)

    object queries:
      // a send of one banana from the king to peaches: not every such send pays, so it is a find's
      // target, not a promise
      val peachesGetsBananas =
        after(
          client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)
        ):
          check(balances(Addr.king, Denom.banana) == 1)
          check(balances(Addr.peaches, Denom.banana) == 1)
      val kingSendsPeachesBananas =
        find(peachesGetsBananas)
